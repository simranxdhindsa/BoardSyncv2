package sync

import (
	"fmt"
	"time"

	"asana-youtrack-sync/cache"
	configpkg "asana-youtrack-sync/config"
	"asana-youtrack-sync/database"
)

// Service handles sync operations with user authentication
type Service struct {
	db              *database.DB
	configService   *configpkg.Service
	rollbackService *RollbackService
	wsManager       *WebSocketManager
	cache           cache.Cache
}

// NewService creates a new sync service
func NewService(db *database.DB, configService *configpkg.Service, rollbackService *RollbackService, wsManager *WebSocketManager, cacheInstance cache.Cache) *Service {
	return &Service{
		db:              db,
		configService:   configService,
		rollbackService: rollbackService,
		wsManager:       wsManager,
		cache:           cacheInstance,
	}
}

// SyncRequest represents a sync request
type SyncRequest struct {
	Type      string                 `json:"type"`      // "asana_to_youtrack", "youtrack_to_asana", "bidirectional"
	Direction string                 `json:"direction"` // "one_way", "two_way"
	Options   map[string]interface{} `json:"options"`
}

// SyncResult represents the result of a sync operation
type SyncResult struct {
	OperationID   int            `json:"operation_id"`
	Status        string         `json:"status"`
	SyncedItems   int            `json:"synced_items"`
	CreatedItems  []CreatedItem  `json:"created_items"`
	ModifiedItems []ModifiedItem `json:"modified_items"`
	Errors        []string       `json:"errors,omitempty"`
	RollbackData  *RollbackData  `json:"rollback_data,omitempty"`
}

// StartSync initiates a sync operation for a user
func (s *Service) StartSync(userID int, request SyncRequest) (*SyncResult, error) {
	// Get user settings
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user settings: %w", err)
	}

	// Validate settings
	if err := s.validateSettings(settings, request.Type); err != nil {
		return nil, fmt.Errorf("invalid settings: %w", err)
	}

	// Create operation record
	operation, err := s.rollbackService.CreateOperation(userID, request.Type, map[string]interface{}{
		"direction": request.Direction,
		"options":   request.Options,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create operation: %w", err)
	}

	// Start sync in background
	go s.performSync(userID, operation, settings, request)

	return &SyncResult{
		OperationID: operation.ID,
		Status:      StatusPending,
	}, nil
}

// performSync executes the actual sync operation
func (s *Service) performSync(userID int, operation *SyncOperation, settings *configpkg.UserSettings, request SyncRequest) {
	// Update status to in progress
	s.rollbackService.UpdateOperationStatus(operation.ID, StatusInProgress, nil)

	// Notify start
	s.wsManager.SendToUser(userID, MsgTypeSyncStart, map[string]interface{}{
		"operation_id": operation.ID,
		"type":         operation.OperationType,
	})

	var result SyncResult
	var rollbackData RollbackData

	switch request.Type {
	case OpTypeAsanaToYouTrack:
		result = s.syncAsanaToYouTrack(userID, operation.ID, settings, request.Options, &rollbackData)
	case OpTypeYouTrackToAsana:
		result = s.syncYouTrackToAsana(userID, operation.ID, settings, request.Options, &rollbackData)
	case OpTypeBidirectional:
		result = s.syncBidirectional(userID, operation.ID, settings, request.Options, &rollbackData)
	default:
		errMsg := "unsupported sync type"
		s.rollbackService.UpdateOperationStatus(operation.ID, StatusFailed, &errMsg)
		s.wsManager.NotifyError(userID, operation.ID, errMsg)
		return
	}

	// Update final status
	if len(result.Errors) > 0 {
		errorMsg := fmt.Sprintf("Sync completed with %d errors", len(result.Errors))
		s.rollbackService.UpdateOperationStatus(operation.ID, StatusCompleted, &errorMsg)
	} else {
		s.rollbackService.UpdateOperationStatus(operation.ID, StatusCompleted, nil)
	}

	// Notify completion
	s.wsManager.NotifyComplete(userID, operation.ID, map[string]interface{}{
		"synced_items":   result.SyncedItems,
		"created_items":  len(result.CreatedItems),
		"modified_items": len(result.ModifiedItems),
		"errors":         result.Errors,
	})
}

// syncAsanaToYouTrack syncs from Asana to YouTrack
func (s *Service) syncAsanaToYouTrack(userID, operationID int, settings *configpkg.UserSettings, options map[string]interface{}, rollbackData *RollbackData) SyncResult {
	var result SyncResult

	s.wsManager.NotifyProgress(userID, operationID, 10, "Fetching Asana tasks...")

	// TODO: Implement actual Asana to YouTrack sync logic
	// This would integrate with your existing sync logic from services.go

	// Placeholder implementation
	time.Sleep(2 * time.Second)
	s.wsManager.NotifyProgress(userID, operationID, 50, "Creating YouTrack issues...")

	time.Sleep(2 * time.Second)
	s.wsManager.NotifyProgress(userID, operationID, 100, "Sync completed")

	result.SyncedItems = 5 // Placeholder
	return result
}

// syncYouTrackToAsana syncs from YouTrack to Asana
func (s *Service) syncYouTrackToAsana(userID, operationID int, settings *configpkg.UserSettings, options map[string]interface{}, rollbackData *RollbackData) SyncResult {
	var result SyncResult

	s.wsManager.NotifyProgress(userID, operationID, 10, "Fetching YouTrack issues...")

	// TODO: Implement actual YouTrack to Asana sync logic

	// Placeholder implementation
	time.Sleep(2 * time.Second)
	s.wsManager.NotifyProgress(userID, operationID, 50, "Creating Asana tasks...")

	time.Sleep(2 * time.Second)
	s.wsManager.NotifyProgress(userID, operationID, 100, "Sync completed")

	result.SyncedItems = 3 // Placeholder
	return result
}

// syncBidirectional performs bidirectional sync
func (s *Service) syncBidirectional(userID, operationID int, settings *configpkg.UserSettings, options map[string]interface{}, rollbackData *RollbackData) SyncResult {
	var result SyncResult

	// First, sync Asana to YouTrack
	s.wsManager.NotifyProgress(userID, operationID, 25, "Syncing Asana to YouTrack...")
	asanaResult := s.syncAsanaToYouTrack(userID, operationID, settings, options, rollbackData)

	// Then, sync YouTrack to Asana
	s.wsManager.NotifyProgress(userID, operationID, 75, "Syncing YouTrack to Asana...")
	youtrackResult := s.syncYouTrackToAsana(userID, operationID, settings, options, rollbackData)

	// Combine results
	result.SyncedItems = asanaResult.SyncedItems + youtrackResult.SyncedItems
	result.CreatedItems = append(asanaResult.CreatedItems, youtrackResult.CreatedItems...)
	result.ModifiedItems = append(asanaResult.ModifiedItems, youtrackResult.ModifiedItems...)
	result.Errors = append(asanaResult.Errors, youtrackResult.Errors...)

	return result
}

// validateSettings validates user settings for sync operation
func (s *Service) validateSettings(settings *configpkg.UserSettings, syncType string) error {
	switch syncType {
	case OpTypeAsanaToYouTrack:
		if settings.AsanaPAT == "" {
			return fmt.Errorf("Asana PAT is required")
		}
		if settings.YouTrackBaseURL == "" || settings.YouTrackToken == "" {
			return fmt.Errorf("YouTrack credentials are required")
		}
		if settings.AsanaProjectID == "" {
			return fmt.Errorf("Asana project ID is required")
		}
		if settings.YouTrackProjectID == "" {
			return fmt.Errorf("YouTrack project ID is required")
		}

	case OpTypeYouTrackToAsana:
		if settings.YouTrackBaseURL == "" || settings.YouTrackToken == "" {
			return fmt.Errorf("YouTrack credentials are required")
		}
		if settings.AsanaPAT == "" {
			return fmt.Errorf("Asana PAT is required")
		}
		if settings.YouTrackProjectID == "" {
			return fmt.Errorf("YouTrack project ID is required")
		}
		if settings.AsanaProjectID == "" {
			return fmt.Errorf("Asana project ID is required")
		}

	case OpTypeBidirectional:
		if settings.AsanaPAT == "" {
			return fmt.Errorf("Asana PAT is required")
		}
		if settings.YouTrackBaseURL == "" || settings.YouTrackToken == "" {
			return fmt.Errorf("YouTrack credentials are required")
		}
		if settings.AsanaProjectID == "" {
			return fmt.Errorf("Asana project ID is required")
		}
		if settings.YouTrackProjectID == "" {
			return fmt.Errorf("YouTrack project ID is required")
		}
	}

	return nil
}

// GetSyncHistory returns sync history for a user
func (s *Service) GetSyncHistory(userID int, limit int) ([]SyncOperation, error) {
	return s.rollbackService.GetUserOperations(userID, limit)
}

// GetSyncStatus returns the status of a specific sync operation
func (s *Service) GetSyncStatus(userID, operationID int) (*SyncOperation, error) {
	operation, err := s.rollbackService.GetOperation(operationID)
	if err != nil {
		return nil, err
	}

	if operation.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	return operation, nil
}

// RollbackSync rolls back a sync operation
func (s *Service) RollbackSync(userID, operationID int) error {
	return s.rollbackService.RollbackOperation(operationID, userID)
}

// CleanupOldOperations removes old sync operations
func (s *Service) CleanupOldOperations(olderThan time.Duration) error {
	return s.rollbackService.DeleteOldOperations(olderThan)
}

// CacheKey generates a cache key for sync-related data
func (s *Service) CacheKey(userID int, prefix string, identifier string) string {
	return fmt.Sprintf("sync:%d:%s:%s", userID, prefix, identifier)
}

// GetCachedData retrieves cached sync data
func (s *Service) GetCachedData(key string, dest interface{}) error {
	return s.cache.Get(key, dest)
}

// SetCachedData stores sync data in cache
func (s *Service) SetCachedData(key string, data interface{}, ttl time.Duration) error {
	return s.cache.Set(key, data, ttl)
}
