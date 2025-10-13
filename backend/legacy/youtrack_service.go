package legacy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"asana-youtrack-sync/config"
	"asana-youtrack-sync/database"
	"asana-youtrack-sync/utils"
)

// YouTrackService handles YouTrack API operations with user-specific settings
type YouTrackService struct {
	configService *config.Service
}

// NewYouTrackService creates a new YouTrack service
func NewYouTrackService(configService *config.Service) *YouTrackService {
	return &YouTrackService{
		configService: configService,
	}
}

// GetIssues retrieves issues from YouTrack using user settings
func (s *YouTrackService) GetIssues(userID int) ([]YouTrackIssue, error) {
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user settings: %w", err)
	}

	if settings.YouTrackBaseURL == "" || settings.YouTrackToken == "" || settings.YouTrackProjectID == "" {
		return nil, fmt.Errorf("youtrack credentials not configured")
	}

	fmt.Printf("Getting YouTrack issues for user %d from project: %s\n", userID, settings.YouTrackProjectID)

	// Try multiple approaches to get issues
	approaches := []func(*config.UserSettings) ([]YouTrackIssue, error){
		s.getIssuesWithProjectKey,
		s.getIssuesWithQuery,
		s.getIssuesSimpleCloud,
		s.getIssuesViaProjects,
	}

	for i, approach := range approaches {
		fmt.Printf("Attempting approach %d...\n", i+1)
		issues, err := approach(settings)
		if err == nil && len(issues) >= 0 {
			fmt.Printf("Approach %d succeeded! Found %d issues for user %d\n", i+1, len(issues), userID)
			return issues, nil
		}
		fmt.Printf("Approach %d failed: %v\n", i+1, err)
	}

	return nil, fmt.Errorf("all approaches failed to connect to YouTrack")
}

// getIssuesWithProjectKey tries direct project key approach
func (s *YouTrackService) getIssuesWithProjectKey(settings *config.UserSettings) ([]YouTrackIssue, error) {
	query := fmt.Sprintf("project: {%s}", settings.YouTrackProjectID)
	fields := "id,idReadable,summary,description,created,updated,customFields(name,value(name,localizedName,description,id,$type,color)),project(shortName)"

	encodedQuery := strings.ReplaceAll(query, " ", "%20")
	encodedQuery = strings.ReplaceAll(encodedQuery, "{", "%7B")
	encodedQuery = strings.ReplaceAll(encodedQuery, "}", "%7D")

	url := fmt.Sprintf("%s/api/issues?fields=%s&query=%s&top=200",
		settings.YouTrackBaseURL, fields, encodedQuery)

	return s.makeRequest(settings, url)
}

// getIssuesWithQuery tries query-based approach
func (s *YouTrackService) getIssuesWithQuery(settings *config.UserSettings) ([]YouTrackIssue, error) {
	queries := []string{
		fmt.Sprintf("project: {%s}", settings.YouTrackProjectID),
		fmt.Sprintf("project:%s", settings.YouTrackProjectID),
		fmt.Sprintf("project: %s", settings.YouTrackProjectID),
		fmt.Sprintf("#%s", settings.YouTrackProjectID),
	}

	fields := "id,idReadable,summary,description,created,updated,customFields(name,value(name,localizedName,description,id,$type,color)),project(shortName)"

	for _, query := range queries {
		encodedQuery := strings.ReplaceAll(query, " ", "%20")
		if strings.Contains(query, "{") {
			encodedQuery = strings.ReplaceAll(encodedQuery, "{", "%7B")
			encodedQuery = strings.ReplaceAll(encodedQuery, "}", "%7D")
		}

		url := fmt.Sprintf("%s/api/issues?fields=%s&query=%s&top=200",
			settings.YouTrackBaseURL, fields, encodedQuery)

		if issues, err := s.makeRequest(settings, url); err == nil {
			return issues, nil
		}
	}

	return nil, fmt.Errorf("all query formats failed")
}

// getIssuesSimpleCloud tries simple issues endpoint
func (s *YouTrackService) getIssuesSimpleCloud(settings *config.UserSettings) ([]YouTrackIssue, error) {
	url := fmt.Sprintf("%s/api/issues?fields=id,idReadable,summary,description,created,updated,customFields(name,value(name,localizedName,description,id,$type)),project(shortName)&top=200",
		settings.YouTrackBaseURL)

	allIssues, err := s.makeRequest(settings, url)
	if err != nil {
		return nil, err
	}

	var projectIssues []YouTrackIssue
	for _, issue := range allIssues {
		if issue.Project.ShortName == settings.YouTrackProjectID {
			projectIssues = append(projectIssues, issue)
		}
	}

	fmt.Printf("Filtered %d issues for project %s\n", len(projectIssues), settings.YouTrackProjectID)
	return projectIssues, nil
}

// getIssuesViaProjects tries project-specific endpoint
func (s *YouTrackService) getIssuesViaProjects(settings *config.UserSettings) ([]YouTrackIssue, error) {
	urls := []string{
		fmt.Sprintf("%s/api/admin/projects/%s/issues?fields=id,idReadable,summary,description,created,updated,customFields(name,value(name,localizedName)),project(shortName)&top=200",
			settings.YouTrackBaseURL, settings.YouTrackProjectID),
		fmt.Sprintf("%s/api/projects/%s/issues?fields=id,idReadable,summary,description,created,updated,customFields(name,value(name,localizedName)),project(shortName)&top=200",
			settings.YouTrackBaseURL, settings.YouTrackProjectID),
	}

	for _, url := range urls {
		if issues, err := s.makeRequest(settings, url); err == nil {
			return issues, nil
		}
	}

	return nil, fmt.Errorf("project endpoint approach failed")
}

// makeRequest makes HTTP request to YouTrack API
func (s *YouTrackService) makeRequest(settings *config.UserSettings, url string) ([]YouTrackIssue, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+settings.YouTrackToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Cache-Control", "no-cache")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var issues []YouTrackIssue
	if err := json.Unmarshal(body, &issues); err != nil {
		return nil, fmt.Errorf("JSON parsing error: %w", err)
	}

	return issues, nil
}

// CreateIssue creates a new YouTrack issue
func (s *YouTrackService) CreateIssue(userID int, task AsanaTask) error {
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return fmt.Errorf("failed to get user settings: %w", err)
	}

	if settings.YouTrackBaseURL == "" || settings.YouTrackToken == "" || settings.YouTrackProjectID == "" {
		return fmt.Errorf("youtrack credentials not configured")
	}

	// Check for duplicate
	if s.IsDuplicateTicket(userID, task.Name) {
		return fmt.Errorf("ticket with title '%s' already exists in YouTrack", task.Name)
	}

	asanaService := NewAsanaService(s.configService)
	state := asanaService.MapStateToYouTrack(task)

	if state == "FINDINGS_NO_SYNC" || state == "READY_FOR_STAGE_NO_SYNC" {
		return fmt.Errorf("cannot create ticket for display-only column")
	}

	// Sanitize title - replace "/" with "or"
	sanitizedTitle := utils.SanitizeTitle(task.Name)

	payload := map[string]interface{}{
		"$type":       "Issue",
		"summary":     sanitizedTitle,
		"description": task.Notes,
		"project": map[string]interface{}{
			"$type":     "Project",
			"shortName": settings.YouTrackProjectID,
		},
	}

	customFields := []map[string]interface{}{}

	if state != "" {
		customFields = append(customFields, map[string]interface{}{
			"$type": "StateIssueCustomField",
			"name":  "State",
			"value": map[string]interface{}{
				"$type": "StateBundleElement",
				"name":  state,
			},
		})
	}

	// Add subsystem mapping using user-specific tag mappings from database
	asanaTags := asanaService.GetTags(task)
	if len(asanaTags) > 0 {
		tagMapper := NewTagMapperForUser(userID, s.configService)
		primaryTag := asanaTags[0]
		subsystem := tagMapper.MapTagToSubsystem(primaryTag)
		if subsystem != "" {
			// Use SingleOwnedIssueCustomField for single-value owned fields
			customFields = append(customFields, map[string]interface{}{
				"$type": "SingleOwnedIssueCustomField",
				"name":  "Subsystem",
				"value": map[string]interface{}{
					"$type": "OwnedBundleElement",
					"name":  subsystem,
				},
			})
		}
	}

	if len(customFields) > 0 {
		payload["customFields"] = customFields
	}

	// Create the issue and get the ID
	issueID, err := s.createIssueAndGetID(settings, payload)
	if err != nil {
		return err
	}

	fmt.Printf("Created YouTrack issue: %s for Asana task: %s\n", issueID, task.GID)

	// Automatically assign the issue to the configured board
	if err := s.AssignIssueToBoard(userID, issueID); err != nil {
		fmt.Printf("Warning: Failed to assign issue to board: %v\n", err)
		// Don't fail the whole operation if board assignment fails
	}

	return nil
}

// CreateIssueWithReturn creates a new YouTrack issue and returns the issue ID
func (s *YouTrackService) CreateIssueWithReturn(userID int, task AsanaTask) (string, error) {
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user settings: %w", err)
	}

	if settings.YouTrackBaseURL == "" || settings.YouTrackToken == "" || settings.YouTrackProjectID == "" {
		return "", fmt.Errorf("youtrack credentials not configured")
	}

	// Check for duplicate
	if s.IsDuplicateTicket(userID, task.Name) {
		return "", fmt.Errorf("ticket with title '%s' already exists in YouTrack", task.Name)
	}

	asanaService := NewAsanaService(s.configService)
	state := asanaService.MapStateToYouTrack(task)

	if state == "FINDINGS_NO_SYNC" || state == "READY_FOR_STAGE_NO_SYNC" {
		return "", fmt.Errorf("cannot create ticket for display-only column")
	}

	// Sanitize title - replace "/" with "or"
	sanitizedTitle := utils.SanitizeTitle(task.Name)

	payload := map[string]interface{}{
		"$type":       "Issue",
		"summary":     sanitizedTitle,
		"description": task.Notes,
		"project": map[string]interface{}{
			"$type":     "Project",
			"shortName": settings.YouTrackProjectID,
		},
	}

	customFields := []map[string]interface{}{}

	if state != "" {
		customFields = append(customFields, map[string]interface{}{
			"$type": "StateIssueCustomField",
			"name":  "State",
			"value": map[string]interface{}{
				"$type": "StateBundleElement",
				"name":  state,
			},
		})
	}

	// Add subsystem mapping using user-specific tag mappings from database
	asanaTags := asanaService.GetTags(task)
	if len(asanaTags) > 0 {
		tagMapper := NewTagMapperForUser(userID, s.configService)
		primaryTag := asanaTags[0]
		subsystem := tagMapper.MapTagToSubsystem(primaryTag)
		if subsystem != "" {
			// Use SingleOwnedIssueCustomField for single-value owned fields
			customFields = append(customFields, map[string]interface{}{
				"$type": "SingleOwnedIssueCustomField",
				"name":  "Subsystem",
				"value": map[string]interface{}{
					"$type": "OwnedBundleElement",
					"name":  subsystem,
				},
			})
		}
	}

	if len(customFields) > 0 {
		payload["customFields"] = customFields
	}

	// Create the issue and get the ID
	issueID, err := s.createIssueAndGetID(settings, payload)
	if err != nil {
		return "", err
	}

	fmt.Printf("Created YouTrack issue: %s for Asana task: %s\n", issueID, task.GID)

	// Automatically assign the issue to the configured board
	if err := s.AssignIssueToBoard(userID, issueID); err != nil {
		fmt.Printf("Warning: Failed to assign issue to board: %v\n", err)
		// Don't fail the whole operation if board assignment fails
	}

	return issueID, nil
}

// Helper method to create issue and return its ID
func (s *YouTrackService) createIssueAndGetID(settings *config.UserSettings, payload map[string]interface{}) (string, error) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/issues", settings.YouTrackBaseURL)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+settings.YouTrackToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyStr := string(body)
		if strings.Contains(bodyStr, "incompatible-issue-custom-field-name-Subsystem") {
			// Retry without subsystem
			if customFields, ok := payload["customFields"].([]map[string]interface{}); ok {
				var filteredFields []map[string]interface{}
				for _, field := range customFields {
					if name, ok := field["name"].(string); ok && name != "Subsystem" {
						filteredFields = append(filteredFields, field)
					}
				}
				payload["customFields"] = filteredFields
			}
			return s.createIssueAndGetID(settings, payload)
		}
		return "", fmt.Errorf("youtrack API error: %d - %s", resp.StatusCode, bodyStr)
	}

	// Parse response to get issue ID
	var createdIssue YouTrackIssue
	if err := json.Unmarshal(body, &createdIssue); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if createdIssue.ID == "" {
		return "", fmt.Errorf("created issue but received empty ID")
	}

	return createdIssue.ID, nil
}

// UpdateIssue updates an existing YouTrack issue
func (s *YouTrackService) UpdateIssue(userID int, issueID string, task AsanaTask) error {
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return fmt.Errorf("failed to get user settings: %w", err)
	}

	asanaService := NewAsanaService(s.configService)
	state := asanaService.MapStateToYouTrack(task)

	if state == "FINDINGS_NO_SYNC" || state == "READY_FOR_STAGE_NO_SYNC" {
		return fmt.Errorf("cannot update ticket for display-only column")
	}

	// Sanitize title - replace "/" with "or"
	sanitizedTitle := utils.SanitizeTitle(task.Name)

	payload := map[string]interface{}{
		"$type":       "Issue",
		"summary":     sanitizedTitle,
		"description": task.Notes,
	}

	customFields := []map[string]interface{}{}

	if state != "" {
		customFields = append(customFields, map[string]interface{}{
			"$type": "StateIssueCustomField",
			"name":  "State",
			"value": map[string]interface{}{
				"$type": "StateBundleElement",
				"name":  state,
			},
		})
	}

	// Add subsystem mapping using user-specific tag mappings from database
	asanaTags := asanaService.GetTags(task)
	if len(asanaTags) > 0 {
		tagMapper := NewTagMapperForUser(userID, s.configService)
		primaryTag := asanaTags[0]
		subsystem := tagMapper.MapTagToSubsystem(primaryTag)
		if subsystem != "" {
			// Use SingleOwnedIssueCustomField for single-value owned fields
			customFields = append(customFields, map[string]interface{}{
				"$type": "SingleOwnedIssueCustomField",
				"name":  "Subsystem",
				"value": map[string]interface{}{
					"$type": "OwnedBundleElement",
					"name":  subsystem,
				},
			})
		}
	}

	if len(customFields) > 0 {
		payload["customFields"] = customFields
	}

	return s.createOrUpdateIssue(settings, issueID, payload)
}

// DeleteIssue deletes a YouTrack issue
func (s *YouTrackService) DeleteIssue(userID int, issueID string) error {
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return fmt.Errorf("failed to get user settings: %w", err)
	}

	if settings.YouTrackBaseURL == "" || settings.YouTrackToken == "" {
		return fmt.Errorf("youtrack credentials not configured")
	}

	url := fmt.Sprintf("%s/api/issues/%s", settings.YouTrackBaseURL, issueID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+settings.YouTrackToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("delete request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("youtrack delete error: %d - %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Successfully deleted YouTrack issue: %s for user %d\n", issueID, userID)
	return nil
}

// createOrUpdateIssue creates or updates a YouTrack issue
func (s *YouTrackService) createOrUpdateIssue(settings *config.UserSettings, issueID string, payload map[string]interface{}) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	var url string
	if issueID == "" {
		url = fmt.Sprintf("%s/api/issues", settings.YouTrackBaseURL)
	} else {
		url = fmt.Sprintf("%s/api/issues/%s", settings.YouTrackBaseURL, issueID)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+settings.YouTrackToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyStr := string(body)
		if strings.Contains(bodyStr, "incompatible-issue-custom-field-name-Subsystem") {
			return s.createOrUpdateIssueWithoutSubsystem(settings, issueID, payload)
		}
		return fmt.Errorf("youtrack API error: %d - %s", resp.StatusCode, bodyStr)
	}

	return nil
}

// createOrUpdateIssueWithoutSubsystem fallback for systems without Subsystem field
func (s *YouTrackService) createOrUpdateIssueWithoutSubsystem(settings *config.UserSettings, issueID string, payload map[string]interface{}) error {
	// Remove subsystem from custom fields
	if customFields, ok := payload["customFields"].([]map[string]interface{}); ok {
		var filteredFields []map[string]interface{}
		for _, field := range customFields {
			if name, ok := field["name"].(string); ok && name != "Subsystem" {
				filteredFields = append(filteredFields, field)
			}
		}
		payload["customFields"] = filteredFields
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	var url string
	if issueID == "" {
		url = fmt.Sprintf("%s/api/issues", settings.YouTrackBaseURL)
	} else {
		url = fmt.Sprintf("%s/api/issues/%s", settings.YouTrackBaseURL, issueID)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+settings.YouTrackToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("youtrack API error: %d - %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Updated issue (without subsystem field)\n")
	return nil
}

// IsDuplicateTicket checks if a ticket with the given title already exists
func (s *YouTrackService) IsDuplicateTicket(userID int, title string) bool {
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return false
	}

	query := fmt.Sprintf("project:%s summary:%s", settings.YouTrackProjectID, title)
	encodedQuery := strings.ReplaceAll(query, " ", "%20")

	url := fmt.Sprintf("%s/api/issues?fields=id,idReadable,summary&query=%s&top=5",
		settings.YouTrackBaseURL, encodedQuery)

	issues, err := s.makeRequest(settings, url)
	if err != nil {
		return false
	}

	for _, issue := range issues {
		if strings.EqualFold(issue.Summary, title) {
			return true
		}
	}

	return false
}

// GetStatus extracts status from a YouTrack issue - FIXED VERSION
// This now prioritizes the technical 'name' field over 'localizedName' for consistency
func (s *YouTrackService) GetStatus(issue YouTrackIssue) string {
	for _, field := range issue.CustomFields {
		if field.Name == "State" {
			switch value := field.Value.(type) {
			case map[string]interface{}:
				// PRIORITY 1: Try to get the technical 'name' field first
				if name, ok := value["name"].(string); ok && name != "" {
					fmt.Printf("DEBUG: YouTrack Status (technical name): %s\n", name)
					return name
				}
				// PRIORITY 2: Fall back to 'localizedName' if 'name' is not available
				if localizedName, ok := value["localizedName"].(string); ok && localizedName != "" {
					fmt.Printf("DEBUG: YouTrack Status (localized name): %s\n", localizedName)
					return localizedName
				}
			case string:
				if value != "" {
					fmt.Printf("DEBUG: YouTrack Status (string): %s\n", value)
					return value
				}
			case nil:
				return "No State"
			}
		}
	}
	return "Unknown"
}

// GetStatusNormalized gets the status and normalizes it for comparison
// This handles different variations of the same status
func (s *YouTrackService) GetStatusNormalized(issue YouTrackIssue) string {
	status := s.GetStatus(issue)

	// Normalize common variations
	statusLower := strings.ToLower(strings.TrimSpace(status))

	// Map common variations to standard names
	statusMap := map[string]string{
		"backlog":     "Backlog",
		"open":        "Backlog",
		"to do":       "Backlog",
		"todo":        "Backlog",
		"in progress": "In Progress",
		"inprogress":  "In Progress",
		"in-progress": "In Progress",
		"dev":         "DEV",
		"development": "DEV",
		"in dev":      "DEV",
		"stage":       "STAGE",
		"staging":     "STAGE",
		"in stage":    "STAGE",
		"blocked":     "Blocked",
		"on hold":     "Blocked",
	}

	if normalized, exists := statusMap[statusLower]; exists {
		fmt.Printf("DEBUG: Normalized '%s' to '%s'\n", status, normalized)
		return normalized
	}

	// Return original if no mapping found
	return status
}

// ExtractAsanaID extracts Asana ID from YouTrack issue description
func (s *YouTrackService) ExtractAsanaID(issue YouTrackIssue) string {
	if strings.Contains(issue.Description, "Asana ID:") {
		lines := strings.Split(issue.Description, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Asana ID:") {
				parts := strings.Split(line, "Asana ID:")
				if len(parts) > 1 {
					return strings.TrimSpace(strings.Trim(parts[1], "]"))
				}
			}
		}
	}
	return ""
}

// FindIssueByAsanaID finds YouTrack issue by Asana task ID
func (s *YouTrackService) FindIssueByAsanaID(userID int, asanaTaskID string) (string, error) {
	issues, err := s.GetIssues(userID)
	if err != nil {
		return "", fmt.Errorf("failed to get YouTrack issues: %w", err)
	}

	for _, issue := range issues {
		asanaID := s.ExtractAsanaID(issue)
		if asanaID == asanaTaskID {
			return issue.ID, nil
		}
	}

	return "", fmt.Errorf("no YouTrack issue found for Asana task %s", asanaTaskID)
}


// GetBoards retrieves available YouTrack agile boards for a user
func (s *YouTrackService) GetBoards(userID int) ([]database.YouTrackBoard, error) {
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user settings: %w", err)
	}

	if settings.YouTrackBaseURL == "" || settings.YouTrackToken == "" {
		return nil, fmt.Errorf("youtrack credentials not configured")
	}

	url := fmt.Sprintf("%s/api/agiles?$top=-1&fields=id,name,sprintsSettings(disableSprints),projects(id)",
		settings.YouTrackBaseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+settings.YouTrackToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("youtrack API error: %d - %s", resp.StatusCode, string(body))
	}

	var rawBoards []map[string]interface{}
	if err := json.Unmarshal(body, &rawBoards); err != nil {
		return nil, fmt.Errorf("failed to parse boards: %w", err)
	}

	var boards []database.YouTrackBoard
	for _, board := range rawBoards {
		id, _ := board["id"].(string)
		name, _ := board["name"].(string)
		if id != "" && name != "" {
			boards = append(boards, database.YouTrackBoard{
				ID:   id,
				Name: name,
			})
		}
	}

	return boards, nil
}

// AssignIssueToBoard assigns a YouTrack issue to an agile board
func (s *YouTrackService) AssignIssueToBoard(userID int, issueID string) error {
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return fmt.Errorf("failed to get user settings: %w", err)
	}

	if settings.YouTrackBaseURL == "" || settings.YouTrackToken == "" {
		return fmt.Errorf("youtrack credentials not configured")
	}

	// If no board is configured, skip board assignment
	if settings.YouTrackBoardID == "" {
		fmt.Printf("No board configured for user %d, skipping board assignment\n", userID)
		return nil
	}

	// Get board details to get the board name
	boards, err := s.GetBoards(userID)
	if err != nil {
		return fmt.Errorf("failed to get boards: %w", err)
	}

	var boardName string
	for _, board := range boards {
		if board.ID == settings.YouTrackBoardID {
			boardName = board.Name
			break
		}
	}

	if boardName == "" {
		return fmt.Errorf("board with ID %s not found", settings.YouTrackBoardID)
	}

	// Get the issue's readable ID (like ARD-123)
	issueReadableID, err := s.getIssueReadableID(settings, issueID)
	if err != nil {
		return fmt.Errorf("failed to get issue readable ID: %w", err)
	}

	// Use YouTrack commands API to add issue to board
	commandPayload := map[string]interface{}{
		"query": fmt.Sprintf("add Board %s", boardName),
		"issues": []map[string]interface{}{
			{
				"idReadable": issueReadableID,
			},
		},
	}

	jsonPayload, err := json.Marshal(commandPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal command payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/commands", settings.YouTrackBaseURL)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create command request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+settings.YouTrackToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("command request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("youtrack command API error: %d - %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Successfully assigned issue %s to board '%s'\n", issueReadableID, boardName)
	return nil
}

// getIssueReadableID retrieves the readable ID (like ARD-123) for an issue
func (s *YouTrackService) getIssueReadableID(settings *config.UserSettings, issueID string) (string, error) {
	url := fmt.Sprintf("%s/api/issues/%s?fields=idReadable", settings.YouTrackBaseURL, issueID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+settings.YouTrackToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("youtrack API error: %d - %s", resp.StatusCode, string(body))
	}

	var issue struct {
		IDReadable string `json:"idReadable"`
	}
	if err := json.Unmarshal(body, &issue); err != nil {
		return "", fmt.Errorf("failed to parse issue: %w", err)
	}

	return issue.IDReadable, nil
}
