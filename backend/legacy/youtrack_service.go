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
	state := asanaService.MapStateToYouTrack(task)

	if state == "FINDINGS_NO_SYNC" || state == "READY_FOR_STAGE_NO_SYNC" {
		return fmt.Errorf("cannot create ticket for display-only column")
	}

	// Sanitize title - replace "/" with "or"
	sanitizedTitle := utils.SanitizeTitle(task.Name)

	payload := map[string]interface{}{
		"$type":       "Issue",
		"summary":     sanitizedTitle,
		"description": fmt.Sprintf("%s\n\n[Synced from Asana ID: %s]", task.Notes, task.GID),
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

	// Add subsystem mapping
	asanaTags := asanaService.GetTags(task)
	if len(asanaTags) > 0 {
		tagMapper := NewTagMapper()
		primaryTag := asanaTags[0]
		subsystem := tagMapper.MapTagToSubsystem(primaryTag)
		if subsystem != "" {
			customFields = append(customFields, map[string]interface{}{
				"$type": "MultiOwnedIssueCustomField",
				"name":  "Subsystem",
				"value": []map[string]interface{}{
					{
						"$type": "OwnedBundleElement",
						"name":  subsystem,
					},
				},
			})
		}
	}

	if len(customFields) > 0 {
		payload["customFields"] = customFields
	}

	return s.createOrUpdateIssue(settings, "", payload)
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
		"description": fmt.Sprintf("%s\n\n[Synced from Asana ID: %s]", task.Notes, task.GID),
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

	// Add subsystem mapping
	asanaTags := asanaService.GetTags(task)
	if len(asanaTags) > 0 {
		tagMapper := NewTagMapper()
		primaryTag := asanaTags[0]
		subsystem := tagMapper.MapTagToSubsystem(primaryTag)
		if subsystem != "" {
			customFields = append(customFields, map[string]interface{}{
				"$type": "MultiOwnedIssueCustomField",
				"name":  "Subsystem",
				"value": []map[string]interface{}{
					{
						"$type": "OwnedBundleElement",
						"name":  subsystem,
					},
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
		"description": fmt.Sprintf("%s\n\n[Synced from Asana ID: %s]", task.Notes, task.GID),
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

	// Add subsystem mapping
	asanaTags := asanaService.GetTags(task)
	if len(asanaTags) > 0 {
		tagMapper := NewTagMapper()
		primaryTag := asanaTags[0]
		subsystem := tagMapper.MapTagToSubsystem(primaryTag)
		if subsystem != "" {
			customFields = append(customFields, map[string]interface{}{
				"$type": "MultiOwnedIssueCustomField",
				"name":  "Subsystem",
				"value": []map[string]interface{}{
					{
						"$type": "OwnedBundleElement",
						"name":  subsystem,
					},
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

// GetStatus extracts status from a YouTrack issue
func (s *YouTrackService) GetStatus(issue YouTrackIssue) string {
	for _, field := range issue.CustomFields {
		if field.Name == "State" {
			switch value := field.Value.(type) {
			case map[string]interface{}:
				if name, ok := value["localizedName"].(string); ok && name != "" {
					return name
				}
				if name, ok := value["name"].(string); ok && name != "" {
					return name
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