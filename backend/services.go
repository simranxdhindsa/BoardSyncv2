package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ENHANCED: Asana API Functions with Tag Support
func getAsanaTasks() ([]AsanaTask, error) {
	url := fmt.Sprintf("https://app.asana.com/api/1.0/projects/%s/tasks?opt_fields=gid,name,notes,completed_at,created_at,modified_at,memberships.section.gid,memberships.section.name,tags.gid,tags.name", config.AsanaProjectID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+config.AsanaPAT)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("asana API error: %d - %s", resp.StatusCode, string(body))
	}

	var asanaResp AsanaResponse
	if err := json.NewDecoder(resp.Body).Decode(&asanaResp); err != nil {
		return nil, err
	}

	return asanaResp.Data, nil
}

// NEW: Delete Asana Task
func deleteAsanaTask(taskID string) error {
	url := fmt.Sprintf("https://app.asana.com/api/1.0/tasks/%s", taskID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+config.AsanaPAT)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("network error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("asana delete error: %d - %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Successfully deleted Asana task: %s\n", taskID)
	return nil
}

// ENHANCED: YouTrack API Functions with Subsystem Support
func getYouTrackIssues() ([]YouTrackIssue, error) {
	fmt.Printf("Connecting to YouTrack Cloud: %s\n", config.YouTrackBaseURL)
	fmt.Printf("Looking for project: %s\n", config.YouTrackProjectID)

	approaches := []func() ([]YouTrackIssue, error){
		getYouTrackIssuesWithQuery,
		getYouTrackIssuesSimpleCloud,
		getYouTrackIssuesViaProjects,
	}

	for i, approach := range approaches {
		fmt.Printf("Attempting approach %d...\n", i+1)
		issues, err := approach()
		if err == nil && len(issues) >= 0 {
			fmt.Printf("Approach %d succeeded! Found %d issues\n", i+1, len(issues))
			return issues, nil
		}
		fmt.Printf("Approach %d failed: %v\n", i+1, err)
	}

	return nil, fmt.Errorf("all approaches failed to connect to YouTrack Cloud")
}

// NEW: Delete YouTrack Issue
func deleteYouTrackIssue(issueID string) error {
	url := fmt.Sprintf("%s/api/issues/%s", config.YouTrackBaseURL, issueID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+config.YouTrackToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("network error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("youTrack delete error: %d - %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Successfully deleted YouTrack issue: %s\n", issueID)
	return nil
}

// NEW: Get ticket name for a given ID (for delete operations)
func getTicketName(ticketID string) string {
	// Try to get from current analysis or cache
	allTasks, err := getAsanaTasks()
	if err == nil {
		for _, task := range allTasks {
			if task.GID == ticketID {
				return task.Name
			}
		}
	}

	youTrackIssues, err := getYouTrackIssues()
	if err == nil {
		for _, issue := range youTrackIssues {
			asanaID := extractAsanaID(issue)
			if asanaID == ticketID {
				return issue.Summary
			}
			if issue.ID == ticketID {
				return issue.Summary
			}
		}
	}

	return fmt.Sprintf("Ticket-%s", ticketID) // Fallback name
}

// NEW: Find YouTrack issue ID by Asana task ID
func findYouTrackIssueByAsanaID(asanaTaskID string) (string, error) {
	youTrackIssues, err := getYouTrackIssues()
	if err != nil {
		return "", fmt.Errorf("failed to get YouTrack issues: %v", err)
	}

	for _, issue := range youTrackIssues {
		asanaID := extractAsanaID(issue)
		if asanaID == asanaTaskID {
			return issue.ID, nil
		}
	}

	return "", fmt.Errorf("no YouTrack issue found for Asana task %s", asanaTaskID)
}

// NEW: Bulk delete tickets
func performBulkDelete(ticketIDs []string, source string) DeleteResponse {
	response := DeleteResponse{
		Source:         source,
		RequestedCount: len(ticketIDs),
		Results:        make([]DeleteResult, 0, len(ticketIDs)),
	}

	for _, ticketID := range ticketIDs {
		result := DeleteResult{
			TicketID:   ticketID,
			TicketName: getTicketName(ticketID),
		}

		switch source {
		case "asana":
			err := deleteAsanaTask(ticketID)
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

		case "youtrack":
			// For YouTrack deletion, we need to check if ticketID is Asana ID or YouTrack ID
			var youtrackIssueID string
			var err error

			// First try to use as direct YouTrack issue ID
			youtrackIssueID = ticketID
			err = deleteYouTrackIssue(youtrackIssueID)

			// If that fails, try to find YouTrack issue by Asana ID
			if err != nil {
				youtrackIssueID, findErr := findYouTrackIssueByAsanaID(ticketID)
				if findErr != nil {
					result.Status = "failed"
					result.YouTrackResult = "failed"
					result.Error = fmt.Sprintf("Issue not found: %v", findErr)
					response.FailureCount++
				} else {
					err = deleteYouTrackIssue(youtrackIssueID)
					if err != nil {
						result.Status = "failed"
						result.YouTrackResult = "failed"
						result.Error = err.Error()
						response.FailureCount++
					} else {
						result.Status = "success"
						result.YouTrackResult = "deleted"
						response.SuccessCount++
					}
				}
			} else {
				result.Status = "success"
				result.YouTrackResult = "deleted"
				response.SuccessCount++
			}

		case "both":
			asanaSuccess := true
			youtrackSuccess := true
			var errors []string

			// Delete from Asana
			err := deleteAsanaTask(ticketID)
			if err != nil {
				asanaSuccess = false
				result.AsanaResult = "failed"
				errors = append(errors, fmt.Sprintf("Asana: %v", err))
			} else {
				result.AsanaResult = "deleted"
			}

			// Delete from YouTrack
			youtrackIssueID, findErr := findYouTrackIssueByAsanaID(ticketID)
			if findErr != nil {
				youtrackSuccess = false
				result.YouTrackResult = "not_found"
				errors = append(errors, fmt.Sprintf("YouTrack: %v", findErr))
			} else {
				err = deleteYouTrackIssue(youtrackIssueID)
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

		default:
			result.Status = "failed"
			result.Error = "Invalid source specified"
			response.FailureCount++
		}

		response.Results = append(response.Results, result)
	}

	// Set overall status
	if response.SuccessCount == response.RequestedCount {
		response.Status = "success"
		response.Summary = fmt.Sprintf("Successfully deleted all %d tickets from %s", response.SuccessCount, source)
	} else if response.SuccessCount > 0 {
		response.Status = "partial"
		response.Summary = fmt.Sprintf("Deleted %d of %d tickets from %s (%d failed)", response.SuccessCount, response.RequestedCount, source, response.FailureCount)
	} else {
		response.Status = "failed"
		response.Summary = fmt.Sprintf("Failed to delete any tickets from %s", source)
	}

	return response
}

func getYouTrackIssuesWithQuery() ([]YouTrackIssue, error) {
	queries := []string{
		fmt.Sprintf("project:%s", config.YouTrackProjectID),
		fmt.Sprintf("project: %s", config.YouTrackProjectID),
		fmt.Sprintf("#%s", config.YouTrackProjectID),
	}

	fields := "id,summary,description,created,updated,customFields(name,value(name,localizedName,description,id,$type,color)),project(shortName)"

	for i, query := range queries {
		fmt.Printf("   Query format %d: %s\n", i+1, query)

		encodedQuery := strings.ReplaceAll(query, " ", "%20")
		url := fmt.Sprintf("%s/api/issues?fields=%s&query=%s&top=200",
			config.YouTrackBaseURL, fields, encodedQuery)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			continue
		}

		req.Header.Set("Authorization", "Bearer "+config.YouTrackToken)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Cache-Control", "no-cache")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("   Network error: %v\n", err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		fmt.Printf("   Status: %d\n", resp.StatusCode)

		if resp.StatusCode == http.StatusOK {
			var issues []YouTrackIssue
			if err := json.Unmarshal(body, &issues); err != nil {
				fmt.Printf("   JSON error: %v\n", err)
				continue
			}
			return issues, nil
		}
	}

	return nil, fmt.Errorf("query approach failed")
}

func getYouTrackIssuesSimpleCloud() ([]YouTrackIssue, error) {
	fmt.Println("   Trying simple issues endpoint...")

	url := fmt.Sprintf("%s/api/issues?fields=id,summary,description,created,updated,customFields(name,value(name,localizedName,description,id,$type)),project(shortName)&top=200",
		config.YouTrackBaseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+config.YouTrackToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("   Status: %d\n", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		bodyStr := string(body)
		if len(bodyStr) > 300 {
			bodyStr = bodyStr[:300] + "..."
		}
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, bodyStr)
	}

	var allIssues []YouTrackIssue

	if err := json.Unmarshal(body, &allIssues); err != nil {
		return nil, fmt.Errorf("JSON parsing error: %v", err)
	}

	var projectIssues []YouTrackIssue
	fmt.Printf("   Filtering %d total issues for project '%s'\n", len(allIssues), config.YouTrackProjectID)

	for _, issue := range allIssues {
		if issue.Project.ShortName == config.YouTrackProjectID {
			projectIssues = append(projectIssues, issue)
		}
	}

	return projectIssues, nil
}

func getYouTrackIssuesViaProjects() ([]YouTrackIssue, error) {
	fmt.Println("   Trying project-specific endpoint...")

	url := fmt.Sprintf("%s/api/admin/projects/%s/issues?fields=id,summary,description,created,updated,customFields(name,value(name,localizedName)),project(shortName)&top=200",
		config.YouTrackBaseURL, config.YouTrackProjectID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+config.YouTrackToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("   Status: %d\n", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("project endpoint failed with status %d", resp.StatusCode)
	}

	var issues []YouTrackIssue
	if err := json.Unmarshal(body, &issues); err != nil {
		return nil, fmt.Errorf("JSON parsing error: %v", err)
	}

	return issues, nil
}

func findYouTrackProject() (string, error) {
	fmt.Println("Testing YouTrack Cloud connection...")
	fmt.Printf("URL: %s\n", config.YouTrackBaseURL)
	fmt.Printf("Project: %s\n", config.YouTrackProjectID)

	url := fmt.Sprintf("%s/api/admin/projects?fields=id,name,shortName&top=10", config.YouTrackBaseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+config.YouTrackToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Cache-Control", "no-cache")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("connection failed: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Response status: %d\n", resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Trying alternative projects endpoint...")
		return findYouTrackProjectAlternative()
	}

	var projects []struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		ShortName string `json:"shortName"`
	}

	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&projects); err != nil {
		fmt.Printf("JSON decode error: %v\n", err)
		return "", err
	}

	fmt.Printf("Found %d projects\n", len(projects))

	for _, proj := range projects {
		if proj.ID == config.YouTrackProjectID || proj.ShortName == config.YouTrackProjectID {
			fmt.Printf("Found matching project: %s (%s)\n", proj.Name, proj.ShortName)
			return proj.ShortName, nil
		}
	}

	return "", fmt.Errorf("project '%s' not found", config.YouTrackProjectID)
}

func findYouTrackProjectAlternative() (string, error) {
	url := fmt.Sprintf("%s/api/projects?fields=id,name,shortName", config.YouTrackBaseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+config.YouTrackToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("alternative connection failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Alternative endpoint status: %d\n", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("alternative endpoint failed: %d", resp.StatusCode)
	}

	var projects []struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		ShortName string `json:"shortName"`
	}

	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&projects); err != nil {
		return "", fmt.Errorf("JSON decode error: %v", err)
	}

	fmt.Printf("Alternative endpoint found %d projects\n", len(projects))

	for _, proj := range projects {
		if proj.ID == config.YouTrackProjectID || proj.ShortName == config.YouTrackProjectID {
			fmt.Printf("Found project: %s (%s)\n", proj.Name, proj.ShortName)
			return proj.ShortName, nil
		}
	}

	return "", fmt.Errorf("project '%s' not found in %d available projects", config.YouTrackProjectID, len(projects))
}

func listYouTrackProjects() {
	fmt.Println("Let me list all available projects...")

	url := fmt.Sprintf("%s/api/admin/projects?fields=id,name,shortName&top=20", config.YouTrackBaseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+config.YouTrackToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Cache-Control", "no-cache")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error connecting to YouTrack: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Projects API Response Status: %d\n", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Raw response: %s\n", string(body))
		return
	}

	var projects []struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		ShortName string `json:"shortName"`
	}

	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&projects); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		fmt.Printf("Response was: %s\n", string(body))
		return
	}

	if len(projects) == 0 {
		fmt.Println("No projects found - check your token permissions")
		return
	}

	fmt.Printf("Found %d projects:\n\n", len(projects))
	for i, proj := range projects {
		fmt.Printf("   %d. Name: %s\n", i+1, proj.Name)
		fmt.Printf("      Key: %s (use this in .env)\n", proj.ShortName)
		fmt.Printf("      ID: %s\n\n", proj.ID)
	}

	fmt.Println("Copy one of the 'Key' values above and update your .env file:")
	fmt.Printf("   YOUTRACK_PROJECT_ID=<paste_key_here>\n")
}

// ENHANCED: Create YouTrack Issue with Tag/Subsystem Support
func createYouTrackIssue(task AsanaTask) error {
	if isDuplicateTicket(task.Name) {
		return fmt.Errorf("ticket with title '%s' already exists in YouTrack", task.Name)
	}

	state := mapAsanaStateToYouTrack(task)

	if state == "FINDINGS_NO_SYNC" || state == "READY_FOR_STAGE_NO_SYNC" {
		return fmt.Errorf("cannot create ticket for display-only column")
	}

	payload := map[string]interface{}{
		"$type":       "Issue",
		"summary":     task.Name,
		"description": fmt.Sprintf("%s\n\n[Synced from Asana ID: %s]", task.Notes, task.GID),
		"project": map[string]interface{}{
			"$type":     "Project",
			"shortName": config.YouTrackProjectID,
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

	asanaTags := getAsanaTags(task)
	if len(asanaTags) > 0 {
		primaryTag := asanaTags[0]
		subsystem := mapTagToSubsystem(primaryTag)
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

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/issues", config.YouTrackBaseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+config.YouTrackToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("YouTrack create error: %d - %s", resp.StatusCode, string(body))
	}

	// FIXED: Only log tags if we have them
	if len(asanaTags) > 0 {
		fmt.Printf("Created ticket with tags: %v\n", asanaTags)
	}

	return nil
}

func isDuplicateTicket(title string) bool {
	query := fmt.Sprintf("project:%s summary:%s", config.YouTrackProjectID, title)
	encodedQuery := strings.ReplaceAll(query, " ", "%20")

	url := fmt.Sprintf("%s/api/issues?fields=id,summary&query=%s&top=5",
		config.YouTrackBaseURL, encodedQuery)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false
	}

	req.Header.Set("Authorization", "Bearer "+config.YouTrackToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	var issues []YouTrackIssue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return false
	}

	for _, issue := range issues {
		if strings.EqualFold(issue.Summary, title) {
			return true
		}
	}

	return false
}

// Analysis Functions
func performTicketAnalysis(selectedColumns []string) (*TicketAnalysis, error) {
	fmt.Printf("ANALYSIS DEBUG: Starting analysis for columns: %v\n", selectedColumns)

	allAsanaTasks, err := getAsanaTasks()
	if err != nil {
		return nil, fmt.Errorf("failed to get Asana tasks: %v", err)
	}

	fmt.Printf("ANALYSIS DEBUG: Retrieved %d total Asana tasks\n", len(allAsanaTasks))

	// CRITICAL: Filter tasks by the specified columns
	asanaTasks := filterAsanaTasksByColumns(allAsanaTasks, selectedColumns)
	fmt.Printf("ANALYSIS DEBUG: After filtering by columns %v: %d tasks remain\n", selectedColumns, len(asanaTasks))

	youTrackIssues, err := getYouTrackIssues()
	if err != nil {
		return nil, fmt.Errorf("failed to get YouTrack issues: %v", err)
	}

	fmt.Printf("ANALYSIS DEBUG: Retrieved %d YouTrack issues\n", len(youTrackIssues))

	youTrackMap := make(map[string]YouTrackIssue)
	asanaMap := make(map[string]AsanaTask)

	// Build maps for efficient lookup
	for _, issue := range youTrackIssues {
		asanaID := extractAsanaID(issue)
		if asanaID != "" {
			youTrackMap[asanaID] = issue
		}
	}

	for _, task := range asanaTasks { // Use filtered tasks
		asanaMap[task.GID] = task
	}

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
		Ignored:          getMapKeys(ignoredTicketsForever),
	}

	// Process only the filtered Asana tasks
	for _, task := range asanaTasks { // Use filtered tasks here too
		if isIgnored(task.GID) {
			continue
		}

		sectionName := getSectionName(task)
		asanaTags := getAsanaTags(task)

		if strings.Contains(sectionName, "findings") {
			analysis.FindingsTickets = append(analysis.FindingsTickets, task)

			if existingIssue, exists := youTrackMap[task.GID]; exists {
				youtrackStatus := getYouTrackStatus(existingIssue)
				if isActiveYouTrackStatus(youtrackStatus) {
					analysis.FindingsAlerts = append(analysis.FindingsAlerts, FindingsAlert{
						AsanaTask:      task,
						YouTrackIssue:  existingIssue,
						YouTrackStatus: youtrackStatus,
						AlertMessage:   fmt.Sprintf("HIGH ALERT: '%s' is in Findings (Asana) but still active in YouTrack (%s)", task.Name, youtrackStatus),
					})
				}
			}
			continue
		}

		if strings.Contains(sectionName, "ready for stage") {
			analysis.ReadyForStage = append(analysis.ReadyForStage, task)
			continue
		}

		if existingIssue, exists := youTrackMap[task.GID]; exists {
			asanaStatus := mapAsanaStateToYouTrack(task)
			youtrackStatus := getYouTrackStatus(existingIssue)

			if strings.Contains(sectionName, "blocked") {
				analysis.BlockedTickets = append(analysis.BlockedTickets, MatchedTicket{
					AsanaTask:         task,
					YouTrackIssue:     existingIssue,
					Status:            asanaStatus,
					AsanaTags:         asanaTags,
					YouTrackSubsystem: "",
					TagMismatch:       false,
				})
			} else if asanaStatus == youtrackStatus {
				analysis.Matched = append(analysis.Matched, MatchedTicket{
					AsanaTask:         task,
					YouTrackIssue:     existingIssue,
					Status:            asanaStatus,
					AsanaTags:         asanaTags,
					YouTrackSubsystem: "",
					TagMismatch:       false,
				})
			} else {
				analysis.Mismatched = append(analysis.Mismatched, MismatchedTicket{
					AsanaTask:         task,
					YouTrackIssue:     existingIssue,
					AsanaStatus:       asanaStatus,
					YouTrackStatus:    youtrackStatus,
					AsanaTags:         asanaTags,
					YouTrackSubsystem: "",
					TagMismatch:       false,
				})
			}
		} else {
			if isSyncableColumn(sectionName) {
				analysis.MissingYouTrack = append(analysis.MissingYouTrack, task)
			}
		}
	}

	// Handle orphaned YouTrack issues - only include those that would have been in the selected columns
	// This is trickier because we need to check if the YouTrack issue corresponds to an Asana task
	// that would have been in our filtered set
	for _, issue := range youTrackIssues {
		asanaID := extractAsanaID(issue)
		if asanaID != "" {
			// Check if this issue corresponds to a task that should have been in our analysis
			// First, check if it exists in our original unfiltered task set
			taskExists := false
			var originalTask AsanaTask
			for _, originalTask = range allAsanaTasks {
				if originalTask.GID == asanaID {
					taskExists = true
					break
				}
			}

			if taskExists {
				// Now check if this task would have been in our filtered set
				filteredTaskExists := false
				for _, filteredTask := range asanaTasks {
					if filteredTask.GID == asanaID {
						filteredTaskExists = true
						break
					}
				}

				// If the task exists in the original set but not in our filtered set,
				// and we don't have it in our asanaMap, then it's orphaned from our perspective
				if !filteredTaskExists {
					// This YouTrack issue corresponds to an Asana task that wasn't in our filtered columns
					// We should include it as orphaned only if the original task would have been syncable
					if len(originalTask.Memberships) > 0 {
						originalSectionName := strings.ToLower(originalTask.Memberships[0].Section.Name)
						if isSyncableColumn(originalSectionName) {
							analysis.OrphanedYouTrack = append(analysis.OrphanedYouTrack, issue)
						}
					}
				}
			} else {
				// The YouTrack issue references an Asana task that doesn't exist at all
				analysis.OrphanedYouTrack = append(analysis.OrphanedYouTrack, issue)
			}
		}
	}

	fmt.Printf("ANALYSIS DEBUG: Analysis complete: %d matched, %d mismatched, %d missing, %d orphaned\n",
		len(analysis.Matched), len(analysis.Mismatched), len(analysis.MissingYouTrack), len(analysis.OrphanedYouTrack))

	return analysis, nil
}

// FIXED: Complete updateYouTrackIssue function
func updateYouTrackIssue(issueID string, task AsanaTask) error {
	state := mapAsanaStateToYouTrack(task)

	if state == "FINDINGS_NO_SYNC" || state == "READY_FOR_STAGE_NO_SYNC" {
		return fmt.Errorf("cannot update ticket for display-only column")
	}

	payload := map[string]interface{}{
		"$type":       "Issue",
		"summary":     task.Name,
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

	asanaTags := getAsanaTags(task)
	if len(asanaTags) > 0 {
		primaryTag := asanaTags[0]
		subsystem := mapTagToSubsystem(primaryTag)
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

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/issues/%s", config.YouTrackBaseURL, issueID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+config.YouTrackToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		bodyStr := string(body)
		if strings.Contains(bodyStr, "incompatible-issue-custom-field-name-Subsystem") {
			return updateYouTrackIssueWithoutSubsystem(issueID, task)
		}
		return fmt.Errorf("YouTrack update error: %d - %s", resp.StatusCode, bodyStr)
	}

	if len(asanaTags) > 0 {
		fmt.Printf("Successfully updated ticket %s with tags: %v\n", issueID, asanaTags)
	}

	return nil
}

func updateYouTrackIssueWithoutSubsystem(issueID string, task AsanaTask) error {
	state := mapAsanaStateToYouTrack(task)

	payload := map[string]interface{}{
		"$type":       "Issue",
		"summary":     task.Name,
		"description": fmt.Sprintf("%s\n\n[Synced from Asana ID: %s]", task.Notes, task.GID),
	}

	if state != "" {
		payload["customFields"] = []map[string]interface{}{
			{
				"$type": "StateIssueCustomField",
				"name":  "State",
				"value": map[string]interface{}{
					"$type": "StateBundleElement",
					"name":  state,
				},
			},
		}
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/issues/%s", config.YouTrackBaseURL, issueID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+config.YouTrackToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("YouTrack update error: %d - %s", resp.StatusCode, string(body))
	}

	asanaTags := getAsanaTags(task)
	if len(asanaTags) > 0 {
		fmt.Printf("Updated ticket %s (status only - Subsystem field not available). Tags: %v\n", issueID, asanaTags)
	}

	return nil
}

// Helper Functions
func getAsanaTags(task AsanaTask) []string {
	var tags []string
	for _, tag := range task.Tags {
		if tag.Name != "" {
			tags = append(tags, tag.Name)
		}
	}
	return tags
}

func mapTagToSubsystem(asanaTag string) string {
	if subsystem, exists := defaultTagMapping[asanaTag]; exists {
		return subsystem
	}

	asanaTagLower := strings.ToLower(asanaTag)
	if subsystem, exists := defaultTagMapping[asanaTagLower]; exists {
		return subsystem
	}

	return strings.ToLower(asanaTag)
}

func mapAsanaStateToYouTrack(task AsanaTask) string {
	if len(task.Memberships) == 0 {
		return "Backlog"
	}

	sectionName := strings.ToLower(task.Memberships[0].Section.Name)

	switch {
	case strings.Contains(sectionName, "backlog"):
		return "Backlog"
	case strings.Contains(sectionName, "in progress"):
		return "In Progress"
	case strings.Contains(sectionName, "dev") && !strings.Contains(sectionName, "ready"):
		return "DEV"
	case strings.Contains(sectionName, "stage") && !strings.Contains(sectionName, "ready"):
		return "STAGE"
	case strings.Contains(sectionName, "blocked"):
		return "Blocked"
	case strings.Contains(sectionName, "findings"):
		return "FINDINGS_NO_SYNC"
	case strings.Contains(sectionName, "ready for stage"):
		return "READY_FOR_STAGE_NO_SYNC"
	default:
		return "Backlog"
	}
}

func getYouTrackStatus(issue YouTrackIssue) string {
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

func extractAsanaID(issue YouTrackIssue) string {
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

func getSectionName(task AsanaTask) string {
	if len(task.Memberships) == 0 {
		return "No Section"
	}
	return strings.ToLower(task.Memberships[0].Section.Name)
}

func isSyncableColumn(sectionName string) bool {
	sectionLower := strings.ToLower(strings.TrimSpace(sectionName))

	// Check against syncable columns with precise matching
	for _, col := range syncableColumns {
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
		default:
			if strings.Contains(sectionLower, colLower) {
				return true
			}
		}
	}
	return false
}

func isActiveYouTrackStatus(status string) bool {
	activeStatuses := []string{"Backlog", "In Progress", "DEV", "STAGE", "Blocked"}
	for _, activeStatus := range activeStatuses {
		if strings.EqualFold(status, activeStatus) {
			return true
		}
	}
	return false
}

func filterAsanaTasksByColumns(tasks []AsanaTask, selectedColumns []string) []AsanaTask {
	if len(selectedColumns) == 0 {
		fmt.Printf("FILTER DEBUG: No columns specified, returning all %d tasks\n", len(tasks))
		return tasks
	}

	fmt.Printf("FILTER DEBUG: Filtering %d tasks by columns: %v\n", len(tasks), selectedColumns)

	filtered := []AsanaTask{}

	for i, task := range tasks {
		if len(task.Memberships) > 0 {
			sectionName := strings.ToLower(strings.TrimSpace(task.Memberships[0].Section.Name))

			// Debug: Print first few tasks to see what sections we're getting
			if i < 5 {
				fmt.Printf("FILTER DEBUG: Task %d '%s' is in section '%s'\n", i, task.Name, sectionName)
			}

			// Check if task's section matches ANY of the selected columns
			matchFound := false
			for _, selectedCol := range selectedColumns {
				selectedColLower := strings.ToLower(strings.TrimSpace(selectedCol))

				// CRITICAL FIX: More precise matching logic
				var matches bool
				switch selectedColLower {
				case "backlog":
					// Must contain "backlog" and NOT be in other specific sections
					matches = strings.Contains(sectionName, "backlog") &&
						!strings.Contains(sectionName, "dev") &&
						!strings.Contains(sectionName, "stage") &&
						!strings.Contains(sectionName, "blocked") &&
						!strings.Contains(sectionName, "progress")

				case "in progress":
					// Must contain "progress" or "in progress"
					matches = strings.Contains(sectionName, "in progress") ||
						(strings.Contains(sectionName, "progress") && !strings.Contains(sectionName, "backlog"))

				case "dev":
					// Must contain "dev" but NOT "ready" (to avoid "ready for dev")
					matches = strings.Contains(sectionName, "dev") &&
						!strings.Contains(sectionName, "ready")

				case "stage":
					// Must contain "stage" but NOT "ready" (to avoid "ready for stage")
					matches = strings.Contains(sectionName, "stage") &&
						!strings.Contains(sectionName, "ready")

				case "blocked":
					// Must contain "blocked"
					matches = strings.Contains(sectionName, "blocked")

				case "ready for stage":
					// Must contain both "ready" AND "stage"
					matches = strings.Contains(sectionName, "ready") && strings.Contains(sectionName, "stage")

				case "findings":
					// Must contain "findings"
					matches = strings.Contains(sectionName, "findings")

				default:
					// Fallback: exact contains check
					matches = strings.Contains(sectionName, selectedColLower)
				}

				if matches {
					matchFound = true
					// Debug: Show which tasks match
					if i < 10 {
						fmt.Printf("FILTER DEBUG: ✓ Task '%s' (section: '%s') matches column '%s'\n",
							task.Name, sectionName, selectedColLower)
					}
					break
				}
			}

			if matchFound {
				filtered = append(filtered, task)
			} else {
				// Debug: Show which tasks are filtered out
				if i < 10 {
					fmt.Printf("FILTER DEBUG: ✗ Task '%s' (section: '%s') does NOT match any of %v\n",
						task.Name, sectionName, selectedColumns)
				}
			}
		} else {
			// Task has no section membership
			if i < 5 {
				fmt.Printf("FILTER DEBUG: Task %d '%s' has no section membership\n", i, task.Name)
			}
		}
	}

	fmt.Printf("FILTER DEBUG: *** RESULT: Filtered %d tasks from %d total for columns: %v ***\n",
		len(filtered), len(tasks), selectedColumns)

	// Debug: Show sample of filtered tasks
	for i, task := range filtered {
		if i < 3 {
			sectionName := "No Section"
			if len(task.Memberships) > 0 {
				sectionName = task.Memberships[0].Section.Name
			}
			fmt.Printf("FILTER DEBUG: Filtered Task %d: '%s' (Section: %s)\n", i+1, task.Name, sectionName)
		}
	}

	return filtered
}

func isIgnored(ticketID string) bool {
	return ignoredTicketsTemp[ticketID] || ignoredTicketsForever[ticketID]
}

func getMapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func loadIgnoredTickets() {
	data, err := os.ReadFile("ignored_tickets.json")
	if err != nil {
		return
	}

	var ignored []string
	if err := json.Unmarshal(data, &ignored); err != nil {
		return
	}

	for _, id := range ignored {
		ignoredTicketsForever[id] = true
	}
}

func saveIgnoredTickets() {
	ignored := getMapKeys(ignoredTicketsForever)
	data, _ := json.MarshalIndent(ignored, "", "  ")
	os.WriteFile("ignored_tickets.json", data, 0644)
}

// simran
