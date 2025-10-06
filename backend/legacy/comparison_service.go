// backend/legacy/comparison_service.go - NEW FILE
package legacy

import (
	"strings"

	configpkg "asana-youtrack-sync/config"
	"asana-youtrack-sync/database"
)

// ComparisonService handles ticket comparison for change detection
type ComparisonService struct {
	db              *database.DB
	configService   *configpkg.Service
	asanaService    *AsanaService
	youtrackService *YouTrackService
}

// NewComparisonService creates a new comparison service
func NewComparisonService(db *database.DB, configService *configpkg.Service) *ComparisonService {
	return &ComparisonService{
		db:              db,
		configService:   configService,
		asanaService:    NewAsanaService(configService),
		youtrackService: NewYouTrackService(configService),
	}
}

// TicketChanges represents detected changes in a ticket
type TicketChanges struct {
	HasTitleChange       bool   `json:"has_title_change"`
	HasDescriptionChange bool   `json:"has_description_change"`
	HasStatusChange      bool   `json:"has_status_change"`
	OldTitle             string `json:"old_title,omitempty"`
	NewTitle             string `json:"new_title,omitempty"`
	OldDescription       string `json:"old_description,omitempty"`
	NewDescription       string `json:"new_description,omitempty"`
}

// CompareTickets compares Asana task with YouTrack issue for changes
func (cs *ComparisonService) CompareTickets(asanaTask AsanaTask, youtrackIssue YouTrackIssue) TicketChanges {
	changes := TicketChanges{}

	// Compare titles (sanitize both for fair comparison)
	asanaTitle := sanitizeForComparison(asanaTask.Name)
	youtrackTitle := sanitizeForComparison(youtrackIssue.Summary)

	if asanaTitle != youtrackTitle {
		changes.HasTitleChange = true
		changes.OldTitle = youtrackIssue.Summary
		changes.NewTitle = asanaTask.Name
	}

	// Compare descriptions
	asanaDesc := sanitizeForComparison(asanaTask.Notes)
	youtrackDesc := cs.extractOriginalDescription(youtrackIssue.Description)
	youtrackDescClean := sanitizeForComparison(youtrackDesc)

	if asanaDesc != youtrackDescClean {
		changes.HasDescriptionChange = true
		changes.OldDescription = youtrackDesc
		changes.NewDescription = asanaTask.Notes
	}

	return changes
}

// extractOriginalDescription removes the Asana ID tag from YouTrack description
func (cs *ComparisonService) extractOriginalDescription(description string) string {
	// Remove the "[Synced from Asana ID: ...]" part
	lines := strings.Split(description, "\n")
	var cleanLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.Contains(trimmed, "[Synced from Asana ID:") {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.TrimSpace(strings.Join(cleanLines, "\n"))
}

// HasAnyChanges checks if there are any changes
func (tc *TicketChanges) HasAnyChanges() bool {
	return tc.HasTitleChange || tc.HasDescriptionChange || tc.HasStatusChange
}

// sanitizeForComparison normalizes text for comparison
func sanitizeForComparison(text string) string {
	// Trim whitespace
	text = strings.TrimSpace(text)

	// Normalize line endings
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// Remove multiple consecutive newlines
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}

	// Lowercase for case-insensitive comparison
	text = strings.ToLower(text)

	return text
}

// CheckMappingChanges checks all mapped tickets for changes
func (cs *ComparisonService) CheckMappingChanges(userID int) ([]MappingChangeInfo, error) {
	// Get all mappings
	mappings, err := cs.db.GetAllTicketMappings(userID)
	if err != nil {
		return nil, err
	}

	// Get all Asana tasks
	asanaTasks, err := cs.asanaService.GetTasks(userID)
	if err != nil {
		return nil, err
	}

	// Get all YouTrack issues
	youtrackIssues, err := cs.youtrackService.GetIssues(userID)
	if err != nil {
		return nil, err
	}

	// Create lookup maps
	asanaMap := make(map[string]AsanaTask)
	for _, task := range asanaTasks {
		asanaMap[task.GID] = task
	}

	youtrackMap := make(map[string]YouTrackIssue)
	for _, issue := range youtrackIssues {
		youtrackMap[issue.ID] = issue
	}

	// Check each mapping for changes
	var changeInfos []MappingChangeInfo
	for _, mapping := range mappings {
		asanaTask, hasAsana := asanaMap[mapping.AsanaTaskID]
		youtrackIssue, hasYoutrack := youtrackMap[mapping.YouTrackIssueID]

		if !hasAsana || !hasYoutrack {
			continue // Skip if either side is missing
		}

		changes := cs.CompareTickets(asanaTask, youtrackIssue)
		if changes.HasAnyChanges() {
			changeInfos = append(changeInfos, MappingChangeInfo{
				MappingID:       mapping.ID,
				AsanaTaskID:     mapping.AsanaTaskID,
				YouTrackIssueID: mapping.YouTrackIssueID,
				Changes:         changes,
			})
		}
	}

	return changeInfos, nil
}

// MappingChangeInfo represents a mapping with detected changes
type MappingChangeInfo struct {
	MappingID       int           `json:"mapping_id"`
	AsanaTaskID     string        `json:"asana_task_id"`
	YouTrackIssueID string        `json:"youtrack_issue_id"`
	Changes         TicketChanges `json:"changes"`
}
