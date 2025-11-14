package legacy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
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
	fields := "id,summary,description,created,updated,customFields(name,value(name,localizedName,description,id,$type,color)),project(shortName)"

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

	fields := "id,summary,description,created,updated,customFields(name,value(name,localizedName,description,id,$type,color)),project(shortName)"

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
	url := fmt.Sprintf("%s/api/issues?fields=id,summary,description,created,updated,customFields(name,value(name,localizedName,description,id,$type)),project(shortName)&top=200",
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
		fmt.Sprintf("%s/api/admin/projects/%s/issues?fields=id,summary,description,created,updated,customFields(name,value(name,localizedName)),project(shortName)&top=200",
			settings.YouTrackBaseURL, settings.YouTrackProjectID),
		fmt.Sprintf("%s/api/projects/%s/issues?fields=id,summary,description,created,updated,customFields(name,value(name,localizedName)),project(shortName)&top=200",
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
	state := asanaService.MapStateToYouTrackWithSettings(userID, task)

	// Check if column is display-only or unmapped
	if state == "DISPLAY_ONLY" {
		return fmt.Errorf("cannot create ticket for display-only column")
	}
	if state == "" {
		return fmt.Errorf("cannot create ticket: column not mapped in settings")
	}

	// Sanitize title - replace "/" with "or"
	sanitizedTitle := utils.SanitizeTitle(task.Name)

	// Convert HTML notes to YouTrack markdown if available, otherwise use plain notes
	description := task.Notes
	if task.HTMLNotes != "" {
		description = utils.ConvertAsanaHTMLToYouTrackMarkdown(task.HTMLNotes)
	}

	payload := map[string]interface{}{
		"$type":       "Issue",
		"summary":     sanitizedTitle,
		"description": description,
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
			// Get the subsystem field and value IDs from YouTrack
			fieldID, valueID, err := s.GetSubsystemFieldInfo(userID, subsystem)
			if err != nil {
				fmt.Printf("Warning: Failed to get subsystem field info for '%s': %v\n", subsystem, err)
				// Don't fail the whole operation, just skip the subsystem field
			} else {
				customFields = append(customFields, map[string]interface{}{
					"$type": "SingleOwnedIssueCustomField",
					"id":    fieldID,
					"value": map[string]interface{}{
						"kind":        "enum",
						"id":          valueID,
						"name":        subsystem,
						"label":       subsystem,
						"description": nil,
					},
				})
			}
		}
	}

	if len(customFields) > 0 {
		payload["fields"] = customFields
	}

	// Create the issue and get the ID
	issueID, err := s.createIssueAndGetID(settings, payload)
	if err != nil {
		return err
	}

	fmt.Printf("Created YouTrack issue: %s for Asana task: %s\n", issueID, task.GID)

	// Auto-assign agile board if configured
	if settings.YouTrackBoardID != "" {
		if err := s.assignIssueToAgileBoard(settings, issueID); err != nil {
			fmt.Printf("Warning: Failed to assign issue to agile board: %v\n", err)
			// Don't fail the whole operation if agile board assignment fails
		}
	}

	// Process attachments - download from Asana and upload to YouTrack
	if len(task.Attachments) > 0 {
		asanaService := NewAsanaService(s.configService)
		if err := s.ProcessAttachments(userID, issueID, task, asanaService); err != nil {
			fmt.Printf("Warning: Failed to process attachments: %v\n", err)
			// Don't fail the whole operation if attachment processing fails
		}
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
	state := asanaService.MapStateToYouTrackWithSettings(userID, task)

	// Check if column is display-only or unmapped
	if state == "DISPLAY_ONLY" {
		return "", fmt.Errorf("cannot create ticket for display-only column")
	}
	if state == "" {
		return "", fmt.Errorf("cannot create ticket: column not mapped in settings")
	}

	// Sanitize title - replace "/" with "or"
	sanitizedTitle := utils.SanitizeTitle(task.Name)

	// Convert HTML notes to YouTrack markdown if available, otherwise use plain notes
	description := task.Notes
	if task.HTMLNotes != "" {
		description = utils.ConvertAsanaHTMLToYouTrackMarkdown(task.HTMLNotes)
	}

	payload := map[string]interface{}{
		"$type":       "Issue",
		"summary":     sanitizedTitle,
		"description": description,
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
			// Get the subsystem field and value IDs from YouTrack
			fieldID, valueID, err := s.GetSubsystemFieldInfo(userID, subsystem)
			if err != nil {
				fmt.Printf("Warning: Failed to get subsystem field info for '%s': %v\n", subsystem, err)
				// Don't fail the whole operation, just skip the subsystem field
			} else {
				customFields = append(customFields, map[string]interface{}{
					"$type": "SingleOwnedIssueCustomField",
					"id":    fieldID,
					"value": map[string]interface{}{
						"kind":        "enum",
						"id":          valueID,
						"name":        subsystem,
						"label":       subsystem,
						"description": nil,
					},
				})
			}
		}
	}

	if len(customFields) > 0 {
		payload["fields"] = customFields
	}

	// Create the issue and get the ID
	issueID, err := s.createIssueAndGetID(settings, payload)
	if err != nil {
		return "", err
	}

	fmt.Printf("Created YouTrack issue: %s for Asana task: %s\n", issueID, task.GID)

	// Auto-assign agile board if configured
	if settings.YouTrackBoardID != "" {
		if err := s.assignIssueToAgileBoard(settings, issueID); err != nil {
			fmt.Printf("Warning: Failed to assign issue to agile board: %v\n", err)
			// Don't fail the whole operation if agile board assignment fails
		}
	}

	// Process attachments - download from Asana and upload to YouTrack
	if len(task.Attachments) > 0 {
		asanaService := NewAsanaService(s.configService)
		if err := s.ProcessAttachments(userID, issueID, task, asanaService); err != nil {
			fmt.Printf("Warning: Failed to process attachments: %v\n", err)
			// Don't fail the whole operation if attachment processing fails
		}
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
			if customFields, ok := payload["fields"].([]map[string]interface{}); ok {
				var filteredFields []map[string]interface{}
				for _, field := range customFields {
					if name, ok := field["name"].(string); ok && name != "Subsystem" {
						filteredFields = append(filteredFields, field)
					}
				}
				payload["fields"] = filteredFields
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

// assignIssueToAgileBoard assigns an issue to the configured agile board using YouTrack commands API
func (s *YouTrackService) assignIssueToAgileBoard(settings *config.UserSettings, issueID string) error {
	// First, get the agile board details to get board name and sprints
	url := fmt.Sprintf("%s/api/agiles/%s?fields=id,name,sprints(id,name,archived,finish,start)",
		settings.YouTrackBaseURL,
		settings.YouTrackBoardID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+settings.YouTrackToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get agile board: %d - %s", resp.StatusCode, string(body))
	}

	var agileBoard map[string]interface{}
	if err := json.Unmarshal(body, &agileBoard); err != nil {
		return fmt.Errorf("failed to parse agile board: %w", err)
	}

	// Get the board name
	boardName, ok := agileBoard["name"].(string)
	if !ok || boardName == "" {
		return fmt.Errorf("could not get agile board name")
	}

	// Get sprints from the agile board
	sprints, ok := agileBoard["sprints"].([]interface{})
	if !ok || len(sprints) == 0 {
		// If no sprints, just add the issue to the agile board without a specific sprint
		return s.addIssueToAgileBoardUsingCommand(settings, issueID, boardName, "")
	}

	// Find the first non-archived sprint
	var targetSprintName string
	for _, sprintInterface := range sprints {
		sprint, ok := sprintInterface.(map[string]interface{})
		if !ok {
			continue
		}
		archived, _ := sprint["archived"].(bool)
		if !archived {
			sprintName, _ := sprint["name"].(string)
			if sprintName != "" {
				targetSprintName = sprintName
				break
			}
		}
	}

	if targetSprintName == "" {
		// No active sprint found, add to board without sprint
		return s.addIssueToAgileBoardUsingCommand(settings, issueID, boardName, "")
	}

	// Add the issue to the agile board and sprint using the commands API
	return s.addIssueToAgileBoardUsingCommand(settings, issueID, boardName, targetSprintName)
}

// addIssueToAgileBoardUsingCommand adds an issue to agile board using YouTrack's commands API
func (s *YouTrackService) addIssueToAgileBoardUsingCommand(settings *config.UserSettings, issueID, boardName, sprintName string) error {
	// Build the command string
	var command string
	if sprintName != "" {
		command = fmt.Sprintf("add Board %s %s", boardName, sprintName)
	} else {
		command = fmt.Sprintf("add Board %s", boardName)
	}

	// Use the YouTrack commands API
	commandURL := fmt.Sprintf("%s/api/commands", settings.YouTrackBaseURL)

	commandPayload := map[string]interface{}{
		"query": command,
		"issues": []map[string]interface{}{
			{
				"idReadable": issueID,
			},
		},
	}

	jsonPayload, err := json.Marshal(commandPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal command payload: %w", err)
	}

	req, err := http.NewRequest("POST", commandURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create command request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+settings.YouTrackToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	commandResp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("command request failed: %w", err)
	}
	defer commandResp.Body.Close()

	commandBody, _ := io.ReadAll(commandResp.Body)

	if commandResp.StatusCode != http.StatusOK && commandResp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to execute command: %d - %s", commandResp.StatusCode, string(commandBody))
	}

	if sprintName != "" {
		fmt.Printf("Successfully assigned issue %s to agile board '%s' (sprint: '%s')\n", issueID, boardName, sprintName)
	} else {
		fmt.Printf("Successfully assigned issue %s to agile board '%s'\n", issueID, boardName)
	}
	return nil
}

// UpdateIssue updates an existing YouTrack issue
func (s *YouTrackService) UpdateIssue(userID int, issueID string, task AsanaTask) error {
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return fmt.Errorf("failed to get user settings: %w", err)
	}

	asanaService := NewAsanaService(s.configService)
	state := asanaService.MapStateToYouTrackWithSettings(userID, task)

	// Check if column is display-only or unmapped
	if state == "DISPLAY_ONLY" {
		return fmt.Errorf("cannot update ticket for display-only column")
	}
	if state == "" {
		return fmt.Errorf("cannot update ticket: column not mapped in settings")
	}

	// Sanitize title - replace "/" with "or"
	sanitizedTitle := utils.SanitizeTitle(task.Name)

	// Convert HTML notes to YouTrack markdown if available, otherwise use plain notes
	description := task.Notes
	if task.HTMLNotes != "" {
		description = utils.ConvertAsanaHTMLToYouTrackMarkdown(task.HTMLNotes)
	}

	payload := map[string]interface{}{
		"$type":       "Issue",
		"summary":     sanitizedTitle,
		"description": description,
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
			// Get the subsystem field and value IDs from YouTrack
			fieldID, valueID, err := s.GetSubsystemFieldInfo(userID, subsystem)
			if err != nil {
				fmt.Printf("Warning: Failed to get subsystem field info for '%s': %v\n", subsystem, err)
				// Don't fail the whole operation, just skip the subsystem field
			} else {
				customFields = append(customFields, map[string]interface{}{
					"$type": "SingleOwnedIssueCustomField",
					"id":    fieldID,
					"value": map[string]interface{}{
						"kind":        "enum",
						"id":          valueID,
						"name":        subsystem,
						"label":       subsystem,
						"description": nil,
					},
				})
			}
		}
	}

	if len(customFields) > 0 {
		payload["fields"] = customFields
	}

	return s.createOrUpdateIssue(settings, issueID, payload)
}

// UpdateIssueStatus updates only the status of a YouTrack issue (for rollback)
func (s *YouTrackService) UpdateIssueStatus(userID int, issueID, status string) error {
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return fmt.Errorf("failed to get user settings: %w", err)
	}

	payload := map[string]interface{}{
		"$type": "Issue",
		"fields": []map[string]interface{}{
			{
				"$type": "StateIssueCustomField",
				"name":  "State",
				"value": map[string]interface{}{
					"$type": "StateBundleElement",
					"name":  status,
				},
			},
		},
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
	if customFields, ok := payload["fields"].([]map[string]interface{}); ok {
		var filteredFields []map[string]interface{}
		for _, field := range customFields {
			if name, ok := field["name"].(string); ok && name != "Subsystem" {
				filteredFields = append(filteredFields, field)
			}
		}
		payload["fields"] = filteredFields
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

	url := fmt.Sprintf("%s/api/issues?fields=id,summary&query=%s&top=5",
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
					return name
				}
				// PRIORITY 2: Fall back to 'localizedName' if 'name' is not available
				if localizedName, ok := value["localizedName"].(string); ok && localizedName != "" {
					return localizedName
				}
			case string:
				if value != "" {
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

// GetStates retrieves all workflow states from a YouTrack project
func (s *YouTrackService) GetStates(userID int) ([]database.YouTrackState, error) {
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user settings: %w", err)
	}

	if settings.YouTrackBaseURL == "" || settings.YouTrackToken == "" || settings.YouTrackProjectID == "" {
		return nil, fmt.Errorf("youtrack credentials not configured")
	}

	// Fetch the project's custom fields to get the State field
	url := fmt.Sprintf("%s/api/admin/projects/%s/customFields?fields=field(name,fieldType(id)),bundle(values(name))",
		settings.YouTrackBaseURL, settings.YouTrackProjectID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+settings.YouTrackToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("youtrack API error: %d - %s", resp.StatusCode, string(body))
	}

	// Parse the response to find the State field
	var customFields []map[string]interface{}
	if err := json.Unmarshal(body, &customFields); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Find the State field and extract its values
	var states []database.YouTrackState
	for _, field := range customFields {
		if fieldInfo, ok := field["field"].(map[string]interface{}); ok {
			if fieldName, ok := fieldInfo["name"].(string); ok && fieldName == "State" {
				if bundle, ok := field["bundle"].(map[string]interface{}); ok {
					if values, ok := bundle["values"].([]interface{}); ok {
						for _, val := range values {
							if valMap, ok := val.(map[string]interface{}); ok {
								if stateName, ok := valMap["name"].(string); ok {
									states = append(states, database.YouTrackState{
										Name: stateName,
									})
								}
							}
						}
					}
				}
			}
		}
	}

	fmt.Printf("Retrieved %d YouTrack states for user %d\n", len(states), userID)
	return states, nil
}

// GetBoards retrieves all agile boards from YouTrack
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
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("youtrack API error: %d - %s", resp.StatusCode, string(body))
	}

	var rawBoards []map[string]interface{}
	if err := json.Unmarshal(body, &rawBoards); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
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

// UploadAttachment uploads an attachment to a YouTrack issue
func (s *YouTrackService) UploadAttachment(userID int, issueID string, filename string, fileData []byte) error {
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return fmt.Errorf("failed to get user settings: %w", err)
	}

	if settings.YouTrackBaseURL == "" || settings.YouTrackToken == "" {
		return fmt.Errorf("youtrack credentials not configured")
	}

	// Create multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add the file
	part, err := writer.CreateFormFile("attachment", filename)
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	_, err = part.Write(fileData)
	if err != nil {
		return fmt.Errorf("failed to write file data: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	// Upload to YouTrack
	url := fmt.Sprintf("%s/api/issues/%s/attachments?fields=id,name,size", settings.YouTrackBaseURL, issueID)

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+settings.YouTrackToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 120 * time.Second} // Longer timeout for uploads
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("youtrack upload error: %d - %s", resp.StatusCode, string(respBody))
	}

	fmt.Printf("Successfully uploaded attachment '%s' to YouTrack issue %s\n", filename, issueID)
	return nil
}

// ProcessAttachments downloads attachments from Asana and uploads them to YouTrack
func (s *YouTrackService) ProcessAttachments(userID int, issueID string, task AsanaTask, asanaService *AsanaService) error {
	if len(task.Attachments) == 0 {
		return nil
	}

	fmt.Printf("Processing %d attachments for issue %s\n", len(task.Attachments), issueID)

	successCount := 0
	failCount := 0

	for _, attachment := range task.Attachments {
		// Skip if no GID
		if attachment.GID == "" {
			fmt.Printf("Skipping attachment '%s' - no GID\n", attachment.Name)
			continue
		}

		// Skip very large files (> 50MB)
		if attachment.Size > 50*1024*1024 {
			fmt.Printf("Skipping attachment '%s' - file too large (%d bytes)\n", attachment.Name, attachment.Size)
			failCount++
			continue
		}

		// Download from Asana
		fmt.Printf("Downloading attachment '%s' from Asana...\n", attachment.Name)
		fileData, err := asanaService.DownloadAttachment(userID, attachment.GID)
		if err != nil {
			fmt.Printf("Warning: Failed to download attachment '%s': %v\n", attachment.Name, err)
			failCount++
			continue
		}

		// Upload to YouTrack
		fmt.Printf("Uploading attachment '%s' to YouTrack...\n", attachment.Name)
		err = s.UploadAttachment(userID, issueID, attachment.Name, fileData)
		if err != nil {
			fmt.Printf("Warning: Failed to upload attachment '%s': %v\n", attachment.Name, err)
			failCount++
			continue
		}

		successCount++
	}

	fmt.Printf("Attachment processing complete: %d succeeded, %d failed\n", successCount, failCount)
	return nil
}

// GetAllUsers fetches all users from YouTrack for the creator dropdown
func (s *YouTrackService) GetAllUsers(userID int) ([]YouTrackUser, error) {
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user settings: %w", err)
	}

	if settings.YouTrackBaseURL == "" || settings.YouTrackToken == "" {
		return nil, fmt.Errorf("youtrack credentials not configured")
	}

	url := fmt.Sprintf("%s/api/users?fields=id,login,fullName,email", settings.YouTrackBaseURL)

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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("youtrack API error: %d - %s", resp.StatusCode, string(body))
	}

	var users []YouTrackUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return users, nil
}

// GetIssuesByCreator fetches YouTrack issues filtered by creator
func (s *YouTrackService) GetIssuesByCreator(userID int, creatorFilter string) ([]YouTrackIssue, error) {
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user settings: %w", err)
	}

	if settings.YouTrackBaseURL == "" || settings.YouTrackToken == "" || settings.YouTrackProjectID == "" {
		return nil, fmt.Errorf("youtrack credentials not configured")
	}

	// Build query with creator filter
	var query string
	if creatorFilter == "All" || creatorFilter == "" {
		query = fmt.Sprintf("project: {%s}", settings.YouTrackProjectID)
	} else {
		query = fmt.Sprintf("project: {%s} created by: {%s}", settings.YouTrackProjectID, creatorFilter)
	}

	// Include all necessary fields including attachments and created by
	fields := "id,idReadable,summary,description,created,updated," +
		"customFields(name,value(name,id))," +
		"attachments(id,name,size,mimeType,url,extension)," +
		"reporter(fullName,login)," +
		"project(shortName)"

	encodedQuery := strings.ReplaceAll(query, " ", "%20")
	encodedQuery = strings.ReplaceAll(encodedQuery, "{", "%7B")
	encodedQuery = strings.ReplaceAll(encodedQuery, "}", "%7D")
	encodedQuery = strings.ReplaceAll(encodedQuery, ":", "%3A")

	url := fmt.Sprintf("%s/api/issues?fields=%s&query=%s&$top=500",
		settings.YouTrackBaseURL, fields, encodedQuery)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+settings.YouTrackToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("youtrack API error: %d - %s", resp.StatusCode, string(body))
	}

	var rawResponse []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Parse the response into YouTrackIssue structs
	issues := make([]YouTrackIssue, 0, len(rawResponse))
	for _, rawIssue := range rawResponse {
		// Use idReadable (e.g., "ARD-123") if available, otherwise fall back to id
		issueID := getString(rawIssue, "idReadable")
		if issueID == "" {
			issueID = getString(rawIssue, "id")
		}

		issue := YouTrackIssue{
			ID:                  issueID,
			Summary:             getString(rawIssue, "summary"),
			Description:         getString(rawIssue, "description"),
			WikifiedDescription: getString(rawIssue, "wikifiedDescription"),
			Created:             getInt64(rawIssue, "created"),
			Updated:             getInt64(rawIssue, "updated"),
		}

		// Extract State and Subsystem from customFields
		if customFields, ok := rawIssue["customFields"].([]interface{}); ok {
			for _, field := range customFields {
				if fieldMap, ok := field.(map[string]interface{}); ok {
					fieldName := getString(fieldMap, "name")

					if fieldName == "State" {
						if value, ok := fieldMap["value"].(map[string]interface{}); ok {
							issue.State = getString(value, "name")
						}
					} else if fieldName == "Subsystem" {
						if value, ok := fieldMap["value"].(map[string]interface{}); ok {
							issue.Subsystem = getString(value, "name")
						}
					}
				}
			}
		}

		// Extract creator name from reporter
		if reporter, ok := rawIssue["reporter"].(map[string]interface{}); ok {
			issue.CreatedBy = getString(reporter, "fullName")
			if issue.CreatedBy == "" {
				issue.CreatedBy = getString(reporter, "login")
			}
		}

		// Extract attachments
		if attachments, ok := rawIssue["attachments"].([]interface{}); ok {
			issue.Attachments = make([]YouTrackAttachment, 0, len(attachments))
			for _, att := range attachments {
				if attMap, ok := att.(map[string]interface{}); ok {
					attachment := YouTrackAttachment{
						ID:        getString(attMap, "id"),
						Name:      getString(attMap, "name"),
						Size:      getInt64(attMap, "size"),
						MimeType:  getString(attMap, "mimeType"),
						URL:       getString(attMap, "url"),
						Extension: getString(attMap, "extension"),
					}
					issue.Attachments = append(issue.Attachments, attachment)
				}
			}
		}

		issues = append(issues, issue)
	}

	return issues, nil
}

// DownloadAttachment downloads an attachment from a YouTrack issue using the attachment URL
func (s *YouTrackService) DownloadAttachment(userID int, issueID, attachmentURL string) ([]byte, error) {
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user settings: %w", err)
	}

	// The attachmentURL comes from the API response and is a relative path like "/api/files/12-6?sign=..."
	// We need to prepend the base URL
	fullURL := settings.YouTrackBaseURL + attachmentURL

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+settings.YouTrackToken)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}

// Helper functions for extracting values from interface{} maps
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getInt64(m map[string]interface{}, key string) int64 {
	if val, ok := m[key].(float64); ok {
		return int64(val)
	}
	if val, ok := m[key].(int64); ok {
		return val
	}
	return 0
}

// GetSubsystemFieldInfo retrieves the subsystem field ID and enum value details from YouTrack
func (s *YouTrackService) GetSubsystemFieldInfo(userID int, subsystemName string) (fieldID string, valueID string, err error) {
	// Hardcoded field ID for Subsystem (from your YouTrack setup)
	fieldID = "172-17"

	// Hardcoded value mappings for subsystems (from your YouTrack cURL)
	subsystemValueMap := map[string]string{
		"UI":     "180-3",
		"MC":     "180-4",
		"Admin":  "180-5",
		"Core":   "180-6",
		"RAG":    "180-7",
		"Studio": "180-8",
		"Mobile": "180-9",
	}

	// Look up the value ID
	if vID, ok := subsystemValueMap[subsystemName]; ok {
		valueID = vID
		return fieldID, valueID, nil
	}

	// If not found in hardcoded map, try the API approach as fallback

	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return "", "", fmt.Errorf("failed to get user settings: %w", err)
	}

	if settings.YouTrackBaseURL == "" || settings.YouTrackToken == "" || settings.YouTrackProjectID == "" {
		return "", "", fmt.Errorf("youtrack credentials not configured")
	}

	// Get project info including custom fields
	// Using /api/admin/projects/{id} with customFields expansion
	url := fmt.Sprintf("%s/api/admin/projects/%s?fields=customFields(field(name),id,bundle(values(id,name)))",
		settings.YouTrackBaseURL, settings.YouTrackProjectID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+settings.YouTrackToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("youtrack API error: %d - %s", resp.StatusCode, string(body))
	}

	// Parse the project response which contains customFields array
	var projectResponse map[string]interface{}
	if err := json.Unmarshal(body, &projectResponse); err != nil {
		return "", "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract customFields array from project response
	customFieldsInterface, ok := projectResponse["customFields"]
	if !ok {
		return "", "", fmt.Errorf("no customFields found in project response")
	}

	customFieldsArray, ok := customFieldsInterface.([]interface{})
	if !ok {
		return "", "", fmt.Errorf("customFields is not an array")
	}

	// Convert to []map[string]interface{}
	customFields := make([]map[string]interface{}, 0, len(customFieldsArray))
	for _, cf := range customFieldsArray {
		if cfMap, ok := cf.(map[string]interface{}); ok {
			customFields = append(customFields, cfMap)
		}
	}

	// Find the Subsystem field
	for _, field := range customFields {
		if fieldInfo, ok := field["field"].(map[string]interface{}); ok {
			if fieldName, ok := fieldInfo["name"].(string); ok {
				if fieldName == "Subsystem" {
					// Get the field ID
					if id, ok := field["id"].(string); ok {
						fieldID = id
					}

					// Find the matching value in the bundle
					if bundle, ok := field["bundle"].(map[string]interface{}); ok {
						if values, ok := bundle["values"].([]interface{}); ok {
							for _, val := range values {
								if valMap, ok := val.(map[string]interface{}); ok {
									if name, ok := valMap["name"].(string); ok {
										if strings.EqualFold(name, subsystemName) {
											if id, ok := valMap["id"].(string); ok {
												valueID = id
												return fieldID, valueID, nil
											}
										}
									}
								}
							}
						}
					}

					// If we found the field but not the value, return an error
					if fieldID != "" {
						return "", "", fmt.Errorf("subsystem value '%s' not found in YouTrack project", subsystemName)
					}
				}
			}
		}
	}

	return "", "", fmt.Errorf("subsystem field not found in YouTrack project")
}
