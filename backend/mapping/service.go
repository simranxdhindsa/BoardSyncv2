// Create new file: backend/mapping/service.go
package mapping

import (
	"fmt"

	configpkg "asana-youtrack-sync/config"
	"asana-youtrack-sync/database"
	"asana-youtrack-sync/utils"
)

// Service handles ticket mapping operations
type Service struct {
	db            *database.DB
	configService *configpkg.Service
}

// NewService creates a new mapping service
func NewService(db *database.DB, configService *configpkg.Service) *Service {
	return &Service{
		db:            db,
		configService: configService,
	}
}

// CreateMappingRequest represents a request to create a mapping
type CreateMappingRequest struct {
	AsanaURL    string `json:"asana_url" validate:"required"`
	YouTrackURL string `json:"youtrack_url" validate:"required"`
}

// MappingResponse represents a mapping response
type MappingResponse struct {
	ID                int    `json:"id"`
	AsanaTaskID       string `json:"asana_task_id"`
	YouTrackIssueID   string `json:"youtrack_issue_id"`
	AsanaProjectID    string `json:"asana_project_id"`
	YouTrackProjectID string `json:"youtrack_project_id"`
	CreatedAt         string `json:"created_at"`
}

// CreateMapping creates a new ticket mapping from URLs
func (s *Service) CreateMapping(userID int, req CreateMappingRequest) (*MappingResponse, error) {
	// Validate URLs
	if !utils.ValidateAsanaURL(req.AsanaURL) {
		return nil, fmt.Errorf("invalid Asana URL format")
	}

	if !utils.ValidateYouTrackURL(req.YouTrackURL) {
		return nil, fmt.Errorf("invalid YouTrack URL format")
	}

	// Extract task IDs
	asanaTaskID, err := utils.ExtractAsanaTaskID(req.AsanaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract Asana task ID: %w", err)
	}

	youtrackIssueID, err := utils.ExtractYouTrackIssueID(req.YouTrackURL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract YouTrack issue ID: %w", err)
	}

	// Get user settings to validate projects
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user settings: %w", err)
	}

	if settings.AsanaProjectID == "" {
		return nil, fmt.Errorf("Asana project not configured in settings")
	}

	if settings.YouTrackProjectID == "" {
		return nil, fmt.Errorf("YouTrack project not configured in settings")
	}

	// Check if mapping already exists
	if s.db.HasTicketMapping(userID, asanaTaskID, youtrackIssueID) {
		return nil, fmt.Errorf("mapping already exists for these tickets")
	}

	// Create mapping
	mapping, err := s.db.CreateTicketMapping(
		userID,
		settings.AsanaProjectID,
		asanaTaskID,
		settings.YouTrackProjectID,
		youtrackIssueID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create mapping: %w", err)
	}

	return &MappingResponse{
		ID:                mapping.ID,
		AsanaTaskID:       mapping.AsanaTaskID,
		YouTrackIssueID:   mapping.YouTrackIssueID,
		AsanaProjectID:    mapping.AsanaProjectID,
		YouTrackProjectID: mapping.YouTrackProjectID,
		CreatedAt:         mapping.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// GetAllMappings gets all mappings for a user
func (s *Service) GetAllMappings(userID int) ([]MappingResponse, error) {
	mappings, err := s.db.GetAllTicketMappings(userID)
	if err != nil {
		return nil, err
	}

	var response []MappingResponse
	for _, mapping := range mappings {
		response = append(response, MappingResponse{
			ID:                mapping.ID,
			AsanaTaskID:       mapping.AsanaTaskID,
			YouTrackIssueID:   mapping.YouTrackIssueID,
			AsanaProjectID:    mapping.AsanaProjectID,
			YouTrackProjectID: mapping.YouTrackProjectID,
			CreatedAt:         mapping.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return response, nil
}

// DeleteMapping deletes a ticket mapping
func (s *Service) DeleteMapping(userID, mappingID int) error {
	return s.db.DeleteTicketMapping(userID, mappingID)
}

// GetMappingByAsanaID gets mapping by Asana task ID
func (s *Service) GetMappingByAsanaID(userID int, asanaTaskID string) (*MappingResponse, error) {
	mapping, err := s.db.GetTicketMappingByAsanaID(userID, asanaTaskID)
	if err != nil {
		return nil, err
	}

	return &MappingResponse{
		ID:                mapping.ID,
		AsanaTaskID:       mapping.AsanaTaskID,
		YouTrackIssueID:   mapping.YouTrackIssueID,
		AsanaProjectID:    mapping.AsanaProjectID,
		YouTrackProjectID: mapping.YouTrackProjectID,
		CreatedAt:         mapping.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// GetMappingByYouTrackID gets mapping by YouTrack issue ID
func (s *Service) GetMappingByYouTrackID(userID int, youtrackIssueID string) (*MappingResponse, error) {
	mapping, err := s.db.GetTicketMappingByYouTrackID(userID, youtrackIssueID)
	if err != nil {
		return nil, err
	}

	return &MappingResponse{
		ID:                mapping.ID,
		AsanaTaskID:       mapping.AsanaTaskID,
		YouTrackIssueID:   mapping.YouTrackIssueID,
		AsanaProjectID:    mapping.AsanaProjectID,
		YouTrackProjectID: mapping.YouTrackProjectID,
		CreatedAt:         mapping.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}
