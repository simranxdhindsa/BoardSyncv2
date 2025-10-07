package legacy

import (
	"fmt"
	"strings"
	"time"

	configpkg "asana-youtrack-sync/config"
	"asana-youtrack-sync/database"
)

// AnalysisService handles comprehensive ticket analysis operations
type AnalysisService struct {
	db              *database.DB
	configService   *configpkg.Service
	asanaService    *AsanaService
	youtrackService *YouTrackService
	ignoreService   *IgnoreService
}

// NewAnalysisService creates a new analysis service with all dependencies
func NewAnalysisService(db *database.DB, configService *configpkg.Service) *AnalysisService {
	return &AnalysisService{
		db:              db,
		configService:   configService,
		asanaService:    NewAsanaService(configService),
		youtrackService: NewYouTrackService(configService),
		ignoreService:   NewIgnoreService(db, configService),
	}
}

// PerformAnalysis performs comprehensive ticket analysis for a user
func (s *AnalysisService) PerformAnalysis(userID int, selectedColumns []string) (*TicketAnalysis, error) {
	fmt.Printf("ANALYSIS: Starting analysis for user %d with columns: %v\n", userID, selectedColumns)

	// Step 1: Get all Asana tasks for the user
	allAsanaTasks, err := s.asanaService.GetTasks(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Asana tasks: %w", err)
	}

	fmt.Printf("ANALYSIS: Retrieved %d total Asana tasks for user %d\n", len(allAsanaTasks), userID)

	// Step 2: Filter tasks by selected columns
	asanaTasks := s.asanaService.FilterTasksByColumns(allAsanaTasks, selectedColumns)
	fmt.Printf("ANALYSIS: After filtering by columns %v: %d tasks remain\n", selectedColumns, len(asanaTasks))

	// Step 3: Get YouTrack issues for the user
	youTrackIssues, err := s.youtrackService.GetIssues(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get YouTrack issues: %w", err)
	}

	fmt.Printf("ANALYSIS: Retrieved %d YouTrack issues for user %d\n", len(youTrackIssues), userID)

	// Step 4: Build lookup maps - PRIORITIZE MAPPING TABLE
	youTrackMap := make(map[string]YouTrackIssue)
	asanaMap := make(map[string]AsanaTask)

	// First, build map from ticket mappings (highest priority)
	mappings, _ := s.db.GetAllTicketMappings(userID)
	mappingAsanaToYT := make(map[string]string) // asana_task_id -> youtrack_issue_id
	mappingYTToAsana := make(map[string]string) // youtrack_issue_id -> asana_task_id

	for _, mapping := range mappings {
		mappingAsanaToYT[mapping.AsanaTaskID] = mapping.YouTrackIssueID
		mappingYTToAsana[mapping.YouTrackIssueID] = mapping.AsanaTaskID
	}

	fmt.Printf("ANALYSIS: Loaded %d ticket mappings from database\n", len(mappings))

	// Build YouTrack map using mappings first, then fallback to description
	for _, issue := range youTrackIssues {
		// Check if this YouTrack issue has a mapping
		if asanaTaskID, hasMappingYT := mappingYTToAsana[issue.ID]; hasMappingYT {
			youTrackMap[asanaTaskID] = issue
			fmt.Printf("ANALYSIS: Mapped YouTrack issue '%s' to Asana task '%s' via mapping table\n", issue.ID, asanaTaskID)
			continue
		}

		// Fallback to description extraction
		asanaID := s.youtrackService.ExtractAsanaID(issue)
		if asanaID != "" {
			youTrackMap[asanaID] = issue
			fmt.Printf("ANALYSIS: Mapped YouTrack issue '%s' to Asana ID '%s' via description\n", issue.ID, asanaID)
		}
	}

	// Build Asana map
	for _, task := range asanaTasks {
		asanaMap[task.GID] = task
	}

	fmt.Printf("ANALYSIS: Built YouTrack map with %d entries\n", len(youTrackMap))

	// Step 5: Initialize analysis result structure
	analysis := &TicketAnalysis{
		SelectedColumn:   strings.Join(selectedColumns, ", "),
		Matched:          []MatchedTicket{},
		Mismatched:       []MismatchedTicket{},
		MissingYouTrack:  []AsanaTask{},
		FindingsTickets:  []AsanaTask{},
		FindingsAlerts:   []FindingsAlert{},
		ReadyForStage:    []AsanaTask{},
		BlockedTickets:   []MatchedTicket{},
		OrphanedYouTrack: []YouTrackIssue{},
		Ignored:          s.ignoreService.GetIgnoredTickets(userID),
	}

	// Step 6: Process filtered Asana tasks
	for _, task := range asanaTasks {
		if s.ignoreService.IsIgnored(userID, task.GID) {
			continue
		}

		sectionName := s.asanaService.GetSectionName(task)
		asanaTags := s.asanaService.GetTags(task)

		// Handle special columns first
		if strings.Contains(sectionName, "findings") {
			s.processFindings(task, youTrackMap, analysis)
			continue
		}

		// Handle "Ready for Stage" - sync with DEV status in YouTrack
		if strings.Contains(sectionName, "ready for stage") {
			existingIssue, existsInYouTrack := youTrackMap[task.GID]

			if existsInYouTrack {
				s.processReadyForStageTicket(task, existingIssue, asanaTags, analysis)
			} else {
				analysis.MissingYouTrack = append(analysis.MissingYouTrack, task)
			}

			analysis.ReadyForStage = append(analysis.ReadyForStage, task)
			continue
		}

		// Check if this task has a corresponding YouTrack issue
		existingIssue, existsInYouTrack := youTrackMap[task.GID]

		if existsInYouTrack {
			s.processExistingTicket(task, existingIssue, asanaTags, sectionName, analysis)
		} else {
			if s.isSyncableSection(sectionName) {
				fmt.Printf("ANALYSIS: Task '%s' (GID: %s) missing in YouTrack\n", task.Name, task.GID)
				analysis.MissingYouTrack = append(analysis.MissingYouTrack, task)
			}
		}
	}

	// Step 7: Handle orphaned YouTrack issues
	s.processOrphanedIssues(allAsanaTasks, asanaTasks, youTrackIssues, analysis)

	fmt.Printf("ANALYSIS: Complete for user %d: %d matched, %d mismatched, %d missing, %d orphaned\n",
		userID, len(analysis.Matched), len(analysis.Mismatched), len(analysis.MissingYouTrack), len(analysis.OrphanedYouTrack))

	return analysis, nil
}

// processFindings handles findings tickets and creates alerts for active YouTrack issues
func (s *AnalysisService) processFindings(task AsanaTask, youTrackMap map[string]YouTrackIssue, analysis *TicketAnalysis) {
	analysis.FindingsTickets = append(analysis.FindingsTickets, task)

	if existingIssue, exists := youTrackMap[task.GID]; exists {
		youtrackStatus := s.youtrackService.GetStatusNormalized(existingIssue)

		if IsActiveYouTrackStatus(youtrackStatus) {
			alert := FindingsAlert{
				AsanaTask:      task,
				YouTrackIssue:  existingIssue,
				YouTrackStatus: youtrackStatus,
				AlertMessage:   fmt.Sprintf("HIGH ALERT: '%s' is in Findings (Asana) but still active in YouTrack (%s)", task.Name, youtrackStatus),
			}
			analysis.FindingsAlerts = append(analysis.FindingsAlerts, alert)
		}
	}
}

// processReadyForStageTicket processes tickets in "Ready for Stage"
func (s *AnalysisService) processReadyForStageTicket(task AsanaTask, existingIssue YouTrackIssue, asanaTags []string, analysis *TicketAnalysis) {
	if existingIssue.ID == "" {
		fmt.Printf("ANALYSIS WARNING: Ready for Stage task '%s' (GID: %s) has empty YouTrack issue ID - treating as missing\n", task.Name, task.GID)
		analysis.MissingYouTrack = append(analysis.MissingYouTrack, task)
		return
	}

	expectedYouTrackStatus := "DEV"
	// Use normalized status for consistent comparison
	actualYouTrackStatus := s.youtrackService.GetStatusNormalized(existingIssue)

	fmt.Printf("ANALYSIS: Processing Ready for Stage ticket '%s' - Expected YT: %s, Actual YT: %s (normalized)\n",
		task.Name, expectedYouTrackStatus, actualYouTrackStatus)

	// Compare normalized statuses
	if normalizeStatusForComparison(actualYouTrackStatus) == normalizeStatusForComparison(expectedYouTrackStatus) {
		matchedTicket := MatchedTicket{
			AsanaTask:         task,
			YouTrackIssue:     existingIssue,
			Status:            expectedYouTrackStatus,
			AsanaTags:         asanaTags,
			YouTrackSubsystem: "",
			TagMismatch:       false,
		}
		analysis.Matched = append(analysis.Matched, matchedTicket)
	} else {
		mismatchedTicket := MismatchedTicket{
			AsanaTask:         task,
			YouTrackIssue:     existingIssue,
			AsanaStatus:       expectedYouTrackStatus,
			YouTrackStatus:    actualYouTrackStatus,
			AsanaTags:         asanaTags,
			YouTrackSubsystem: "",
			TagMismatch:       false,
		}
		analysis.Mismatched = append(analysis.Mismatched, mismatchedTicket)
	}
}

// processExistingTicket processes tickets that exist in both systems
func (s *AnalysisService) processExistingTicket(task AsanaTask, existingIssue YouTrackIssue, asanaTags []string, sectionName string, analysis *TicketAnalysis) {
	if existingIssue.ID == "" {
		fmt.Printf("ANALYSIS WARNING: Task '%s' (GID: %s) has empty YouTrack issue ID - treating as missing\n", task.Name, task.GID)
		if s.isSyncableSection(sectionName) {
			analysis.MissingYouTrack = append(analysis.MissingYouTrack, task)
		}
		return
	}

	asanaStatus := s.asanaService.MapStateToYouTrack(task)
	// Use normalized status for consistent comparison
	youtrackStatus := s.youtrackService.GetStatusNormalized(existingIssue)

	fmt.Printf("ANALYSIS: Processing existing ticket '%s' - Asana: %s, YouTrack: %s (normalized)\n", task.Name, asanaStatus, youtrackStatus)

	matchedTicket := MatchedTicket{
		AsanaTask:         task,
		YouTrackIssue:     existingIssue,
		Status:            asanaStatus,
		AsanaTags:         asanaTags,
		YouTrackSubsystem: "",
		TagMismatch:       false,
	}

	if strings.Contains(sectionName, "blocked") {
		analysis.BlockedTickets = append(analysis.BlockedTickets, matchedTicket)
		return
	}

	// Compare normalized statuses
	if normalizeStatusForComparison(asanaStatus) == normalizeStatusForComparison(youtrackStatus) {
		analysis.Matched = append(analysis.Matched, matchedTicket)
	} else {
		mismatchedTicket := MismatchedTicket{
			AsanaTask:         task,
			YouTrackIssue:     existingIssue,
			AsanaStatus:       asanaStatus,
			YouTrackStatus:    youtrackStatus,
			AsanaTags:         asanaTags,
			YouTrackSubsystem: "",
			TagMismatch:       false,
		}
		analysis.Mismatched = append(analysis.Mismatched, mismatchedTicket)
	}
}

// processOrphanedIssues handles YouTrack issues without corresponding Asana tasks
func (s *AnalysisService) processOrphanedIssues(allAsanaTasks, filteredTasks []AsanaTask, youTrackIssues []YouTrackIssue, analysis *TicketAnalysis) {
	for _, issue := range youTrackIssues {
		asanaID := s.youtrackService.ExtractAsanaID(issue)
		if asanaID == "" {
			continue
		}

		taskExists := false
		var originalTask AsanaTask

		for _, originalTask = range allAsanaTasks {
			if originalTask.GID == asanaID {
				taskExists = true
				break
			}
		}

		if taskExists {
			filteredTaskExists := false
			for _, filteredTask := range filteredTasks {
				if filteredTask.GID == asanaID {
					filteredTaskExists = true
					break
				}
			}

			if !filteredTaskExists {
				if len(originalTask.Memberships) > 0 {
					originalSectionName := strings.ToLower(originalTask.Memberships[0].Section.Name)
					if s.isSyncableSection(originalSectionName) {
						analysis.OrphanedYouTrack = append(analysis.OrphanedYouTrack, issue)
					}
				}
			}
		} else {
			analysis.OrphanedYouTrack = append(analysis.OrphanedYouTrack, issue)
		}
	}
}

// isSyncableSection checks if a section name is syncable
func (s *AnalysisService) isSyncableSection(sectionName string) bool {
	sectionLower := strings.ToLower(strings.TrimSpace(sectionName))

	for _, col := range SyncableColumns {
		colLower := strings.ToLower(col)

		switch colLower {
		case "backlog":
			if strings.Contains(sectionLower, "backlog") &&
				!strings.Contains(sectionLower, "dev") &&
				!strings.Contains(sectionLower, "stage") &&
				!strings.Contains(sectionLower, "blocked") &&
				!strings.Contains(sectionLower, "progress") {
				return true
			}
		case "in progress":
			if strings.Contains(sectionLower, "in progress") ||
				(strings.Contains(sectionLower, "progress") && !strings.Contains(sectionLower, "backlog")) {
				return true
			}
		case "dev":
			if strings.Contains(sectionLower, "dev") && !strings.Contains(sectionLower, "ready") {
				return true
			}
		case "stage":
			if strings.Contains(sectionLower, "stage") && !strings.Contains(sectionLower, "ready") {
				return true
			}
		case "blocked":
			if strings.Contains(sectionLower, "blocked") {
				return true
			}
		case "ready for stage":
			if strings.Contains(sectionLower, "ready") && strings.Contains(sectionLower, "stage") {
				return true
			}
		default:
			if strings.Contains(sectionLower, colLower) {
				return true
			}
		}
	}
	return false
}

// normalizeStatusForComparison normalizes status strings for comparison
// This ensures "Backlog" matches "Backlog", "DEV" matches "DEV", etc.
func normalizeStatusForComparison(status string) string {
	statusLower := strings.ToLower(strings.TrimSpace(status))

	// Map to normalized values
	statusMap := map[string]string{
		"backlog":     "backlog",
		"open":        "backlog",
		"to do":       "backlog",
		"todo":        "backlog",
		"in progress": "in_progress",
		"inprogress":  "in_progress",
		"in-progress": "in_progress",
		"dev":         "dev",
		"development": "dev",
		"in dev":      "dev",
		"stage":       "stage",
		"staging":     "stage",
		"in stage":    "stage",
		"blocked":     "blocked",
		"on hold":     "blocked",
	}

	if normalized, exists := statusMap[statusLower]; exists {
		return normalized
	}

	return statusLower
}

// GetTicketsByType returns tickets of a specific type
func (s *AnalysisService) GetTicketsByType(userID int, ticketType string, column string) (interface{}, error) {
	if ticketType == "ignored" {
		return s.ignoreService.GetIgnoredTickets(userID), nil
	}

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
			fmt.Printf("ANALYSIS: Unknown column '%s', using all syncable columns\n", column)
			columnsToAnalyze = SyncableColumns
		}
	}

	analysis, err := s.PerformAnalysis(userID, columnsToAnalyze)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	switch ticketType {
	case "matched":
		return analysis.Matched, nil
	case "mismatched":
		return analysis.Mismatched, nil
	case "missing":
		return analysis.MissingYouTrack, nil
	case "findings":
		return analysis.FindingsTickets, nil
	case "ready_for_stage":
		return analysis.ReadyForStage, nil
	case "blocked":
		return analysis.BlockedTickets, nil
	case "orphaned":
		return analysis.OrphanedYouTrack, nil
	default:
		return nil, fmt.Errorf("invalid ticket type: %s", ticketType)
	}
}

// GetAnalysisSummary returns a comprehensive summary
func (s *AnalysisService) GetAnalysisSummary(userID int, selectedColumns []string) (map[string]interface{}, error) {
	analysis, err := s.PerformAnalysis(userID, selectedColumns)
	if err != nil {
		return nil, err
	}

	tagMismatchCount := 0
	statusMismatchCount := 0

	for _, ticket := range analysis.Mismatched {
		if ticket.TagMismatch {
			tagMismatchCount++
		}
		if ticket.AsanaStatus != ticket.YouTrackStatus {
			statusMismatchCount++
		}
	}

	totalTickets := len(analysis.Matched) + len(analysis.Mismatched) + len(analysis.MissingYouTrack)
	syncHealthPercentage := 100.0
	if totalTickets > 0 {
		matchedCount := len(analysis.Matched)
		syncHealthPercentage = float64(matchedCount) / float64(totalTickets) * 100
	}

	return map[string]interface{}{
		"matched":             len(analysis.Matched),
		"mismatched":          len(analysis.Mismatched),
		"missing_youtrack":    len(analysis.MissingYouTrack),
		"findings_tickets":    len(analysis.FindingsTickets),
		"findings_alerts":     len(analysis.FindingsAlerts),
		"ready_for_stage":     len(analysis.ReadyForStage),
		"blocked_tickets":     len(analysis.BlockedTickets),
		"orphaned_youtrack":   len(analysis.OrphanedYouTrack),
		"ignored":             len(analysis.Ignored),
		"tag_mismatches":      tagMismatchCount,
		"status_mismatches":   statusMismatchCount,
		"total_tickets":       totalTickets,
		"sync_health_percent": syncHealthPercentage,
	}, nil
}

// GetDetailedAnalysis returns detailed analysis with breakdowns
func (s *AnalysisService) GetDetailedAnalysis(userID int, selectedColumns []string) (map[string]interface{}, error) {
	analysis, err := s.PerformAnalysis(userID, selectedColumns)
	if err != nil {
		return nil, err
	}

	statusBreakdown := make(map[string]int)
	for _, ticket := range analysis.Mismatched {
		key := fmt.Sprintf("%s -> %s", ticket.YouTrackStatus, ticket.AsanaStatus)
		statusBreakdown[key]++
	}

	columnBreakdown := make(map[string]int)
	for _, task := range analysis.MissingYouTrack {
		sectionName := s.asanaService.GetSectionName(task)
		columnBreakdown[sectionName]++
	}

	tagAnalysis := s.analyzeTagUsage(analysis)

	return map[string]interface{}{
		"analysis":           analysis,
		"status_breakdown":   statusBreakdown,
		"column_breakdown":   columnBreakdown,
		"tag_analysis":       tagAnalysis,
		"selected_columns":   selectedColumns,
		"analysis_timestamp": fmt.Sprintf("%d", time.Now().Unix()),
	}, nil
}

// analyzeTagUsage analyzes tag usage across tickets
func (s *AnalysisService) analyzeTagUsage(analysis *TicketAnalysis) map[string]interface{} {
	tagCounts := make(map[string]int)
	totalTaggedTickets := 0

	allTasks := []AsanaTask{}

	for _, ticket := range analysis.Matched {
		allTasks = append(allTasks, ticket.AsanaTask)
	}

	for _, ticket := range analysis.Mismatched {
		allTasks = append(allTasks, ticket.AsanaTask)
	}

	allTasks = append(allTasks, analysis.MissingYouTrack...)

	for _, task := range allTasks {
		tags := s.asanaService.GetTags(task)
		if len(tags) > 0 {
			totalTaggedTickets++
			for _, tag := range tags {
				tagCounts[tag]++
			}
		}
	}

	return map[string]interface{}{
		"tag_counts":           tagCounts,
		"total_tagged_tickets": totalTaggedTickets,
		"total_tickets":        len(allTasks),
		"tag_coverage":         float64(totalTaggedTickets) / float64(len(allTasks)) * 100,
	}
}

// GetColumnAnalysis returns analysis for a specific column
func (s *AnalysisService) GetColumnAnalysis(userID int, column string) (map[string]interface{}, error) {
	if column == "" {
		return nil, fmt.Errorf("column parameter is required")
	}

	analysis, err := s.PerformAnalysis(userID, []string{column})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"column":           column,
		"matched":          len(analysis.Matched),
		"mismatched":       len(analysis.Mismatched),
		"missing_youtrack": len(analysis.MissingYouTrack),
		"tickets":          analysis,
	}, nil
}

// ValidateAnalysisRequest validates analysis request parameters
func (s *AnalysisService) ValidateAnalysisRequest(userID int, columns []string) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user ID")
	}

	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return fmt.Errorf("failed to get user settings: %w", err)
	}

	if settings.AsanaPAT == "" || settings.AsanaProjectID == "" {
		return fmt.Errorf("Asana configuration incomplete")
	}

	if settings.YouTrackBaseURL == "" || settings.YouTrackToken == "" || settings.YouTrackProjectID == "" {
		return fmt.Errorf("YouTrack configuration incomplete")
	}

	if len(columns) > 0 {
		validColumns := make(map[string]bool)
		for _, col := range AllColumns {
			validColumns[col] = true
		}

		for _, col := range columns {
			if !validColumns[col] {
				return fmt.Errorf("invalid column: %s", col)
			}
		}
	}

	return nil
}

// GetAnalysisHealth returns the health status of the analysis system
func (s *AnalysisService) GetAnalysisHealth(userID int) (map[string]interface{}, error) {
	asanaHealth := "healthy"
	youtrackHealth := "healthy"

	_, err := s.asanaService.GetTasks(userID)
	if err != nil {
		asanaHealth = "unhealthy: " + err.Error()
	}

	_, err = s.youtrackService.GetIssues(userID)
	if err != nil {
		youtrackHealth = "unhealthy: " + err.Error()
	}

	overallHealth := "healthy"
	if asanaHealth != "healthy" || youtrackHealth != "healthy" {
		overallHealth = "unhealthy"
	}

	return map[string]interface{}{
		"overall_health":  overallHealth,
		"asana_health":    asanaHealth,
		"youtrack_health": youtrackHealth,
		"ignored_count":   s.ignoreService.CountIgnored(userID),
		"timestamp":       time.Now().Format(time.RFC3339),
	}, nil
}

// PerformAnalysisWithFiltering performs analysis with filtering and sorting
func (s *AnalysisService) PerformAnalysisWithFiltering(userID int, selectedColumns []string, filter TicketFilter, sortOpts TicketSortOptions) (*TicketAnalysis, error) {
	fmt.Printf("ANALYSIS: Starting analysis for user %d with columns: %v, filter: %+v, sort: %+v\n", userID, selectedColumns, filter, sortOpts)

	// Perform base analysis
	analysis, err := s.PerformAnalysis(userID, selectedColumns)
	if err != nil {
		return nil, err
	}

	// Apply filters
	analysis.Matched = FilterMatchedTickets(analysis.Matched, filter)
	analysis.Mismatched = FilterMismatchedTickets(analysis.Mismatched, filter)
	analysis.MissingYouTrack = FilterAsanaTasks(analysis.MissingYouTrack, filter, s.asanaService, userID)

	// Apply sorting
	analysis.Matched = SortMatchedTickets(analysis.Matched, sortOpts)
	analysis.Mismatched = SortMismatchedTickets(analysis.Mismatched, sortOpts)
	analysis.MissingYouTrack = SortAsanaTasks(analysis.MissingYouTrack, sortOpts, s.asanaService, userID)

	return analysis, nil
}

// Enhanced processExistingTicket with change detection
func (s *AnalysisService) processExistingTicketEnhanced(task AsanaTask, existingIssue YouTrackIssue, asanaTags []string, sectionName string, analysis *TicketAnalysis, userID int) {
	if existingIssue.ID == "" {
		fmt.Printf("ANALYSIS WARNING: Task '%s' (GID: %s) has empty YouTrack issue ID - treating as missing\n", task.Name, task.GID)
		if s.isSyncableSection(sectionName) {
			analysis.MissingYouTrack = append(analysis.MissingYouTrack, task)
		}
		return
	}

	asanaStatus := s.asanaService.MapStateToYouTrack(task)
	// Use normalized status for consistent comparison
	youtrackStatus := s.youtrackService.GetStatusNormalized(existingIssue)

	// Get enhanced data
	assigneeName := s.asanaService.GetAssigneeName(task)
	priority := s.asanaService.GetPriority(task, userID)
	createdAt := s.asanaService.GetCreatedAt(task)

	// Check for title/description changes
	comparisonService := NewComparisonService(s.db, s.configService)
	changes := comparisonService.CompareTickets(task, existingIssue)

	fmt.Printf("ANALYSIS: Processing existing ticket '%s' - Asana: %s, YouTrack: %s (normalized), Changes: %+v\n", task.Name, asanaStatus, youtrackStatus, changes)

	matchedTicket := MatchedTicket{
		AsanaTask:         task,
		YouTrackIssue:     existingIssue,
		Status:            asanaStatus,
		AsanaTags:         asanaTags,
		YouTrackSubsystem: "",
		TagMismatch:       false,
		AssigneeName:      assigneeName,
		Priority:          priority,
		CreatedAt:         createdAt,
	}

	if strings.Contains(sectionName, "blocked") {
		analysis.BlockedTickets = append(analysis.BlockedTickets, matchedTicket)
		return
	}

	// If there are title/description changes OR status mismatch, add to mismatched
	normalizedAsana := normalizeStatusForComparison(asanaStatus)
	normalizedYT := normalizeStatusForComparison(youtrackStatus)

	if normalizedAsana != normalizedYT || changes.HasAnyChanges() {
		mismatchedTicket := MismatchedTicket{
			AsanaTask:           task,
			YouTrackIssue:       existingIssue,
			AsanaStatus:         asanaStatus,
			YouTrackStatus:      youtrackStatus,
			AsanaTags:           asanaTags,
			YouTrackSubsystem:   "",
			TagMismatch:         false,
			AssigneeName:        assigneeName,
			Priority:            priority,
			CreatedAt:           createdAt,
			TitleMismatch:       changes.HasTitleChange,
			DescriptionMismatch: changes.HasDescriptionChange,
		}
		analysis.Mismatched = append(analysis.Mismatched, mismatchedTicket)
	} else {
		analysis.Matched = append(analysis.Matched, matchedTicket)
	}
}

// GetFilterOptions returns available filter options for the analysis
func (s *AnalysisService) GetFilterOptions(userID int, selectedColumns []string) (map[string]interface{}, error) {
	analysis, err := s.PerformAnalysis(userID, selectedColumns)
	if err != nil {
		return nil, err
	}

	assignees := GetUniqueAssignees(analysis.Matched, analysis.Mismatched, analysis.MissingYouTrack, s.asanaService)
	priorities := GetUniquePriorities(analysis.Matched, analysis.Mismatched, analysis.MissingYouTrack, s.asanaService, userID)

	// Get date range
	var minDate, maxDate time.Time
	allTasks := []AsanaTask{}

	for _, ticket := range analysis.Matched {
		allTasks = append(allTasks, ticket.AsanaTask)
	}
	for _, ticket := range analysis.Mismatched {
		allTasks = append(allTasks, ticket.AsanaTask)
	}
	allTasks = append(allTasks, analysis.MissingYouTrack...)

	for _, task := range allTasks {
		createdAt := s.asanaService.GetCreatedAt(task)
		if !createdAt.IsZero() {
			if minDate.IsZero() || createdAt.Before(minDate) {
				minDate = createdAt
			}
			if maxDate.IsZero() || createdAt.After(maxDate) {
				maxDate = createdAt
			}
		}
	}

	return map[string]interface{}{
		"assignees":  assignees,
		"priorities": priorities,
		"date_range": map[string]string{
			"min": minDate.Format("2006-01-02"),
			"max": maxDate.Format("2006-01-02"),
		},
	}, nil
}

// GetChangedMappings returns all ticket mappings that have title/description changes
func (s *AnalysisService) GetChangedMappings(userID int) ([]MappingChangeInfo, error) {
	comparisonService := NewComparisonService(s.db, s.configService)
	return comparisonService.CheckMappingChanges(userID)
}


//DEBUG
// Add this to backend/legacy/analysis_service.go

// VerifyColumnsAndMapping returns detailed information about columns and mapping
func (s *AnalysisService) VerifyColumnsAndMapping(userID int) (map[string]interface{}, error) {
	fmt.Printf("VERIFY: Starting column verification for user %d\n", userID)

	// Get all Asana tasks
	asanaTasks, err := s.asanaService.GetTasks(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Asana tasks: %w", err)
	}

	// Get all YouTrack issues
	youtrackIssues, err := s.youtrackService.GetIssues(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get YouTrack issues: %w", err)
	}

	// Analyze Asana columns
	asanaColumns := make(map[string][]map[string]interface{})
	asanaColumnCounts := make(map[string]int)

	for _, task := range asanaTasks {
		sectionName := s.asanaService.GetSectionName(task)
		asanaColumnCounts[sectionName]++

		taskInfo := map[string]interface{}{
			"gid":          task.GID,
			"name":         task.Name,
			"section":      sectionName,
			"mapped_to_yt": s.asanaService.MapStateToYouTrack(task),
		}

		asanaColumns[sectionName] = append(asanaColumns[sectionName], taskInfo)
	}

	// Analyze YouTrack states/columns
	youtrackStates := make(map[string][]map[string]interface{})
	youtrackStateCounts := make(map[string]int)
	youtrackStateDetails := make(map[string]map[string]interface{})

	for _, issue := range youtrackIssues {
		// Get both raw and normalized status
		rawStatus := s.youtrackService.GetStatus(issue)
		normalizedStatus := s.youtrackService.GetStatusNormalized(issue)

		youtrackStateCounts[normalizedStatus]++

		// Store detailed state information
		if _, exists := youtrackStateDetails[normalizedStatus]; !exists {
			youtrackStateDetails[normalizedStatus] = map[string]interface{}{
				"raw_status":        rawStatus,
				"normalized_status": normalizedStatus,
				"sample_issue_id":   issue.ID,
			}
		}

		issueInfo := map[string]interface{}{
			"id":                issue.ID,
			"summary":           issue.Summary,
			"raw_status":        rawStatus,
			"normalized_status": normalizedStatus,
			"asana_id":          s.youtrackService.ExtractAsanaID(issue),
		}

		youtrackStates[normalizedStatus] = append(youtrackStates[normalizedStatus], issueInfo)
	}

	// Create mapping matrix showing how Asana columns map to YouTrack states
	mappingMatrix := []map[string]interface{}{}

	for asanaSection, count := range asanaColumnCounts {
		mappedYTStatus := s.asanaService.MapStateToYouTrack(AsanaTask{
			Memberships: []struct {
				Section struct {
					GID  string `json:"gid"`
					Name string `json:"name"`
				} `json:"section"`
			}{
				{
					Section: struct {
						GID  string `json:"gid"`
						Name string `json:"name"`
					}{
						Name: asanaSection,
					},
				},
			},
		})

		normalizedMapped := normalizeStatusForComparison(mappedYTStatus)
		ytCount := youtrackStateCounts[mappedYTStatus]

		mappingMatrix = append(mappingMatrix, map[string]interface{}{
			"asana_section":         asanaSection,
			"asana_task_count":      count,
			"maps_to_yt_status":     mappedYTStatus,
			"normalized_yt_status":  normalizedMapped,
			"yt_issue_count":        ytCount,
			"is_syncable":           s.isSyncableSection(asanaSection),
			"match_found":           ytCount > 0,
		})
	}

	// Analyze mappings
	mappings, _ := s.db.GetAllTicketMappings(userID)
	mappingDetails := []map[string]interface{}{}

	existingYTIssues := make(map[string]YouTrackIssue)
	for _, issue := range youtrackIssues {
		existingYTIssues[issue.ID] = issue
	}

	existingAsanaTasks := make(map[string]AsanaTask)
	for _, task := range asanaTasks {
		existingAsanaTasks[task.GID] = task
	}

	validMappings := 0
	invalidMappings := 0

	for _, mapping := range mappings {
		asanaTask, asanaExists := existingAsanaTasks[mapping.AsanaTaskID]
		ytIssue, ytExists := existingYTIssues[mapping.YouTrackIssueID]

		status := "valid"
		issues := []string{}

		if !asanaExists {
			status = "invalid"
			issues = append(issues, "Asana task not found")
			invalidMappings++
		}
		if !ytExists {
			status = "invalid"
			issues = append(issues, "YouTrack issue not found")
			invalidMappings++
		}

		if status == "valid" {
			validMappings++
		}

		mappingInfo := map[string]interface{}{
			"mapping_id":        mapping.ID,
			"asana_task_id":     mapping.AsanaTaskID,
			"youtrack_issue_id": mapping.YouTrackIssueID,
			"status":            status,
			"issues":            issues,
		}

		if asanaExists {
			mappingInfo["asana_task_name"] = asanaTask.Name
			mappingInfo["asana_section"] = s.asanaService.GetSectionName(asanaTask)
		}

		if ytExists {
			mappingInfo["youtrack_summary"] = ytIssue.Summary
			mappingInfo["youtrack_status"] = s.youtrackService.GetStatusNormalized(ytIssue)
		}

		mappingDetails = append(mappingDetails, mappingInfo)
	}

	// Get all unique YouTrack State field values with their raw structure
	stateFieldAnalysis := s.analyzeYouTrackStateField(youtrackIssues)

	result := map[string]interface{}{
		"summary": map[string]interface{}{
			"total_asana_tasks":     len(asanaTasks),
			"total_youtrack_issues": len(youtrackIssues),
			"asana_sections_count":  len(asanaColumnCounts),
			"youtrack_states_count": len(youtrackStateCounts),
			"total_mappings":        len(mappings),
			"valid_mappings":        validMappings,
			"invalid_mappings":      invalidMappings,
		},
		"asana_columns": map[string]interface{}{
			"columns":       asanaColumns,
			"column_counts": asanaColumnCounts,
		},
		"youtrack_states": map[string]interface{}{
			"states":        youtrackStates,
			"state_counts":  youtrackStateCounts,
			"state_details": youtrackStateDetails,
		},
		"mapping_matrix":       mappingMatrix,
		"database_mappings":    mappingDetails,
		"state_field_analysis": stateFieldAnalysis,
		"normalization_rules": map[string]string{
			"backlog":     "Backlog, Open, To Do, Todo",
			"in_progress": "In Progress, InProgress, In-Progress",
			"dev":         "DEV, Development, In Dev",
			"stage":       "STAGE, Staging, In Stage",
			"blocked":     "Blocked, On Hold",
		},
	}

	fmt.Printf("VERIFY: Completed column verification for user %d\n", userID)
	return result, nil
}

// analyzeYouTrackStateField analyzes the raw State field structure from YouTrack
func (s *AnalysisService) analyzeYouTrackStateField(youtrackIssues []YouTrackIssue) map[string]interface{} {
	stateFieldSamples := []map[string]interface{}{}
	uniqueStates := make(map[string]map[string]interface{})

	for i, issue := range youtrackIssues {
		if i >= 10 { // Only analyze first 10 for samples
			break
		}

		for _, field := range issue.CustomFields {
			if field.Name == "State" {
				rawStatus := s.youtrackService.GetStatus(issue)
				normalizedStatus := s.youtrackService.GetStatusNormalized(issue)

				sample := map[string]interface{}{
					"issue_id":          issue.ID,
					"raw_field_value":   field.Value,
					"extracted_status":  rawStatus,
					"normalized_status": normalizedStatus,
				}

				stateFieldSamples = append(stateFieldSamples, sample)

				// Track unique states
				if _, exists := uniqueStates[normalizedStatus]; !exists {
					uniqueStates[normalizedStatus] = map[string]interface{}{
						"raw_status":        rawStatus,
						"normalized_status": normalizedStatus,
						"raw_field_value":   field.Value,
					}
				}
			}
		}
	}

	return map[string]interface{}{
		"samples":       stateFieldSamples,
		"unique_states": uniqueStates,
		"note":          "Shows raw State field structure from YouTrack API",
	}
}

// GetColumnMappingReport generates a detailed report of column mappings
func (s *AnalysisService) GetColumnMappingReport(userID int) (map[string]interface{}, error) {
	verification, err := s.VerifyColumnsAndMapping(userID)
	if err != nil {
		return nil, err
	}

	// Create a simplified, human-readable report
	mappingMatrix := verification["mapping_matrix"].([]map[string]interface{})
	
	report := []string{}
	report = append(report, "=== ASANA TO YOUTRACK COLUMN MAPPING ===\n")

	for _, mapping := range mappingMatrix {
		asanaSection := mapping["asana_section"].(string)
		asanaCount := mapping["asana_task_count"].(int)
		mapsToYT := mapping["maps_to_yt_status"].(string)
		ytCount := mapping["yt_issue_count"].(int)
		isSyncable := mapping["is_syncable"].(bool)
		matchFound := mapping["match_found"].(bool)

		syncableStr := "❌ DISPLAY ONLY"
		if isSyncable {
			syncableStr = "✅ SYNCABLE"
		}

		matchStr := "⚠️  NO MATCH"
		if matchFound {
			matchStr = "✓ MATCH FOUND"
		}

		line := fmt.Sprintf("Asana: '%s' (%d tasks) → YouTrack: '%s' (%d issues) [%s] %s",
			asanaSection, asanaCount, mapsToYT, ytCount, syncableStr, matchStr)
		report = append(report, line)
	}

	return map[string]interface{}{
		"report":       strings.Join(report, "\n"),
		"full_details": verification,
	}, nil
}