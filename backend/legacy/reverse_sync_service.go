package legacy

import (
	"fmt"
	"log"
	"strings"
	"time"

	configpkg "asana-youtrack-sync/config"
	"asana-youtrack-sync/database"
	"asana-youtrack-sync/utils"
)

type ReverseSyncService struct {
	db               *database.DB
	youtrackService  *YouTrackService
	asanaService     *AsanaService
	analysisService  *ReverseAnalysisService
	configService    *configpkg.Service
	tagMapper        *TagMapper
}

func NewReverseSyncService(db *database.DB, youtrackService *YouTrackService, asanaService *AsanaService, configService *configpkg.Service) *ReverseSyncService {
	analysisService := NewReverseAnalysisService(db, youtrackService, asanaService, configService)
	tagMapper := NewTagMapper()

	return &ReverseSyncService{
		db:               db,
		youtrackService:  youtrackService,
		asanaService:     asanaService,
		analysisService:  analysisService,
		configService:    configService,
		tagMapper:        tagMapper,
	}
}

// GetYouTrackUsers fetches all users from YouTrack for the dropdown
func (s *ReverseSyncService) GetYouTrackUsers(userID int) ([]YouTrackUser, error) {
	return s.youtrackService.GetAllUsers(userID)
}

// PerformReverseAnalysis analyzes tickets created by specific user(s)
func (s *ReverseSyncService) PerformReverseAnalysis(userID int, creatorFilter string) (*ReverseTicketAnalysis, error) {
	return s.analysisService.PerformReverseAnalysis(userID, creatorFilter)
}

// CreateMissingAsanaTickets creates all missing tickets from YouTrack to Asana
func (s *ReverseSyncService) CreateMissingAsanaTickets(userID int, analysis *ReverseTicketAnalysis) (*ReverseSyncResult, error) {
	result := &ReverseSyncResult{
		TotalTickets:    len(analysis.MissingAsana),
		SuccessCount:    0,
		FailedCount:     0,
		FailedTickets:   []FailedTicket{},
		CreatedMappings: []*database.TicketMapping{},
	}

	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user settings: %w", err)
	}

	for i, ytIssue := range analysis.MissingAsana {
		log.Printf("[Reverse Sync] Creating Asana task %d/%d: %s", i+1, len(analysis.MissingAsana), ytIssue.ID)

		// Create the ticket in Asana
		asanaTaskID, err := s.CreateSingleAsanaTicket(userID, ytIssue, settings)
		if err != nil {
			log.Printf("[Reverse Sync] Failed to create task for %s: %v", ytIssue.ID, err)
			result.FailedCount++
			result.FailedTickets = append(result.FailedTickets, FailedTicket{
				IssueID: ytIssue.ID,
				Title:   ytIssue.Summary,
				Error:   err.Error(),
			})
			continue
		}

		// Create database mapping
		mapping, err := s.db.CreateTicketMapping(
			userID,
			settings.AsanaProjectID,
			asanaTaskID,
			settings.YouTrackProjectID,
			ytIssue.ID,
		)
		if err != nil {
			log.Printf("[Reverse Sync] Warning: Failed to create mapping for %s -> %s: %v", ytIssue.ID, asanaTaskID, err)
		} else {
			result.CreatedMappings = append(result.CreatedMappings, mapping)
		}

		result.SuccessCount++
		log.Printf("[Reverse Sync] Successfully created task: %s -> %s", ytIssue.ID, asanaTaskID)
	}

	return result, nil
}

// CreateSingleAsanaTicket creates a single ticket in Asana from YouTrack issue
func (s *ReverseSyncService) CreateSingleAsanaTicket(userID int, ytIssue YouTrackIssue, settings *configpkg.UserSettings) (string, error) {
	// 1. Get the Asana section (column) based on YouTrack state
	asanaSection, err := s.mapYouTrackStateToAsanaSection(userID, ytIssue.State, settings)
	if err != nil {
		return "", fmt.Errorf("failed to map state: %w", err)
	}

	// 2. Keep the YouTrack title format with ID prefix (e.g., "ARD-123 Fix bug")
	taskTitle := fmt.Sprintf("%s %s", ytIssue.ID, ytIssue.Summary)

	// 3. Convert YouTrack wikified HTML description to Asana HTML
	// Use wikifiedDescription if available (it has proper HTML formatting)
	// Otherwise fall back to plain text description
	htmlDescription := ""
	if ytIssue.WikifiedDescription != "" {
		log.Printf("[Reverse Sync] Original wikified HTML for %s: %s", ytIssue.ID, ytIssue.WikifiedDescription)
		htmlDescription = utils.ConvertYouTrackWikifiedToAsanaHTML(ytIssue.WikifiedDescription)
		log.Printf("[Reverse Sync] Converted HTML for %s: %s", ytIssue.ID, htmlDescription)
	} else {
		log.Printf("[Reverse Sync] No wikified description for %s, using plain text", ytIssue.ID)
	}

	// 4. Map YouTrack subsystem to Asana tags
	asanaTags, err := s.mapSubsystemToAsanaTags(userID, ytIssue.Subsystem, settings)
	if err != nil {
		log.Printf("[Reverse Sync] Warning: Failed to map subsystem '%s' to tags: %v", ytIssue.Subsystem, err)
		asanaTags = []string{} // Skip tags if mapping fails
	}

	// 5. Create the task in Asana with HTML description
	taskData := map[string]interface{}{
		"name":     taskTitle,
		"projects": []string{settings.AsanaProjectID},
	}

	// Add description - use HTML if available, otherwise plain text
	if htmlDescription != "" {
		taskData["html_notes"] = htmlDescription
	} else {
		taskData["notes"] = ytIssue.Description
	}

	// Add section/column
	if asanaSection != "" {
		taskData["memberships"] = []map[string]string{
			{
				"project": settings.AsanaProjectID,
				"section": asanaSection,
			},
		}
	}

	asanaTaskID, err := s.asanaService.CreateTask(userID, taskData)
	if err != nil {
		return "", fmt.Errorf("failed to create Asana task: %w", err)
	}

	// 6. Add tags to the created task
	for _, tagName := range asanaTags {
		err := s.asanaService.AddTagToTask(userID, asanaTaskID, tagName)
		if err != nil {
			log.Printf("[Reverse Sync] Warning: Failed to add tag '%s' to task %s: %v", tagName, asanaTaskID, err)
		}
	}

	// 7. Sync attachments from YouTrack to Asana
	if len(ytIssue.Attachments) > 0 {
		log.Printf("[Reverse Sync] Syncing %d attachments for %s", len(ytIssue.Attachments), ytIssue.ID)
		err := s.syncAttachmentsToAsana(userID, ytIssue.ID, asanaTaskID, ytIssue.Attachments)
		if err != nil {
			log.Printf("[Reverse Sync] Warning: Failed to sync attachments: %v", err)
		}
	}

	return asanaTaskID, nil
}

// mapYouTrackStateToAsanaSection maps YouTrack state to Asana section using reverse column mappings
func (s *ReverseSyncService) mapYouTrackStateToAsanaSection(userID int, ytState string, settings *configpkg.UserSettings) (string, error) {
	// Use the existing asana_to_youtrack mappings in reverse
	for _, mapping := range settings.ColumnMappings.AsanaToYouTrack {
		if strings.EqualFold(mapping.YouTrackStatus, ytState) {
			// Find the Asana section ID by name
			sections, err := s.asanaService.GetProjectSections(userID, settings.AsanaProjectID)
			if err != nil {
				return "", fmt.Errorf("failed to get Asana sections: %w", err)
			}

			for _, section := range sections {
				if strings.EqualFold(section.Name, mapping.AsanaColumn) {
					return section.GID, nil
				}
			}
			return "", fmt.Errorf("Asana section not found: %s", mapping.AsanaColumn)
		}
	}

	return "", fmt.Errorf("no mapping found for YouTrack state: %s", ytState)
}

// mapSubsystemToAsanaTags maps YouTrack subsystem to Asana tags using reverse tag mappings
func (s *ReverseSyncService) mapSubsystemToAsanaTags(userID int, subsystem string, settings *configpkg.UserSettings) ([]string, error) {
	if subsystem == "" {
		return []string{}, nil
	}

	// Use the existing tag mappings in reverse
	// TagMapping format: {"AsanaTagName": "YouTrackSubsystem"}
	// We need to reverse it: find AsanaTagName where YouTrackSubsystem matches
	for asanaTag, ytSubsystem := range settings.CustomFieldMappings.TagMapping {
		if strings.EqualFold(ytSubsystem, subsystem) {
			return []string{asanaTag}, nil
		}
	}

	// No mapping found - skip the tag as per requirements
	log.Printf("[Reverse Sync] No tag mapping found for subsystem: %s", subsystem)
	return []string{}, nil
}

// syncAttachmentsToAsana downloads attachments from YouTrack and uploads them to Asana
func (s *ReverseSyncService) syncAttachmentsToAsana(userID int, ytIssueID, asanaTaskID string, attachments []YouTrackAttachment) error {
	successCount := 0
	for i, attachment := range attachments {
		log.Printf("[Reverse Sync] Processing attachment %d/%d: %s", i+1, len(attachments), attachment.Name)

		// Download from YouTrack using the URL from the API response
		fileData, err := s.youtrackService.DownloadAttachment(userID, ytIssueID, attachment.URL)
		if err != nil {
			log.Printf("[Reverse Sync] Failed to download attachment %s: %v", attachment.Name, err)
			continue
		}

		// Upload to Asana
		err = s.asanaService.UploadAttachment(userID, asanaTaskID, attachment.Name, fileData)
		if err != nil {
			log.Printf("[Reverse Sync] Failed to upload attachment %s to Asana: %v", attachment.Name, err)
			continue
		}

		successCount++
		log.Printf("[Reverse Sync] Successfully synced attachment: %s", attachment.Name)

		// Small delay to avoid rate limiting
		time.Sleep(500 * time.Millisecond)
	}

	if successCount > 0 {
		log.Printf("[Reverse Sync] Synced %d/%d attachments successfully", successCount, len(attachments))
	}

	return nil
}

// CreateSelectedAsanaTickets creates only selected tickets from the analysis
func (s *ReverseSyncService) CreateSelectedAsanaTickets(userID int, selectedIssueIDs []string, analysis *ReverseTicketAnalysis) (*ReverseSyncResult, error) {
	result := &ReverseSyncResult{
		TotalTickets:    len(selectedIssueIDs),
		SuccessCount:    0,
		FailedCount:     0,
		FailedTickets:   []FailedTicket{},
		CreatedMappings: []*database.TicketMapping{},
	}

	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user settings: %w", err)
	}

	// Create a map for quick lookup
	issueMap := make(map[string]YouTrackIssue)
	for _, issue := range analysis.MissingAsana {
		issueMap[issue.ID] = issue
	}

	for i, issueID := range selectedIssueIDs {
		ytIssue, exists := issueMap[issueID]
		if !exists {
			log.Printf("[Reverse Sync] Issue not found in analysis: %s", issueID)
			result.FailedCount++
			result.FailedTickets = append(result.FailedTickets, FailedTicket{
				IssueID: issueID,
				Title:   "",
				Error:   "Issue not found in analysis results",
			})
			continue
		}

		log.Printf("[Reverse Sync] Creating selected task %d/%d: %s", i+1, len(selectedIssueIDs), ytIssue.ID)

		asanaTaskID, err := s.CreateSingleAsanaTicket(userID, ytIssue, settings)
		if err != nil {
			log.Printf("[Reverse Sync] Failed to create task for %s: %v", ytIssue.ID, err)
			result.FailedCount++
			result.FailedTickets = append(result.FailedTickets, FailedTicket{
				IssueID: ytIssue.ID,
				Title:   ytIssue.Summary,
				Error:   err.Error(),
			})
			continue
		}

		// Create database mapping
		mapping, err := s.db.CreateTicketMapping(
			userID,
			settings.AsanaProjectID,
			asanaTaskID,
			settings.YouTrackProjectID,
			ytIssue.ID,
		)
		if err != nil {
			log.Printf("[Reverse Sync] Warning: Failed to create mapping for %s -> %s: %v", ytIssue.ID, asanaTaskID, err)
		} else {
			result.CreatedMappings = append(result.CreatedMappings, mapping)
		}

		result.SuccessCount++
		log.Printf("[Reverse Sync] Successfully created selected task: %s -> %s", ytIssue.ID, asanaTaskID)
	}

	return result, nil
}
