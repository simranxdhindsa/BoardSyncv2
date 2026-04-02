package legacy

import (
	"fmt"
	"time"

	configpkg "asana-youtrack-sync/config"
	"asana-youtrack-sync/database"
)

const ticketStabilityWindow = 10 * time.Minute

// isTicketStable returns true if the ticket hasn't been modified in the last 10 minutes
func isTicketStable(task AsanaTask) bool {
	if task.ModifiedAt == "" {
		return true // no modified_at means treat as stable
	}
	modifiedAt, err := time.Parse(time.RFC3339, task.ModifiedAt)
	if err != nil {
		return true // can't parse — treat as stable
	}
	return time.Since(modifiedAt) >= ticketStabilityWindow
}

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
	asanaSvc := NewAsanaService(configService)
	return &SyncService{
		db:              db,
		configService:   configService,
		asanaService:    asanaSvc,
		youtrackService: NewYouTrackService(configService, asanaSvc),
		analysisService: NewAnalysisService(db, configService),
		ignoreService:   NewIgnoreService(db, configService),
	}
}

// CreateMissingTickets creates missing tickets in YouTrack.
// Optimized: skips full PerformAnalysis — fetches Asana tasks, checks DB mappings, creates only truly new ones.
func (s *SyncService) CreateMissingTickets(userID int, column ...string) (map[string]interface{}, error) {
	var columnsToProcess []string
	if len(column) > 0 && column[0] != "" && column[0] != "all_syncable" {
		columnsToProcess = []string{column[0]}
		fmt.Printf("CREATE: Creating tickets for specific column: %s (user %d)\n", column[0], userID)
	} else {
		columnsToProcess = SyncableColumns
		fmt.Printf("CREATE: Creating tickets for all syncable columns (user %d)\n", userID)
	}

	// Fetch and filter Asana tasks (cached — fast)
	allTasks, err := s.asanaService.GetTasks(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Asana tasks: %w", err)
	}
	filteredTasks := s.asanaService.FilterTasksByColumns(allTasks, columnsToProcess)

	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	// Build a set of all already-mapped Asana GIDs for O(1) lookup
	existingMappings, _ := s.db.GetAllTicketMappings(userID)
	mappedGIDs := make(map[string]bool, len(existingMappings))
	for _, m := range existingMappings {
		mappedGIDs[m.AsanaTaskID] = true
	}

	// Fetch YT issues ONCE before the loop — CreateIssueWithReturn invalidates cache,
	// so fetching inside the loop would hit the live API on every iteration after the first create.
	ytIssues, _ := s.youtrackService.GetIssues(userID)
	// Build a title→ID map for O(1) duplicate detection per task
	ytTitleMap := make(map[string]string, len(ytIssues)) // normalized title -> YT issue ID
	ytSummaryMap := make(map[string]string, len(ytIssues)) // YT issue ID -> original summary
	for _, issue := range ytIssues {
		norm := normalizeTitle(issue.Summary)
		if norm != "" {
			ytTitleMap[norm] = issue.ID
			ytSummaryMap[issue.ID] = issue.Summary
		}
	}

	results := []map[string]interface{}{}
	created := 0
	skipped := 0

	for _, task := range filteredTasks {
		asanaTags := s.asanaService.GetTags(task)
		result := map[string]interface{}{
			"task_id":    task.GID,
			"task_name":  task.Name,
			"asana_tags": asanaTags,
		}

		// Skip if already mapped in DB (prevents double-create)
		if mappedGIDs[task.GID] {
			result["status"] = "skipped"
			result["reason"] = "Already mapped"
			skipped++
			results = append(results, result)
			continue
		}

		// STABILITY GUARD: skip if ticket was modified less than 10 minutes ago
		if !isTicketStable(task) {
			result["status"] = "skipped"
			result["reason"] = fmt.Sprintf("Ticket modified recently — waiting for stability (modified_at: %s)", task.ModifiedAt)
			skipped++
			results = append(results, result)
			continue
		}

		if s.ignoreService.IsIgnored(userID, task.GID) {
			result["status"] = "skipped"
			result["reason"] = "Ticket is ignored"
			skipped++
			results = append(results, result)
			continue
		}

		// Check for existing YT issue by normalized title (no API call — pre-built map)
		normTask := normalizeTitle(task.Name)
		if issueID, exists := ytTitleMap[normTask]; exists {
			s.db.CreateTicketMapping(userID, settings.AsanaProjectID, task.GID, settings.YouTrackProjectID, issueID)
			mappedGIDs[task.GID] = true
			result["status"] = "already_exists"
			result["youtrack_issue_id"] = issueID
			result["youtrack_summary"] = ytSummaryMap[issueID]
			skipped++
			results = append(results, result)
			continue
		}

		// No existing match — create it
		createdIssueID, createErr := s.youtrackService.CreateIssueWithReturn(userID, task)
		if createErr != nil {
			result["status"] = "failed"
			result["error"] = createErr.Error()
		} else {
			result["status"] = "created"
			result["youtrack_issue_id"] = createdIssueID

			_, mappingErr := s.db.CreateTicketMapping(
				userID, settings.AsanaProjectID, task.GID, settings.YouTrackProjectID, createdIssueID,
			)
			if mappingErr != nil {
				fmt.Printf("WARNING: Created ticket but failed to create mapping: %v\n", mappingErr)
			} else {
				result["mapping_created"] = true
				mappedGIDs[task.GID] = true
				// Add newly created issue to title map so subsequent tasks in same batch don't re-create it
				ytTitleMap[normalizeTitle(task.Name)] = createdIssueID
				fmt.Printf("Created mapping: Asana %s <-> YouTrack %s\n", task.GID, createdIssueID)
			}

			if len(asanaTags) > 0 {
				tagMapper := NewTagMapperForUser(userID, s.configService)
				result["mapped_subsystem"] = tagMapper.MapTagToSubsystem(asanaTags[0])
			}
			created++
		}
		results = append(results, result)
	}

	return map[string]interface{}{
		"status":  "completed",
		"created": created,
		"skipped": skipped,
		"total":   len(filteredTasks),
		"column":  columnsToProcess,
		"results": results,
	}, nil
}

// CreateSingleTicket creates a single ticket in YouTrack.
// Optimized: fetches only the one Asana task by GID, uses cached YT issues for duplicate check.
func (s *SyncService) CreateSingleTicket(userID int, taskID string) (map[string]interface{}, error) {
	// Check DB mapping first — instant skip if already created
	if _, mappingErr := s.db.GetTicketMappingByAsanaID(userID, taskID); mappingErr == nil {
		return map[string]interface{}{
			"status":  "skipped",
			"reason":  "Already mapped",
			"task_id": taskID,
		}, nil
	}

	if s.ignoreService.IsIgnored(userID, taskID) {
		return map[string]interface{}{
			"status":  "skipped",
			"reason":  "Ticket is ignored",
			"task_id": taskID,
		}, nil
	}

	// Fetch single task directly — no full project fetch
	targetTask, err := s.asanaService.GetTaskByGID(userID, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Asana task: %w", err)
	}

	asanaTags := s.asanaService.GetTags(*targetTask)

	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	// Duplicate check using cached YT issues — no extra API call
	allIssues, issErr := s.youtrackService.GetIssues(userID)
	if issErr == nil {
		for _, issue := range allIssues {
			if titlesMatch(targetTask.Name, issue.Summary) {
				// Save mapping so future calls skip instantly
				s.db.CreateTicketMapping(userID, settings.AsanaProjectID, taskID, settings.YouTrackProjectID, issue.ID)
				return map[string]interface{}{
					"status":              "already_exists",
					"reason":              "Matching YouTrack issue found",
					"task_id":             taskID,
					"task_name":           targetTask.Name,
					"youtrack_issue_id":   issue.ID,
					"youtrack_summary":    issue.Summary,
					"asana_tags":          asanaTags,
					"mapping_created":     true,
				}, nil
			}
		}
	}

	// Create issue in YouTrack
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

	_, mappingErr := s.db.CreateTicketMapping(
		userID, settings.AsanaProjectID, taskID, settings.YouTrackProjectID, createdIssueID,
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
		response["mapped_subsystem"] = tagMapper.MapTagToSubsystem(asanaTags[0])
	}

	return response, nil
}

// SyncMismatchedTickets synchronizes mismatched tickets
// Optimized: Uses DB mappings + cached Asana tasks instead of full PerformAnalysis
func (s *SyncService) SyncMismatchedTickets(userID int, requests []SyncRequest, column ...string) (map[string]interface{}, error) {
	columnInfo := "all_syncable"
	if len(column) > 0 && column[0] != "" && column[0] != "all_syncable" {
		columnInfo = column[0]
	}
	fmt.Printf("SYNC: Syncing %d tickets for column: %s (user %d) — using DB mappings, no full analysis\n", len(requests), columnInfo, userID)

	// Get all Asana tasks (cached — no API call if cache is fresh)
	allTasks, err := s.asanaService.GetTasks(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Asana tasks: %w", err)
	}

	// Build a quick lookup map: Asana GID -> AsanaTask
	taskMap := make(map[string]AsanaTask)
	for _, task := range allTasks {
		taskMap[task.GID] = task
	}

	// Pre-fetch settings and YT issues once — used in fallback paths inside the loop
	syncSettings, _ := s.configService.GetSettings(userID)
	// Lazily populated on first fallback need — avoids API call when all tickets have DB mappings
	var ytIssuesCache []YouTrackIssue
	var ytIssuesFetched bool
	getYTIssues := func() []YouTrackIssue {
		if !ytIssuesFetched {
			ytIssuesCache, _ = s.youtrackService.GetIssues(userID)
			ytIssuesFetched = true
		}
		return ytIssuesCache
	}

	results := []map[string]interface{}{}
	synced := 0

	for _, req := range requests {
		result := map[string]interface{}{
			"ticket_id": req.TicketID,
			"action":    req.Action,
		}

		switch req.Action {
		case "sync":
			if s.ignoreService.IsIgnored(userID, req.TicketID) {
				result["status"] = "skipped"
				result["reason"] = "Ticket is ignored"
				results = append(results, result)
				continue
			}

			// Look up the Asana task from cache
			asanaTask, taskExists := taskMap[req.TicketID]
			if !taskExists {
				result["status"] = "failed"
				result["error"] = "Asana task not found in cache"
				results = append(results, result)
				continue
			}

			// Look up the YouTrack issue ID — try DB mapping first, then description, then title
			var youtrackIssueID string

			// Priority 1: DB mapping (instant)
			mapping, mappingErr := s.db.GetTicketMappingByAsanaID(userID, req.TicketID)
			if mappingErr == nil {
				youtrackIssueID = mapping.YouTrackIssueID
			} else {
				// Priority 2: Search YouTrack by Asana ID in description
				// getYTIssues() fetches once and caches — safe to call multiple times
				foundID, searchErr := s.youtrackService.FindIssueByAsanaID(userID, req.TicketID)
				if searchErr == nil {
					youtrackIssueID = foundID
					if syncSettings != nil {
						s.db.CreateTicketMapping(userID, syncSettings.AsanaProjectID, req.TicketID, syncSettings.YouTrackProjectID, youtrackIssueID)
					}
				} else {
					// Priority 3: Title matching — YT issues fetched once for all tickets needing fallback
					for _, issue := range getYTIssues() {
						if titlesMatch(asanaTask.Name, issue.Summary) {
							youtrackIssueID = issue.ID
							if syncSettings != nil {
								s.db.CreateTicketMapping(userID, syncSettings.AsanaProjectID, req.TicketID, syncSettings.YouTrackProjectID, youtrackIssueID)
							}
							break
						}
					}
				}
			}

			if youtrackIssueID == "" {
				result["status"] = "failed"
				result["error"] = "Could not find matching YouTrack issue (no DB mapping, description match, or title match)"
				results = append(results, result)
				continue
			}

			// Update the YouTrack issue directly
			err = s.youtrackService.UpdateIssue(userID, youtrackIssueID, asanaTask)
			if err != nil {
				result["status"] = "failed"
				result["error"] = err.Error()
			} else {
				result["status"] = "synced"
				result["youtrack_issue_id"] = youtrackIssueID

				asanaTags := s.asanaService.GetTags(asanaTask)
				if len(asanaTags) > 0 {
					tagMapper := NewTagMapperForUser(userID, s.configService)
					primaryTag := asanaTags[0]
					mappedSubsystem := tagMapper.MapTagToSubsystem(primaryTag)
					result["tag_sync"] = map[string]interface{}{
						"asana_tags":       asanaTags,
						"mapped_subsystem": mappedSubsystem,
					}
				}
				synced++
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
		"column":  columnInfo,
		"results": results,
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

// GetSyncableTickets returns tickets that can be synced.
// Optimized: reads DB mappings — no full PerformAnalysis.
func (s *SyncService) GetSyncableTickets(userID int) (map[string]interface{}, error) {
	mappings, err := s.db.GetAllTicketMappings(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get mappings: %w", err)
	}

	syncableTickets := []map[string]interface{}{}
	for _, m := range mappings {
		if !s.ignoreService.IsIgnored(userID, m.AsanaTaskID) {
			syncableTickets = append(syncableTickets, map[string]interface{}{
				"ticket_id":         m.AsanaTaskID,
				"youtrack_issue_id": m.YouTrackIssueID,
				"type":              "mapped",
			})
		}
	}

	return map[string]interface{}{
		"syncable_tickets": syncableTickets,
		"count":            len(syncableTickets),
		"ignored_count":    s.ignoreService.CountIgnored(userID),
	}, nil
}

// GetSyncStats returns synchronization statistics.
// Optimized: reads DB mappings — no full PerformAnalysis.
func (s *SyncService) GetSyncStats(userID int) (map[string]interface{}, error) {
	mappings, err := s.db.GetAllTicketMappings(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get mappings: %w", err)
	}

	ignored := s.ignoreService.GetIgnoredTickets(userID)
	mapped := len(mappings)
	ignoredCount := len(ignored)

	var syncHealth float64 = 100.0
	if mapped > 0 {
		syncHealth = float64(mapped) / float64(mapped+ignoredCount) * 100
	}

	return map[string]interface{}{
		"mapped_tickets":        mapped,
		"ignored_tickets":       ignoredCount,
		"sync_health_percentage": syncHealth,
		"note":                  "For full analysis breakdown use GET /analyze",
	}, nil
}

// SyncTicketsByColumn syncs tickets from a specific column.
// Optimized: uses DB mappings + cached Asana tasks — no full PerformAnalysis.
func (s *SyncService) SyncTicketsByColumn(userID int, column string) (map[string]interface{}, error) {
	var columnsToProcess []string
	if column == "" || column == "all_syncable" {
		columnsToProcess = SyncableColumns
	} else {
		columnMap := map[string]string{
			"backlog":         "backlog",
			"in_progress":     "in progress",
			"dev":             "dev",
			"stage":           "stage",
			"prod":            "prod",
			"blocked":         "blocked",
			"ready_for_stage": "ready for stage",
			"findings":        "findings",
		}
		mapped, exists := columnMap[column]
		if !exists {
			return nil, fmt.Errorf("invalid column: %s", column)
		}
		columnsToProcess = []string{mapped}
	}

	// Get Asana tasks filtered to this column (cached)
	allTasks, err := s.asanaService.GetTasks(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Asana tasks: %w", err)
	}
	filteredTasks := s.asanaService.FilterTasksByColumns(allTasks, columnsToProcess)

	// Build GID set for fast lookup
	columnGIDs := make(map[string]AsanaTask, len(filteredTasks))
	for _, t := range filteredTasks {
		columnGIDs[t.GID] = t
	}

	// Build sync requests from DB mappings, restricted to this column's tasks
	var syncRequests []SyncRequest
	mappings, _ := s.db.GetAllTicketMappings(userID)
	for _, m := range mappings {
		if _, inColumn := columnGIDs[m.AsanaTaskID]; inColumn {
			if !s.ignoreService.IsIgnored(userID, m.AsanaTaskID) {
				syncRequests = append(syncRequests, SyncRequest{
					TicketID: m.AsanaTaskID,
					Action:   "sync",
				})
			}
		}
	}

	if len(syncRequests) == 0 {
		return map[string]interface{}{
			"column":  column,
			"synced":  0,
			"errors":  0,
			"total":   0,
			"results": []interface{}{},
		}, nil
	}

	result, err := s.SyncMismatchedTickets(userID, syncRequests, column)
	if err != nil {
		return nil, err
	}
	result["column"] = column
	return result, nil
}

// CreateTicketsByColumn creates missing tickets from a specific column.
// Optimized: skips full PerformAnalysis — uses DB mappings to avoid double-creates.
func (s *SyncService) CreateTicketsByColumn(userID int, column string) (map[string]interface{}, error) {
	// Delegate to CreateMissingTickets which now has the optimized path
	return s.CreateMissingTickets(userID, column)
}

// GetSyncPreview provides a preview of what would be synced.
// Optimized: checks DB mappings — no full PerformAnalysis.
func (s *SyncService) GetSyncPreview(userID int, ticketIDs []string) (map[string]interface{}, error) {
	preview := []map[string]interface{}{}

	for _, ticketID := range ticketIDs {
		item := map[string]interface{}{
			"ticket_id": ticketID,
			"ignored":   s.ignoreService.IsIgnored(userID, ticketID),
			"will_sync": !s.ignoreService.IsIgnored(userID, ticketID),
		}

		if mapping, err := s.db.GetTicketMappingByAsanaID(userID, ticketID); err == nil {
			item["youtrack_issue_id"] = mapping.YouTrackIssueID
			item["mapped"] = true
		} else {
			item["mapped"] = false
			item["note"] = "No DB mapping — will attempt description/title fallback on sync"
		}

		preview = append(preview, item)
	}

	return map[string]interface{}{
		"preview": preview,
		"total":   len(ticketIDs),
	}, nil
}

// AutoSync performs auto-sync for all mapped tickets.
// Optimized: reads DB mappings directly — no full PerformAnalysis.
func (s *SyncService) AutoSync(userID int) error {
	mappings, err := s.db.GetAllTicketMappings(userID)
	if err != nil {
		return fmt.Errorf("failed to get ticket mappings: %w", err)
	}
	if len(mappings) == 0 {
		return nil
	}

	var syncRequests []SyncRequest
	for _, m := range mappings {
		if !s.ignoreService.IsIgnored(userID, m.AsanaTaskID) {
			syncRequests = append(syncRequests, SyncRequest{
				TicketID: m.AsanaTaskID,
				Action:   "sync",
			})
		}
	}

	if len(syncRequests) == 0 {
		return nil
	}

	_, err = s.SyncMismatchedTickets(userID, syncRequests)
	if err != nil {
		return fmt.Errorf("sync operation failed: %w", err)
	}

	fmt.Printf("AUTO-SYNC: Processed %d mapped tickets for user %d\n", len(syncRequests), userID)
	return nil
}
