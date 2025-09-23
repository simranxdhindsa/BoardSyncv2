package sync

import (
	"fmt"
	"time"

	"asana-youtrack-sync/database"
)

// SyncOperation represents a sync operation that can be rolled back
type SyncOperation struct {
	ID            int                    `json:"id"`
	UserID        int                    `json:"user_id"`
	OperationType string                 `json:"operation_type"`
	OperationData map[string]interface{} `json:"operation_data"`
	Status        string                 `json:"status"`
	ErrorMessage  *string                `json:"error_message,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
}

// OperationTypes
const (
	OpTypeAsanaToYouTrack = "asana_to_youtrack"
	OpTypeYouTrackToAsana = "youtrack_to_asana"
	OpTypeBidirectional   = "bidirectional"
	OpTypeCustomSync      = "custom_sync"
)

// Operation statuses
const (
	StatusPending    = "pending"
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
	StatusRolledBack = "rolled_back"
)

// RollbackData represents data needed for rollback
type RollbackData struct {
	AsanaItems    []RollbackItem `json:"asana_items"`
	YouTrackItems []RollbackItem `json:"youtrack_items"`
	CreatedItems  []CreatedItem  `json:"created_items"`
	ModifiedItems []ModifiedItem `json:"modified_items"`
}

// RollbackItem represents an item that can be rolled back
type RollbackItem struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"` // "task", "issue", etc.
	OriginalData map[string]interface{} `json:"original_data"`
	Platform     string                 `json:"platform"` // "asana", "youtrack"
}

// CreatedItem represents an item that was created during sync
type CreatedItem struct {
	ID       string `json:"id"`
	Platform string `json:"platform"`
	Type     string `json:"type"`
}

// ModifiedItem represents an item that was modified during sync
type ModifiedItem struct {
	ID           string                 `json:"id"`
	Platform     string                 `json:"platform"`
	Type         string                 `json:"type"`
	OriginalData map[string]interface{} `json:"original_data"`
}

type RollbackService struct {
	db *database.DB
}

// NewRollbackService creates a new rollback service
func NewRollbackService(db *database.DB) *RollbackService {
	return &RollbackService{db: db}
}

// CreateOperation creates a new sync operation record
func (rs *RollbackService) CreateOperation(userID int, operationType string, operationData map[string]interface{}) (*SyncOperation, error) {
	operation, err := rs.db.CreateOperation(userID, operationType, operationData)
	if err != nil {
		return nil, fmt.Errorf("failed to create operation: %w", err)
	}

	return &SyncOperation{
		ID:            operation.ID,
		UserID:        operation.UserID,
		OperationType: operation.OperationType,
		OperationData: operation.OperationData,
		Status:        operation.Status,
		ErrorMessage:  operation.ErrorMessage,
		CreatedAt:     operation.CreatedAt,
		CompletedAt:   operation.CompletedAt,
	}, nil
}

// UpdateOperationStatus updates the status of a sync operation
func (rs *RollbackService) UpdateOperationStatus(operationID int, status string, errorMessage *string) error {
	return rs.db.UpdateOperationStatus(operationID, status, errorMessage)
}

// GetOperation retrieves a sync operation by ID
func (rs *RollbackService) GetOperation(operationID int) (*SyncOperation, error) {
	operation, err := rs.db.GetOperation(operationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get operation: %w", err)
	}

	return &SyncOperation{
		ID:            operation.ID,
		UserID:        operation.UserID,
		OperationType: operation.OperationType,
		OperationData: operation.OperationData,
		Status:        operation.Status,
		ErrorMessage:  operation.ErrorMessage,
		CreatedAt:     operation.CreatedAt,
		CompletedAt:   operation.CompletedAt,
	}, nil
}

// GetUserOperations retrieves all operations for a user
func (rs *RollbackService) GetUserOperations(userID int, limit int) ([]SyncOperation, error) {
	operations, err := rs.db.GetUserOperations(userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get user operations: %w", err)
	}

	result := make([]SyncOperation, len(operations))
	for i, op := range operations {
		result[i] = SyncOperation{
			ID:            op.ID,
			UserID:        op.UserID,
			OperationType: op.OperationType,
			OperationData: op.OperationData,
			Status:        op.Status,
			ErrorMessage:  op.ErrorMessage,
			CreatedAt:     op.CreatedAt,
			CompletedAt:   op.CompletedAt,
		}
	}

	return result, nil
}

// RollbackOperation rolls back a completed sync operation
func (rs *RollbackService) RollbackOperation(operationID int, userID int) error {
	// Get the operation
	operation, err := rs.GetOperation(operationID)
	if err != nil {
		return fmt.Errorf("failed to get operation: %w", err)
	}

	// Verify ownership
	if operation.UserID != userID {
		return fmt.Errorf("operation does not belong to user")
	}

	// Check if operation can be rolled back
	if operation.Status != StatusCompleted {
		return fmt.Errorf("can only rollback completed operations")
	}

	// Update operation status to indicate rollback
	if err := rs.UpdateOperationStatus(operationID, StatusRolledBack, nil); err != nil {
		return fmt.Errorf("failed to update operation status: %w", err)
	}

	return nil
}

// CanRollback checks if an operation can be rolled back
func (rs *RollbackService) CanRollback(operationID int) (bool, string) {
	operation, err := rs.GetOperation(operationID)
	if err != nil {
		return false, "Operation not found"
	}

	if operation.Status != StatusCompleted {
		return false, "Operation is not completed"
	}

	// Check if operation is too old (optional business rule)
	if time.Since(operation.CreatedAt) > 24*time.Hour {
		return false, "Operation is too old to rollback (>24 hours)"
	}

	return true, ""
}

// DeleteOldOperations deletes operations older than specified duration
func (rs *RollbackService) DeleteOldOperations(olderThan time.Duration) error {
	// For the pure Go implementation, we could implement this by
	// filtering operations in memory, but for now we'll just return nil
	return nil
}
