package legacy

import (
	"fmt"

	configpkg "asana-youtrack-sync/config"
	"asana-youtrack-sync/database"
)

// SyncService handles synchronization operations
type SyncService struct {
	db              *database.DB
	configService   *configpkg.Service
	asanaService    *AsanaService
	youtrackService *YouTrackService
	analysisService *AnalysisService
	ignoreService   *IgnoreService
}

// NewSyncService creates a new sync service
func NewSyncService(db *database.DB, configService *configpkg.Service) *SyncService {
	return &SyncService{
		db:              db,
		configService:   configService,
		asanaService:    NewAsanaService(configService),
		youtrackService: NewYouTrackService(configService),
		analysisService: NewAnalysisService(db, configService),
		ignoreService:   NewIgnoreService(db, configService),
	}
}

// CreateMissingTickets creates missing tickets in YouTrack
func (s *SyncService) CreateMissingTickets(userID int, column ...string) (map[string]interface{}, error) {
	var columnsToAnalyze []string

	if len(column) > 0 && column[0] != "" && column[0] != "all_syncable" {
		columnsToAnalyze = []string{column[0]}
		fmt.Printf("CREATE: Creating tickets for specific column: %s (user %d)\n", column[0], userID)
	} else {
		columnsToAnalyze = SyncableColumns
		fmt.Printf("CREATE: Creating tickets for all syncable columns (user %d)\n", userID)
	}

	analysis, err := s.analysisService.PerformAnalysis(userID, columnsToAnalyze)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	if len(analysis.MissingYouTrack) == 0 {
		return map[string]interface{}{
			"status":  "success",
			"message": "No missing tickets to create",
			"created": 0,
			"column":  columnsToAnalyze,
		}, nil
	}

	// Get user settings for project IDs
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	results := []map[string]interface{}{}
	created := 0
	skipped := 0

	for _, task := range analysis.MissingYouTrack {
		asanaTags := s.asanaService.GetTags(task)

		result := map[string]interface{}{
			"task_id":    task.GID,
			"task_name":  task.Name,
			"asana_tags": asanaTags,
		}

		if s.youtrackService.IsDuplicateTicket(userID, task.Name) {
			result["status"] = "skipped"
			result["reason"] = "Duplicate ticket already exists"
			skipped++
		} else if s.ignoreService.IsIgnored(userID, task.GID) {
			result["status"] = "skipped"
			result["reason"] = "Ticket is ignored"
			skipped++
		} else {
			// Create issue in YouTrack and get the issue ID
			createdIssueID, err := s.youtrackService.CreateIssueWithReturn(userID, task)
			if err != nil {
				result["status"] = "failed"
				result["error"] = err.Error()
			} else {
				result["status"] = "created"
				result["youtrack_issue_id"] = createdIssueID

				// Create mapping in database
				_, mappingErr := s.db.CreateTicketMapping(
					userID,
					settings.AsanaProjectID,
					task.GID,
					settings.YouTrackProjectID,
					createdIssueID,
				)

				if mappingErr != nil {
					fmt.Printf("WARNING: Created ticket but failed to create mapping: %v\n", mappingErr)
				} else {
					result["mapping_created"] = true
					fmt.Printf("Created mapping: Asana %s <-> YouTrack %s\n", task.GID, createdIssueID)
				}

				if len(asanaTags) > 0 {
					tagMapper := NewTagMapperForUser(userID, s.configService)
					primaryTag := asanaTags[0]
					mappedSubsystem := tagMapper.MapTagToSubsystem(primaryTag)
					result["mapped_subsystem"] = mappedSubsystem
				}
				created++
			}
		}
		results = append(results, result)
	}

	return map[string]interface{}{
		"status":  "completed",
		"created": created,
		"skipped": skipped,
		"total":   len(analysis.MissingYouTrack),
		"column":  columnsToAnalyze,
		"results": results,
	}, nil
}

// CreateSingleTicket creates a single ticket in YouTrack
func (s *SyncService) CreateSingleTicket(userID int, taskID string) (map[string]interface{}, error) {
	allTasks, err := s.asanaService.GetTasks(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Asana tasks: %w", err)
	}

	var targetTask *AsanaTask
	for _, task := range allTasks {
		if task.GID == taskID {
			targetTask = &task
			break
		}
	}

	if targetTask == nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	asanaTags := s.asanaService.GetTags(*targetTask)

	if s.youtrackService.IsDuplicateTicket(userID, targetTask.Name) {
		return map[string]interface{}{
			"status":     "skipped",
			"reason":     "Duplicate ticket already exists",
			"task_id":    taskID,
			"task_name":  targetTask.Name,
			"asana_tags": asanaTags,
		}, nil
	}

	if s.ignoreService.IsIgnored(userID, taskID) {
		return map[string]interface{}{
			"status":     "skipped",
			"reason":     "Ticket is ignored",
			"task_id":    taskID,
			"task_name":  targetTask.Name,
			"asana_tags": asanaTags,
		}, nil
	}

	// Get user settings for project IDs
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	// Create issue and get the created issue ID
	createdIssueID, err := s.youtrackService.CreateIssueWithReturn(userID, *targetTask)
	if err != nil {
		return map[string]interface{}{
			"status":     "failed",
			"error":      err.Error(),
			"task_id":    taskID,
			"task_name":  targetTask.Name,
			"asana_tags": asanaTags,
		}, nil
	}

	response := map[string]interface{}{
		"status":            "created",
		"task_id":           taskID,
		"task_name":         targetTask.Name,
		"asana_tags":        asanaTags,
		"youtrack_issue_id": createdIssueID,
	}

	// Create mapping in database
	_, mappingErr := s.db.CreateTicketMapping(
		userID,
		settings.AsanaProjectID,
		taskID,
		settings.YouTrackProjectID,
		createdIssueID,
	)

	if mappingErr != nil {
		fmt.Printf("WARNING: Created ticket but failed to create mapping: %v\n", mappingErr)
		response["mapping_error"] = mappingErr.Error()
	} else {
		response["mapping_created"] = true
		fmt.Printf("Created mapping: Asana %s <-> YouTrack %s\n", taskID, createdIssueID)
	}

	if len(asanaTags) > 0 {
		tagMapper := NewTagMapperForUser(userID, s.configService)
		primaryTag := asanaTags[0]
		mappedSubsystem := tagMapper.MapTagToSubsystem(primaryTag)
		response["mapped_subsystem"] = mappedSubsystem
	}

	return response, nil
}

// SyncMismatchedTickets synchronizes mismatched tickets
func (s *SyncService) SyncMismatchedTickets(userID int, requests []SyncRequest, column ...string) (map[string]interface{}, error) {
	var columnsToAnalyze []string

	if len(column) > 0 && column[0] != "" && column[0] != "all_syncable" {
		columnsToAnalyze = []string{column[0]}
		fmt.Printf("SYNC: Syncing tickets for specific column: %s (user %d)\n", column[0], userID)
	} else {
		columnsToAnalyze = SyncableColumns
		fmt.Printf("SYNC: Syncing tickets for all syncable columns (user %d)\n", userID)
	}

	analysis, err := s.analysisService.PerformAnalysis(userID, columnsToAnalyze)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	mismatchMap := make(map[string]MismatchedTicket)
	for _, ticket := range analysis.Mismatched {
		mismatchMap[ticket.AsanaTask.GID] = ticket
	}

	results := []map[string]interface{}{}
	synced := 0

	for _, req := range requests {
		result := map[string]interface{}{
			"ticket_id": req.TicketID,
			"action":    req.Action,
		}

		ticket, exists := mismatchMap[req.TicketID]
		if !exists {
			result["status"] = "failed"
			result["error"] = "Ticket not found in mismatched list for this column"
			results = append(results, result)
			continue
		}

		switch req.Action {
		case "sync":
			if s.ignoreService.IsIgnored(userID, req.TicketID) {
				result["status"] = "skipped"
				result["reason"] = "Ticket is ignored"
			} else {
				err := s.youtrackService.UpdateIssue(userID, ticket.YouTrackIssue.ID, ticket.AsanaTask)
				if err != nil {
					result["status"] = "failed"
					result["error"] = err.Error()
				} else {
					result["status"] = "synced"
					result["status_change"] = map[string]string{
						"from": ticket.YouTrackStatus,
						"to":   ticket.AsanaStatus,
					}

					asanaTags := s.asanaService.GetTags(ticket.AsanaTask)
					if len(asanaTags) > 0 {
						tagMapper := NewTagMapperForUser(userID, s.configService)
						primaryTag := asanaTags[0]
						mappedSubsystem := tagMapper.MapTagToSubsystem(primaryTag)
						result["tag_sync"] = map[string]interface{}{
							"asana_tags":         asanaTags,
							"mapped_subsystem":   mappedSubsystem,
							"previous_subsystem": ticket.YouTrackSubsystem,
						}
					}
					synced++
				}
			}

		case "ignore_temp":
			err := s.ignoreService.AddTemporaryIgnore(userID, req.TicketID)
			if err != nil {
				result["status"] = "failed"
				result["error"] = err.Error()
			} else {
				result["status"] = "ignored_temporarily"
			}

		case "ignore_forever":
			err := s.ignoreService.AddForeverIgnore(userID, req.TicketID)
			if err != nil {
				result["status"] = "failed"
				result["error"] = err.Error()
			} else {
				result["status"] = "ignored_permanently"
			}

		default:
			result["status"] = "failed"
			result["error"] = "Invalid action"
		}

		results = append(results, result)
	}

	return map[string]interface{}{
		"status":  "completed",
		"synced":  synced,
		"total":   len(requests),
		"column":  columnsToAnalyze,
		"results": results,
		"note":    "Sync operations now include both status and tag/subsystem updates",
	}, nil
}

// GetMismatchedTickets returns mismatched tickets for preview
func (s *SyncService) GetMismatchedTickets(userID int, column ...string) (map[string]interface{}, error) {
	var columnsToAnalyze []string

	if len(column) > 0 && column[0] != "" && column[0] != "all_syncable" {
		columnsToAnalyze = []string{column[0]}
	} else {
		columnsToAnalyze = SyncableColumns
	}

	analysis, err := s.analysisService.PerformAnalysis(userID, columnsToAnalyze)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	return map[string]interface{}{
		"status":     "success",
		"message":    "Mismatched tickets available for sync",
		"count":      len(analysis.Mismatched),
		"column":     columnsToAnalyze,
		"mismatched": analysis.Mismatched,
		"usage": map[string]string{
			"sync_all":       "POST with [{\"ticket_id\":\"ID\",\"action\":\"sync\"}] for each ticket",
			"ignore_temp":    "POST with [{\"ticket_id\":\"ID\",\"action\":\"ignore_temp\"}]",
			"ignore_forever": "POST with [{\"ticket_id\":\"ID\",\"action\":\"ignore_forever\"}]",
		},
		"note": "Sync now includes both status and tag/subsystem synchronization",
	}, nil
}

// ValidateSyncRequests validates sync requests
func (s *SyncService) ValidateSyncRequests(requests []SyncRequest) error {
	if len(requests) == 0 {
		return fmt.Errorf("no sync requests provided")
	}

	validActions := map[string]bool{
		"sync":           true,
		"ignore_temp":    true,
		"ignore_forever": true,
	}

	for i, req := range requests {
		if req.TicketID == "" {
			return fmt.Errorf("request %d: ticket_id is required", i)
		}
		if req.Action == "" {
			return fmt.Errorf("request %d: action is required", i)
		}
		if !validActions[req.Action] {
			return fmt.Errorf("request %d: invalid action '%s'. Valid actions: sync, ignore_temp, ignore_forever", i, req.Action)
		}
	}

	return nil
}

// GetSyncableTickets returns tickets that can be synced
func (s *SyncService) GetSyncableTickets(userID int) (map[string]interface{}, error) {
	analysis, err := s.analysisService.PerformAnalysis(userID, SyncableColumns)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	syncableTickets := []map[string]interface{}{}

	for _, ticket := range analysis.Mismatched {
		if !s.ignoreService.IsIgnored(userID, ticket.AsanaTask.GID) {
			syncableTickets = append(syncableTickets, map[string]interface{}{
				"ticket_id":       ticket.AsanaTask.GID,
				"ticket_name":     ticket.AsanaTask.Name,
				"type":            "mismatched",
				"asana_status":    ticket.AsanaStatus,
				"youtrack_status": ticket.YouTrackStatus,
				"tags":            s.asanaService.GetTags(ticket.AsanaTask),
			})
		}
	}

	return map[string]interface{}{
		"syncable_tickets": syncableTickets,
		"count":            len(syncableTickets),
		"ignored_count":    s.ignoreService.CountIgnored(userID),
	}, nil
}

// GetSyncStats returns synchronization statistics
func (s *SyncService) GetSyncStats(userID int) (map[string]interface{}, error) {
	analysis, err := s.analysisService.PerformAnalysis(userID, SyncableColumns)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	stats := map[string]interface{}{
		"total_asana_tasks":     len(analysis.Matched) + len(analysis.Mismatched) + len(analysis.MissingYouTrack),
		"total_youtrack_issues": len(analysis.Matched) + len(analysis.Mismatched) + len(analysis.OrphanedYouTrack),
		"matched_tickets":       len(analysis.Matched),
		"mismatched_tickets":    len(analysis.Mismatched),
		"missing_in_youtrack":   len(analysis.MissingYouTrack),
		"orphaned_in_youtrack":  len(analysis.OrphanedYouTrack),
		"blocked_tickets":       len(analysis.BlockedTickets),
		"findings_tickets":      len(analysis.FindingsTickets),
		"findings_alerts":       len(analysis.FindingsAlerts),
		"ready_for_stage":       len(analysis.ReadyForStage),
		"ignored_tickets":       len(analysis.Ignored),
	}

	totalTickets := stats["total_asana_tasks"].(int)
	if totalTickets > 0 {
		matchedCount := stats["matched_tickets"].(int)
		stats["sync_health_percentage"] = float64(matchedCount) / float64(totalTickets) * 100
	} else {
		stats["sync_health_percentage"] = 100.0
	}

	statusMismatches := make(map[string]int)
	for _, ticket := range analysis.Mismatched {
		key := fmt.Sprintf("%s -> %s", ticket.YouTrackStatus, ticket.AsanaStatus)
		statusMismatches[key]++
	}
	stats["status_mismatches_breakdown"] = statusMismatches

	return stats, nil
}

// SyncTicketsByColumn syncs tickets from a specific column
func (s *SyncService) SyncTicketsByColumn(userID int, column string) (map[string]interface{}, error) {
	var columnsToAnalyze []string
	if column == "" || column == "all_syncable" {
		columnsToAnalyze = SyncableColumns
	} else {
		columnMap := map[string]string{
			"backlog":         "backlog",
			"in_progress":     "in progress",
			"dev":             "dev",
			"stage":           "stage",
			"blocked":         "blocked",
			"ready_for_stage": "ready for stage",
			"findings":        "findings",
		}

		if mappedColumn, exists := columnMap[column]; exists {
			columnsToAnalyze = []string{mappedColumn}
		} else {
			return nil, fmt.Errorf("invalid column: %s", column)
		}
	}

	analysis, err := s.analysisService.PerformAnalysis(userID, columnsToAnalyze)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	synced := 0
	errors := 0
	results := []map[string]interface{}{}

	for _, ticket := range analysis.Mismatched {
		if s.ignoreService.IsIgnored(userID, ticket.AsanaTask.GID) {
			continue
		}

		result := map[string]interface{}{
			"ticket_id":   ticket.AsanaTask.GID,
			"ticket_name": ticket.AsanaTask.Name,
		}

		err := s.youtrackService.UpdateIssue(userID, ticket.YouTrackIssue.ID, ticket.AsanaTask)
		if err != nil {
			result["status"] = "failed"
			result["error"] = err.Error()
			errors++
		} else {
			result["status"] = "synced"
			result["status_change"] = map[string]string{
				"from": ticket.YouTrackStatus,
				"to":   ticket.AsanaStatus,
			}
			synced++
		}

		results = append(results, result)
	}

	return map[string]interface{}{
		"column":  column,
		"synced":  synced,
		"errors":  errors,
		"total":   len(analysis.Mismatched),
		"results": results,
	}, nil
}

// CreateTicketsByColumn creates missing tickets from a specific column
func (s *SyncService) CreateTicketsByColumn(userID int, column string) (map[string]interface{}, error) {
	var columnsToAnalyze []string
	if column == "" || column == "all_syncable" {
		columnsToAnalyze = SyncableColumns
	} else {
		columnMap := map[string]string{
			"backlog":     "backlog",
			"in_progress": "in progress",
			"dev":         "dev",
			"stage":       "stage",
			"blocked":     "blocked",
		}

		if mappedColumn, exists := columnMap[column]; exists {
			columnsToAnalyze = []string{mappedColumn}
		} else {
			return nil, fmt.Errorf("invalid column: %s", column)
		}
	}

	analysis, err := s.analysisService.PerformAnalysis(userID, columnsToAnalyze)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	// Get user settings for project IDs
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	created := 0
	skipped := 0
	errors := 0
	results := []map[string]interface{}{}

	for _, task := range analysis.MissingYouTrack {
		result := map[string]interface{}{
			"task_id":   task.GID,
			"task_name": task.Name,
		}

		if s.youtrackService.IsDuplicateTicket(userID, task.Name) {
			result["status"] = "skipped"
			result["reason"] = "Duplicate ticket already exists"
			skipped++
		} else if s.ignoreService.IsIgnored(userID, task.GID) {
			result["status"] = "skipped"
			result["reason"] = "Ticket is ignored"
			skipped++
		} else {
			createdIssueID, err := s.youtrackService.CreateIssueWithReturn(userID, task)
			if err != nil {
				result["status"] = "failed"
				result["error"] = err.Error()
				errors++
			} else {
				result["status"] = "created"
				result["youtrack_issue_id"] = createdIssueID

				// Create mapping in database
				_, mappingErr := s.db.CreateTicketMapping(
					userID,
					settings.AsanaProjectID,
					task.GID,
					settings.YouTrackProjectID,
					createdIssueID,
				)

				if mappingErr != nil {
					fmt.Printf("WARNING: Created ticket but failed to create mapping: %v\n", mappingErr)
				} else {
					result["mapping_created"] = true
				}

				created++
			}
		}

		results = append(results, result)
	}

	return map[string]interface{}{
		"column":  column,
		"created": created,
		"skipped": skipped,
		"errors":  errors,
		"total":   len(analysis.MissingYouTrack),
		"results": results,
	}, nil
}

// GetSyncPreview provides a preview of what would be synced
func (s *SyncService) GetSyncPreview(userID int, ticketIDs []string) (map[string]interface{}, error) {
	analysis, err := s.analysisService.PerformAnalysis(userID, SyncableColumns)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	mismatchMap := make(map[string]MismatchedTicket)
	for _, ticket := range analysis.Mismatched {
		mismatchMap[ticket.AsanaTask.GID] = ticket
	}

	preview := []map[string]interface{}{}

	for _, ticketID := range ticketIDs {
		item := map[string]interface{}{
			"ticket_id": ticketID,
		}

		if ticket, exists := mismatchMap[ticketID]; exists {
			item["ticket_name"] = ticket.AsanaTask.Name
			item["current_youtrack_status"] = ticket.YouTrackStatus
			item["target_asana_status"] = ticket.AsanaStatus
			item["will_sync"] = !s.ignoreService.IsIgnored(userID, ticketID)
			item["ignored"] = s.ignoreService.IsIgnored(userID, ticketID)
			item["tags"] = s.asanaService.GetTags(ticket.AsanaTask)
		} else {
			item["error"] = "Ticket not found in mismatched list"
		}

		preview = append(preview, item)
	}

	return map[string]interface{}{
		"preview": preview,
		"total":   len(ticketIDs),
	}, nil
}

// AutoSync performs auto-sync for mismatched tickets (status only)
func (s *SyncService) AutoSync(userID int) error {
	// Get all mismatched tickets
	result, err := s.GetMismatchedTickets(userID)
	if err != nil {
		return fmt.Errorf("failed to get mismatched tickets: %w", err)
	}

	// Check if there are tickets to sync
	if mismatchedData, ok := result["mismatched"].([]MismatchedTicket); ok && len(mismatchedData) > 0 {
		// Create sync requests for all mismatched tickets
		var syncRequests []SyncRequest
		for _, ticket := range mismatchedData {
			// Only sync if not ignored
			if !s.ignoreService.IsIgnored(userID, ticket.AsanaTask.GID) {
				syncRequests = append(syncRequests, SyncRequest{
					TicketID: ticket.AsanaTask.GID,
					Action:   "sync",
				})
			}
		}

		if len(syncRequests) > 0 {
			// Perform the sync
			_, err = s.SyncMismatchedTickets(userID, syncRequests)
			if err != nil {
				return fmt.Errorf("sync operation failed: %w", err)
			}

			fmt.Printf("AUTO-SYNC: Synced %d tickets for user %d\n", len(syncRequests), userID)
		}
	}

	return nil
}
