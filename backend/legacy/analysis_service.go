package legacy

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	configpkg "asana-youtrack-sync/config"
	"asana-youtrack-sync/database"
	"asana-youtrack-sync/utils"
)

var nonAlphanumRe = regexp.MustCompile(`[^a-z0-9\s]`)

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

// normalizeTitle normalizes a title for fuzzy matching (strips all non-alphanumeric chars)
func normalizeTitle(title string) string {
	s := strings.ToLower(title)
	s = nonAlphanumRe.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ") // collapse whitespace
}

// titlesMatch checks if two titles match using word overlap threshold.
// Exact normalized match always passes. Fuzzy requires >= 92% overlap to
// avoid false-positive dedup bindings during ticket creation.
func titlesMatch(title1, title2 string) bool {
	n1 := normalizeTitle(title1)
	n2 := normalizeTitle(title2)
	if n1 == n2 {
		return true
	}

	words1 := strings.Fields(n1)
	words2 := strings.Fields(n2)
	if len(words1) == 0 || len(words2) == 0 {
		return false
	}

	wordSet := make(map[string]bool, len(words1))
	for _, w := range words1 {
		wordSet[w] = true
	}

	overlap := 0
	for _, w := range words2 {
		if wordSet[w] {
			overlap++
		}
	}

	shorter := len(words1)
	if len(words2) < shorter {
		shorter = len(words2)
	}
	return float64(overlap)/float64(shorter) >= 0.92
}

// PerformAnalysis performs comprehensive ticket analysis for a user
func (s *AnalysisService) PerformAnalysis(userID int, selectedColumns []string) (*TicketAnalysis, error) {
	fmt.Printf("ANALYSIS: Starting analysis for user %d with columns: %v\n", userID, selectedColumns)

	// Load settings once for self-healing mapping persistence
	userSettings, settingsErr := s.configService.GetSettings(userID)

	// Step 1: Get all Asana tasks for the user
	allAsanaTasks, err := s.asanaService.GetTasks(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Asana tasks: %w", err)
	}

	// Step 2: Filter tasks by selected columns
	asanaTasks := s.asanaService.FilterTasksByColumns(allAsanaTasks, selectedColumns)

	// Step 3: Get YouTrack issues for the user
	youTrackIssues, err := s.youtrackService.GetIssues(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get YouTrack issues: %w", err)
	}

	// Step 4: Build lookup maps - PRIORITIZE MAPPING TABLE, then DESCRIPTION, then TITLE
	youTrackMap := make(map[string]YouTrackIssue)
	asanaMap := make(map[string]AsanaTask)

	// Priority 1: Build map from ticket mappings (highest priority)
	mappings, _ := s.db.GetAllTicketMappings(userID)
	mappingAsanaToYT := make(map[string]string) // asana_task_id -> youtrack_issue_id
	mappingYTToAsana := make(map[string]string) // youtrack_issue_id -> asana_task_id

	for _, mapping := range mappings {
		mappingAsanaToYT[mapping.AsanaTaskID] = mapping.YouTrackIssueID
		mappingYTToAsana[mapping.YouTrackIssueID] = mapping.AsanaTaskID
	}

	fmt.Printf("ANALYSIS: Loaded %d ticket mappings from database\n", len(mappings))

	// Priority 2 & 3: Build YouTrack map using mappings first, then description, then title matching
	usedYouTrackIssues := make(map[string]bool) // Track which YouTrack issues are already mapped

	for _, issue := range youTrackIssues {
		// Check if this YouTrack issue has a database mapping
		if asanaTaskID, hasMappingYT := mappingYTToAsana[issue.ID]; hasMappingYT {
			youTrackMap[asanaTaskID] = issue
			usedYouTrackIssues[issue.ID] = true
			fmt.Printf("ANALYSIS: Mapped YouTrack issue '%s' to Asana task '%s' via DATABASE mapping\n", issue.ID, asanaTaskID)
			continue
		}

		// Fallback to description extraction
		asanaID := s.youtrackService.ExtractAsanaID(issue)
		if asanaID != "" {
			youTrackMap[asanaID] = issue
			usedYouTrackIssues[issue.ID] = true
			fmt.Printf("ANALYSIS: Mapped YouTrack issue '%s' to Asana ID '%s' via DESCRIPTION\n", issue.ID, asanaID)
			// Self-heal: persist mapping so future analyses use DB (O(1) lookup)
			if settingsErr == nil {
				s.db.CreateTicketMapping(userID, userSettings.AsanaProjectID, asanaID, userSettings.YouTrackProjectID, issue.ID)
				fmt.Printf("ANALYSIS: Self-healed mapping via DESCRIPTION: Asana %s <-> YT %s\n", asanaID, issue.ID)
			}
		}
	}

	// Priority 3: Title-based matching for unmapped issues (NEW LOGIC)
	unmatchedYouTrackIssues := []YouTrackIssue{}
	for _, issue := range youTrackIssues {
		if !usedYouTrackIssues[issue.ID] {
			unmatchedYouTrackIssues = append(unmatchedYouTrackIssues, issue)
		}
	}

	fmt.Printf("ANALYSIS: Found %d unmatched YouTrack issues, attempting TITLE matching\n", len(unmatchedYouTrackIssues))

	titleMatchCount := 0
	for _, task := range asanaTasks {
		// Skip if already mapped
		if _, alreadyMapped := youTrackMap[task.GID]; alreadyMapped {
			continue
		}

		// Try to find matching YouTrack issue by title
		for _, issue := range unmatchedYouTrackIssues {
			if titlesMatch(task.Name, issue.Summary) {
				youTrackMap[task.GID] = issue
				usedYouTrackIssues[issue.ID] = true
				titleMatchCount++
				fmt.Printf("ANALYSIS: Mapped YouTrack issue '%s' to Asana task '%s' via TITLE matching ('%s' ≈ '%s')\n",
					issue.ID, task.GID, issue.Summary, task.Name)
				// Self-heal: persist mapping so future analyses use DB (O(1) lookup)
				if settingsErr == nil {
					s.db.CreateTicketMapping(userID, userSettings.AsanaProjectID, task.GID, userSettings.YouTrackProjectID, issue.ID)
					fmt.Printf("ANALYSIS: Self-healed mapping via TITLE: Asana %s <-> YT %s\n", task.GID, issue.ID)
				}
				break
			}
		}
	}

	fmt.Printf("ANALYSIS: Title matching found %d additional matches\n", titleMatchCount)

	// Build Asana map
	for _, task := range asanaTasks {
		asanaMap[task.GID] = task
	}

	fmt.Printf("ANALYSIS: Built YouTrack map with %d entries (DB: %d, Description: %d, Title: %d)\n",
		len(youTrackMap), len(mappings), len(youTrackMap)-len(mappings)-titleMatchCount, titleMatchCount)

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
		AlreadyExists:    []AlreadyExistsTicket{},
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
			// PRIORITY FIX: Check database mapping first
			_, hasDBMapping := mappingAsanaToYT[task.GID]
			existingIssue, existsInYouTrack := youTrackMap[task.GID]

			// If not found via DB mapping or description, try title matching
			if !existsInYouTrack && !hasDBMapping {
				// Try to find matching YouTrack issue by title
				for _, issue := range youTrackIssues {
					if !usedYouTrackIssues[issue.ID] && titlesMatch(task.Name, issue.Summary) {
						youTrackMap[task.GID] = issue
						usedYouTrackIssues[issue.ID] = true
						existingIssue = issue
						existsInYouTrack = true
						fmt.Printf("ANALYSIS: Mapped YouTrack issue '%s' to Ready for Stage task '%s' via TITLE matching ('%s' ≈ '%s')\n",
							issue.ID, task.GID, issue.Summary, task.Name)
						break
					}
				}
			}

			if existsInYouTrack {
				s.processReadyForStageTicket(userID, task, existingIssue, asanaTags, analysis)
			} else if hasDBMapping {
				// Has DB mapping but YouTrack issue not found in current fetch - treat as matched
				fmt.Printf("ANALYSIS: Task '%s' (GID: %s) has DB mapping but YouTrack issue not in current results - treating as matched\n", task.Name, task.GID)
				// Don't add to MissingYouTrack since mapping exists
			} else {
				analysis.MissingYouTrack = append(analysis.MissingYouTrack, task)
			}

			analysis.ReadyForStage = append(analysis.ReadyForStage, task)
			continue
		}

		// PRIORITY FIX: Check database mapping FIRST before checking YouTrack map
		_, hasDBMapping := mappingAsanaToYT[task.GID]
		existingIssue, existsInYouTrack := youTrackMap[task.GID]

		if existsInYouTrack {
			s.processExistingTicket(userID, task, existingIssue, asanaTags, sectionName, analysis)
		} else if hasDBMapping {
			// Has DB mapping but YouTrack issue not found in current fetch - treat as matched
			fmt.Printf("ANALYSIS: Task '%s' (GID: %s) has DB mapping but YouTrack issue not in current results - treating as matched\n", task.Name, task.GID)
			// Don't add to MissingYouTrack since mapping exists in database
		} else {
			if s.isSyncableSection(sectionName) {
				fmt.Printf("ANALYSIS: Task '%s' (GID: %s) missing in YouTrack\n", task.Name, task.GID)
				analysis.MissingYouTrack = append(analysis.MissingYouTrack, task)
			}
		}
	}

	// Step 6.5: Detect already_exists — missing tasks that have an unmapped YT issue by title
	// These are pre-existing YT issues that were never linked via mapping/description
	stillMissing := []AsanaTask{}
	for _, task := range analysis.MissingYouTrack {
		found := false
		for _, issue := range youTrackIssues {
			if usedYouTrackIssues[issue.ID] {
				continue // already mapped to another Asana task
			}
			if titlesMatch(task.Name, issue.Summary) {
				analysis.AlreadyExists = append(analysis.AlreadyExists, AlreadyExistsTicket{
					AsanaTask:     task,
					YouTrackIssue: issue,
					MatchMethod:   "title",
				})
				usedYouTrackIssues[issue.ID] = true
				found = true
				fmt.Printf("ANALYSIS: already_exists — Asana '%s' <-> YT '%s' (title match, no mapping)\n", task.GID, issue.ID)
				break
			}
		}
		if !found {
			stillMissing = append(stillMissing, task)
		}
	}
	analysis.MissingYouTrack = stillMissing

	// Step 7: Handle orphaned YouTrack issues
	s.processOrphanedIssues(allAsanaTasks, asanaTasks, youTrackIssues, analysis)

	fmt.Printf("ANALYSIS: Complete for user %d: %d matched, %d mismatched, %d missing, %d orphaned\n",
		userID, len(analysis.Matched), len(analysis.Mismatched), len(analysis.MissingYouTrack), len(analysis.OrphanedYouTrack))

	// Step 8: Populate MissingBoard — matched tickets not on the configured agile board
	if settingsErr == nil && userSettings.SyncBoardMembership && userSettings.YouTrackBoardID != "" {
		boardIDs, err := s.youtrackService.GetBoardIssueIDs(userID)
		if err != nil {
			fmt.Printf("ANALYSIS: Could not fetch board issue IDs: %v (skipping board check)\n", err)
		} else {
			for _, t := range analysis.Matched {
				if !boardIDs[t.YouTrackIssue.ID] {
					analysis.MissingBoard = append(analysis.MissingBoard, MissingBoardTicket{
						AsanaTask:     t.AsanaTask,
						YouTrackIssue: t.YouTrackIssue,
						Status:        t.Status,
					})
				}
			}
			fmt.Printf("ANALYSIS: %d matched tickets not on configured board\n", len(analysis.MissingBoard))
		}
	}

	return analysis, nil
}

// processFindings handles findings tickets and creates alerts for active YouTrack issues
func (s *AnalysisService) processFindings(task AsanaTask, youTrackMap map[string]YouTrackIssue, analysis *TicketAnalysis) {
	analysis.FindingsTickets = append(analysis.FindingsTickets, task)

	if existingIssue, exists := youTrackMap[task.GID]; exists {
		youtrackStatus := s.youtrackService.GetStatus(existingIssue)

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
func (s *AnalysisService) processReadyForStageTicket(userID int, task AsanaTask, existingIssue YouTrackIssue, asanaTags []string, analysis *TicketAnalysis) {
	if existingIssue.ID == "" {
		fmt.Printf("ANALYSIS WARNING: Ready for Stage task '%s' (GID: %s) has empty YouTrack issue ID - treating as missing\n", task.Name, task.GID)
		analysis.MissingYouTrack = append(analysis.MissingYouTrack, task)
		return
	}

	// Use the mapping function to get the expected YouTrack status for "Ready for Stage"
	expectedYouTrackStatus := s.asanaService.MapStateToYouTrackWithSettings(userID, task)
	actualYouTrackStatus := s.youtrackService.GetStatus(existingIssue)

	fmt.Printf("ANALYSIS: Processing Ready for Stage ticket '%s' - Expected YT: %s (mapped from Asana), Actual YT: %s\n",
		task.Name, expectedYouTrackStatus, actualYouTrackStatus)

	// Assignee comparison
	ytAssignee := s.youtrackService.GetAssignee(existingIssue)
	asanaAssignee := task.Assignee.Name
	var assigneeDiff *FieldDiff
	assigneeMismatch := false
	if asanaAssignee != "" {
		if !strings.EqualFold(strings.TrimSpace(asanaAssignee), strings.TrimSpace(ytAssignee)) {
			assigneeMismatch = true
			assigneeDiff = &FieldDiff{
				AsanaValue:    asanaAssignee,
				YouTrackValue: ytAssignee,
				HasDiff:       true,
			}
		}
	}

	// Case-insensitive comparison for status matching; also check assignee
	if strings.EqualFold(actualYouTrackStatus, expectedYouTrackStatus) && !assigneeMismatch {
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
			AsanaTask:        task,
			YouTrackIssue:    existingIssue,
			AsanaStatus:      expectedYouTrackStatus,
			YouTrackStatus:   actualYouTrackStatus,
			AsanaTags:        asanaTags,
			YouTrackSubsystem: "",
			TagMismatch:      false,
			AssigneeDiff:     assigneeDiff,
			AssigneeMismatch: assigneeMismatch,
		}
		analysis.Mismatched = append(analysis.Mismatched, mismatchedTicket)
	}
}

// computeDiffs computes title and description diffs between Asana and YouTrack
func computeDiffs(task AsanaTask, issue YouTrackIssue) (titleDiff *FieldDiff, descDiff *FieldDiff) {
	// Title diff
	asanaTitle := task.Name
	ytTitle := issue.Summary
	if !strings.EqualFold(normalizeTitle(asanaTitle), normalizeTitle(ytTitle)) {
		titleDiff = &FieldDiff{
			AsanaValue:    asanaTitle,
			YouTrackValue: ytTitle,
			HasDiff:       true,
		}
	}

	// Description diff — convert Asana HTML to markdown for fair comparison
	asanaDesc := task.Notes
	if task.HTMLNotes != "" {
		asanaDesc = utils.ConvertAsanaHTMLToYouTrackMarkdown(task.HTMLNotes)
	}
	ytDesc := issue.Description
	if strings.TrimSpace(asanaDesc) != strings.TrimSpace(ytDesc) && strings.TrimSpace(asanaDesc) != "" {
		descDiff = &FieldDiff{
			AsanaValue:    asanaDesc,
			YouTrackValue: ytDesc,
			HasDiff:       true,
		}
	}
	return
}

// processExistingTicket processes tickets that exist in both systems
func (s *AnalysisService) processExistingTicket(userID int, task AsanaTask, existingIssue YouTrackIssue, asanaTags []string, sectionName string, analysis *TicketAnalysis) {
	if existingIssue.ID == "" {
		fmt.Printf("ANALYSIS WARNING: Task '%s' (GID: %s) has empty YouTrack issue ID - treating as missing\n", task.Name, task.GID)
		if s.isSyncableSection(sectionName) {
			analysis.MissingYouTrack = append(analysis.MissingYouTrack, task)
		}
		return
	}

	asanaStatus := s.asanaService.MapStateToYouTrackWithSettings(userID, task)
	youtrackStatus := s.youtrackService.GetStatus(existingIssue)

	titleDiff, descDiff := computeDiffs(task, existingIssue)

	// Assignee comparison
	ytAssignee := s.youtrackService.GetAssignee(existingIssue)
	asanaAssignee := task.Assignee.Name
	var assigneeDiff *FieldDiff
	assigneeMismatch := false
	if asanaAssignee != "" {
		if !strings.EqualFold(strings.TrimSpace(asanaAssignee), strings.TrimSpace(ytAssignee)) {
			assigneeMismatch = true
			assigneeDiff = &FieldDiff{
				AsanaValue:    asanaAssignee,
				YouTrackValue: ytAssignee,
				HasDiff:       true,
			}
		}
	}

	matchedTicket := MatchedTicket{
		AsanaTask:         task,
		YouTrackIssue:     existingIssue,
		Status:            asanaStatus,
		AsanaTags:         asanaTags,
		YouTrackSubsystem: "",
		TagMismatch:       false,
		TitleDiff:         titleDiff,
		DescriptionDiff:   descDiff,
	}

	if strings.Contains(sectionName, "blocked") {
		analysis.BlockedTickets = append(analysis.BlockedTickets, matchedTicket)
		return
	}

	// Case-insensitive comparison for status matching; also check assignee
	if strings.EqualFold(asanaStatus, youtrackStatus) && !assigneeMismatch {
		analysis.Matched = append(analysis.Matched, matchedTicket)
	} else {
		mismatchedTicket := MismatchedTicket{
			AsanaTask:        task,
			YouTrackIssue:    existingIssue,
			AsanaStatus:      asanaStatus,
			YouTrackStatus:   youtrackStatus,
			AsanaTags:        asanaTags,
			YouTrackSubsystem: "",
			TagMismatch:      false,
			TitleDiff:        titleDiff,
			DescriptionDiff:  descDiff,
			AssigneeDiff:     assigneeDiff,
			AssigneeMismatch: assigneeMismatch,
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
		case "prod":
			if strings.Contains(sectionLower, "prod") {
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
			"prod":            "prod",
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
		// For display-only columns or any other column type, return all tickets from that column
		// This includes matched, mismatched, and missing tickets
		allTickets := []interface{}{}

		// Add matched tickets from this column
		for _, ticket := range analysis.Matched {
			allTickets = append(allTickets, ticket)
		}

		// Add mismatched tickets from this column
		for _, ticket := range analysis.Mismatched {
			allTickets = append(allTickets, ticket)
		}

		// Add missing tickets from this column
		for _, ticket := range analysis.MissingYouTrack {
			allTickets = append(allTickets, ticket)
		}

		return allTickets, nil
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

// Enhanced processExistingTicket (simplified - removed title/description change detection)
func (s *AnalysisService) processExistingTicketEnhanced(task AsanaTask, existingIssue YouTrackIssue, asanaTags []string, sectionName string, analysis *TicketAnalysis, userID int) {
	if existingIssue.ID == "" {
		fmt.Printf("ANALYSIS WARNING: Task '%s' (GID: %s) has empty YouTrack issue ID - treating as missing\n", task.Name, task.GID)
		if s.isSyncableSection(sectionName) {
			analysis.MissingYouTrack = append(analysis.MissingYouTrack, task)
		}
		return
	}

	asanaStatus := s.asanaService.MapStateToYouTrackWithSettings(userID, task)
	youtrackStatus := s.youtrackService.GetStatus(existingIssue)

	// Get enhanced data
	assigneeName := s.asanaService.GetAssigneeName(task)
	priority := s.asanaService.GetPriority(task, userID)
	createdAt := s.asanaService.GetCreatedAt(task)

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

	// Only check status mismatch (removed title/description comparison)
	// Use case-insensitive comparison for status
	if !strings.EqualFold(asanaStatus, youtrackStatus) {
		mismatchedTicket := MismatchedTicket{
			AsanaTask:         task,
			YouTrackIssue:     existingIssue,
			AsanaStatus:       asanaStatus,
			YouTrackStatus:    youtrackStatus,
			AsanaTags:         asanaTags,
			YouTrackSubsystem: "",
			TagMismatch:       false,
			AssigneeName:      assigneeName,
			Priority:          priority,
			CreatedAt:         createdAt,
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

// GetChangedMappings removed - title/description change detection no longer needed
