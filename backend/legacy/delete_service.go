package legacy

import (
	"fmt"
	"strings"

	configpkg "asana-youtrack-sync/config"
)

// DeleteService handles bulk deletion operations
type DeleteService struct {
	configService   *configpkg.Service
	asanaService    *AsanaService
	youtrackService *YouTrackService
}

// NewDeleteService creates a new delete service
func NewDeleteService(configService *configpkg.Service) *DeleteService {
	return &DeleteService{
		configService:   configService,
		asanaService:    NewAsanaService(configService),
		youtrackService: NewYouTrackService(configService),
	}
}

// PerformBulkDelete performs bulk deletion of tickets
func (s *DeleteService) PerformBulkDelete(userID int, ticketIDs []string, source string) DeleteResponse {
	response := DeleteResponse{
		Source:         source,
		RequestedCount: len(ticketIDs),
		Results:        make([]DeleteResult, 0, len(ticketIDs)),
	}

	for _, ticketID := range ticketIDs {
		result := DeleteResult{
			TicketID:   ticketID,
			TicketName: s.getTicketName(userID, ticketID),
		}

		switch source {
		case "asana":
			s.deleteFromAsana(userID, ticketID, &result, &response)
		case "youtrack":
			s.deleteFromYouTrack(userID, ticketID, &result, &response)
		case "both":
			s.deleteFromBoth(userID, ticketID, &result, &response)
		default:
			result.Status = "failed"
			result.Error = "Invalid source specified"
			response.FailureCount++
		}

		response.Results = append(response.Results, result)
	}

	// Set overall status and summary
	s.setResponseStatus(&response)
	return response
}

// deleteFromAsana deletes a ticket from Asana only
func (s *DeleteService) deleteFromAsana(userID int, ticketID string, result *DeleteResult, response *DeleteResponse) {
	err := s.asanaService.DeleteTask(userID, ticketID)
	if err != nil {
		result.Status = "failed"
		result.AsanaResult = "failed"
		result.Error = err.Error()
		response.FailureCount++
	} else {
		result.Status = "success"
		result.AsanaResult = "deleted"
		response.SuccessCount++
	}
}

// deleteFromYouTrack deletes a ticket from YouTrack only
func (s *DeleteService) deleteFromYouTrack(userID int, ticketID string, result *DeleteResult, response *DeleteResponse) {
	// First try to use as direct YouTrack issue ID
	youtrackIssueID := ticketID
	err := s.youtrackService.DeleteIssue(userID, youtrackIssueID)

	// If that fails, try to find YouTrack issue by Asana ID
	if err != nil {
		youtrackIssueID, findErr := s.youtrackService.FindIssueByAsanaID(userID, ticketID)
		if findErr != nil {
			result.Status = "failed"
			result.YouTrackResult = "failed"
			result.Error = fmt.Sprintf("Issue not found: %v", findErr)
			response.FailureCount++
			return
		}

		err = s.youtrackService.DeleteIssue(userID, youtrackIssueID)
		if err != nil {
			result.Status = "failed"
			result.YouTrackResult = "failed"
			result.Error = err.Error()
			response.FailureCount++
			return
		}
	}

	result.Status = "success"
	result.YouTrackResult = "deleted"
	response.SuccessCount++
}

// deleteFromBoth deletes a ticket from both Asana and YouTrack
func (s *DeleteService) deleteFromBoth(userID int, ticketID string, result *DeleteResult, response *DeleteResponse) {
	asanaSuccess := true
	youtrackSuccess := true
	var errors []string

	// Delete from Asana
	err := s.asanaService.DeleteTask(userID, ticketID)
	if err != nil {
		asanaSuccess = false
		result.AsanaResult = "failed"
		errors = append(errors, fmt.Sprintf("Asana: %v", err))
	} else {
		result.AsanaResult = "deleted"
	}

	// Delete from YouTrack
	youtrackIssueID, findErr := s.youtrackService.FindIssueByAsanaID(userID, ticketID)
	if findErr != nil {
		youtrackSuccess = false
		result.YouTrackResult = "not_found"
		errors = append(errors, fmt.Sprintf("YouTrack: %v", findErr))
	} else {
		err = s.youtrackService.DeleteIssue(userID, youtrackIssueID)
		if err != nil {
			youtrackSuccess = false
			result.YouTrackResult = "failed"
			errors = append(errors, fmt.Sprintf("YouTrack: %v", err))
		} else {
			result.YouTrackResult = "deleted"
		}
	}

	// Determine overall status
	if asanaSuccess && youtrackSuccess {
		result.Status = "success"
		response.SuccessCount++
	} else if asanaSuccess || youtrackSuccess {
		result.Status = "partial"
		response.SuccessCount++
	} else {
		result.Status = "failed"
		response.FailureCount++
	}

	if len(errors) > 0 {
		result.Error = strings.Join(errors, "; ")
	}
}

// getTicketName gets the name/title of a ticket for deletion reporting
func (s *DeleteService) getTicketName(userID int, ticketID string) string {
	// Try to get from Asana tasks
	tasks, err := s.asanaService.GetTasks(userID)
	if err == nil {
		for _, task := range tasks {
			if task.GID == ticketID {
				return task.Name
			}
		}
	}

	// Try to get from YouTrack issues
	issues, err := s.youtrackService.GetIssues(userID)
	if err == nil {
		for _, issue := range issues {
			asanaID := s.youtrackService.ExtractAsanaID(issue)
			if asanaID == ticketID || issue.ID == ticketID {
				return issue.Summary
			}
		}
	}

	// Fallback name
	return fmt.Sprintf("Ticket-%s", ticketID)
}

// setResponseStatus sets the overall response status and summary
func (s *DeleteService) setResponseStatus(response *DeleteResponse) {
	if response.SuccessCount == response.RequestedCount {
		response.Status = "success"
		response.Summary = fmt.Sprintf("Successfully deleted all %d tickets from %s", 
			response.SuccessCount, response.Source)
	} else if response.SuccessCount > 0 {
		response.Status = "partial"
		response.Summary = fmt.Sprintf("Deleted %d of %d tickets from %s (%d failed)", 
			response.SuccessCount, response.RequestedCount, response.Source, response.FailureCount)
	} else {
		response.Status = "failed"
		response.Summary = fmt.Sprintf("Failed to delete any tickets from %s", response.Source)
	}
}

// ValidateDeleteRequest validates a delete request
func (s *DeleteService) ValidateDeleteRequest(req DeleteTicketsRequest) error {
	if len(req.TicketIDs) == 0 {
		return fmt.Errorf("ticket_ids is required and must not be empty")
	}

	if req.Source == "" {
		return fmt.Errorf("source is required")
	}

	validSources := map[string]bool{"asana": true, "youtrack": true, "both": true}
	if !validSources[req.Source] {
		return fmt.Errorf("invalid source: %s. Valid sources are: asana, youtrack, both", req.Source)
	}

	return nil
}

// GetDeletionPreview provides a preview of what will be deleted
func (s *DeleteService) GetDeletionPreview(userID int, ticketIDs []string, source string) (map[string]interface{}, error) {
	preview := map[string]interface{}{
		"source":          source,
		"requested_count": len(ticketIDs),
		"preview":         []map[string]interface{}{},
	}

	var previewItems []map[string]interface{}

	for _, ticketID := range ticketIDs {
		item := map[string]interface{}{
			"ticket_id":   ticketID,
			"ticket_name": s.getTicketName(userID, ticketID),
		}

		switch source {
		case "asana":
			item["will_delete_from"] = []string{"asana"}
		case "youtrack":
			item["will_delete_from"] = []string{"youtrack"}
		case "both":
			item["will_delete_from"] = []string{"asana", "youtrack"}
		}

		// Check if ticket exists in each system
		item["exists_in_asana"] = s.ticketExistsInAsana(userID, ticketID)
		item["exists_in_youtrack"] = s.ticketExistsInYouTrack(userID, ticketID)

		previewItems = append(previewItems, item)
	}

	preview["preview"] = previewItems
	return preview, nil
}

// ticketExistsInAsana checks if a ticket exists in Asana
func (s *DeleteService) ticketExistsInAsana(userID int, ticketID string) bool {
	tasks, err := s.asanaService.GetTasks(userID)
	if err != nil {
		return false
	}

	for _, task := range tasks {
		if task.GID == ticketID {
			return true
		}
	}
	return false
}

// ticketExistsInYouTrack checks if a ticket exists in YouTrack
func (s *DeleteService) ticketExistsInYouTrack(userID int, ticketID string) bool {
	// Try direct YouTrack ID lookup
	issues, err := s.youtrackService.GetIssues(userID)
	if err != nil {
		return false
	}

	for _, issue := range issues {
		if issue.ID == ticketID {
			return true
		}
		// Also check Asana ID mapping
		asanaID := s.youtrackService.ExtractAsanaID(issue)
		if asanaID == ticketID {
			return true
		}
	}
	return false
}

// GetDeleteStats returns statistics about deletion operations
func (s *DeleteService) GetDeleteStats(response DeleteResponse) map[string]interface{} {
	stats := map[string]interface{}{
		"total_requested": response.RequestedCount,
		"successful":      response.SuccessCount,
		"failed":          response.FailureCount,
		"success_rate":    0.0,
	}

	if response.RequestedCount > 0 {
		stats["success_rate"] = float64(response.SuccessCount) / float64(response.RequestedCount) * 100
	}

	// Breakdown by result status
	statusCounts := make(map[string]int)
	for _, result := range response.Results {
		statusCounts[result.Status]++
	}
	stats["status_breakdown"] = statusCounts

	// Breakdown by platform (for "both" source)
	if response.Source == "both" {
		asanaSuccess := 0
		youtrackSuccess := 0
		for _, result := range response.Results {
			if result.AsanaResult == "deleted" {
				asanaSuccess++
			}
			if result.YouTrackResult == "deleted" {
				youtrackSuccess++
			}
		}
		stats["asana_success_count"] = asanaSuccess
		stats["youtrack_success_count"] = youtrackSuccess
	}

	return stats
}