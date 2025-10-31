package legacy

import (
	"fmt"
	"log"
	"strings"

	configpkg "asana-youtrack-sync/config"
	"asana-youtrack-sync/database"
)

type ReverseAnalysisService struct {
	db              *database.DB
	youtrackService *YouTrackService
	asanaService    *AsanaService
	configService   *configpkg.Service
}

func NewReverseAnalysisService(db *database.DB, youtrackService *YouTrackService, asanaService *AsanaService, configService *configpkg.Service) *ReverseAnalysisService {
	return &ReverseAnalysisService{
		db:              db,
		youtrackService: youtrackService,
		asanaService:    asanaService,
		configService:   configService,
	}
}

// PerformReverseAnalysis analyzes YouTrack tickets and categorizes them as Matched or Missing in Asana
func (s *ReverseAnalysisService) PerformReverseAnalysis(userID int, creatorFilter string) (*ReverseTicketAnalysis, error) {
	log.Printf("[Reverse Analysis] Starting analysis for userID: %d, creator filter: %s", userID, creatorFilter)

	analysis := &ReverseTicketAnalysis{
		Matched:      []ReverseMatchedTicket{},
		MissingAsana: []YouTrackIssue{},
	}

	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	// 1. Fetch YouTrack issues with creator filter
	ytIssues, err := s.youtrackService.GetIssuesByCreator(userID, creatorFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch YouTrack issues: %w", err)
	}
	log.Printf("[Reverse Analysis] Found %d YouTrack issues created by '%s'", len(ytIssues), creatorFilter)

	// 2. Fetch existing ticket mappings from database
	mappings, err := s.db.GetAllTicketMappings(userID)
	if err != nil {
		log.Printf("[Reverse Analysis] Warning: Failed to get ticket mappings: %v", err)
		mappings = []*database.TicketMapping{}
	}

	// Create a map for quick lookup: YouTrackIssueID -> AsanaTaskID
	ytToAsanaMap := make(map[string]string)
	for _, mapping := range mappings {
		ytToAsanaMap[mapping.YouTrackIssueID] = mapping.AsanaTaskID
	}
	log.Printf("[Reverse Analysis] Loaded %d existing mappings from database", len(mappings))

	// 3. Fetch all Asana tasks to verify existence and check titles
	asanaTasks, err := s.asanaService.GetAllTasks(userID, settings.AsanaProjectID)
	if err != nil {
		log.Printf("[Reverse Analysis] Warning: Failed to fetch Asana tasks: %v", err)
		asanaTasks = []AsanaTask{}
	}
	log.Printf("[Reverse Analysis] Found %d Asana tasks", len(asanaTasks))

	// Create maps for lookup:
	// 1. By task ID (from database mappings)
	asanaTaskSet := make(map[string]bool)
	// 2. By YouTrack ID in title (e.g., "ARD-123" in "ARD-123 Fix bug")
	asanaTaskByYTID := make(map[string]string)

	for _, task := range asanaTasks {
		asanaTaskSet[task.GID] = true

		// Check if task name starts with a YouTrack ID pattern (e.g., "ARD-123")
		if len(task.Name) > 0 {
			// Extract potential YouTrack ID from the beginning of the title
			// Pattern: PROJECT-NUMBER (e.g., "ARD-123", "PROJ-456")
			parts := strings.Fields(task.Name)
			if len(parts) > 0 {
				firstPart := parts[0]
				// Check if it matches pattern: LETTERS-NUMBERS
				if strings.Contains(firstPart, "-") {
					asanaTaskByYTID[firstPart] = task.GID
				}
			}
		}
	}

	// 4. Categorize each YouTrack issue
	for _, ytIssue := range ytIssues {
		asanaTaskID := ""
		matchReason := ""

		// Priority 1: Check database mappings
		if mappedTaskID, hasMapping := ytToAsanaMap[ytIssue.ID]; hasMapping && asanaTaskSet[mappedTaskID] {
			asanaTaskID = mappedTaskID
			matchReason = "database mapping"
		} else if taskID, foundInTitle := asanaTaskByYTID[ytIssue.ID]; foundInTitle {
			// Priority 2: Check if YouTrack ID exists in Asana task title
			asanaTaskID = taskID
			matchReason = "title contains ID"
		}

		if asanaTaskID != "" {
			// Ticket exists in both systems
			analysis.Matched = append(analysis.Matched, ReverseMatchedTicket{
				YouTrackIssue: ytIssue,
				AsanaTaskID:   asanaTaskID,
			})
			log.Printf("[Reverse Analysis] Matched: %s <-> %s (via %s)", ytIssue.ID, asanaTaskID, matchReason)
		} else {
			// Ticket exists in YouTrack but not in Asana
			analysis.MissingAsana = append(analysis.MissingAsana, ytIssue)
			log.Printf("[Reverse Analysis] Missing in Asana: %s - %s", ytIssue.ID, ytIssue.Summary)
		}
	}

	log.Printf("[Reverse Analysis] Analysis complete - Matched: %d, Missing in Asana: %d",
		len(analysis.Matched), len(analysis.MissingAsana))

	return analysis, nil
}

// Additional helper methods can be added here as needed
