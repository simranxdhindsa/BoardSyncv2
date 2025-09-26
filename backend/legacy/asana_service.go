package legacy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	configpkg "asana-youtrack-sync/config"
)

// AsanaService handles Asana API operations with user-specific settings
type AsanaService struct {
	configService *configpkg.Service
}

// NewAsanaService creates a new Asana service
func NewAsanaService(configService *configpkg.Service) *AsanaService {
	return &AsanaService{
		configService: configService,
	}
}

// GetTasks retrieves tasks from Asana using user settings
func (s *AsanaService) GetTasks(userID int) ([]AsanaTask, error) {
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user settings: %w", err)
	}

	if settings.AsanaPAT == "" || settings.AsanaProjectID == "" {
		return nil, fmt.Errorf("asana credentials not configured")
	}

	url := fmt.Sprintf("https://app.asana.com/api/1.0/projects/%s/tasks?opt_fields=gid,name,notes,completed_at,created_at,modified_at,memberships.section.gid,memberships.section.name,tags.gid,tags.name",
		settings.AsanaProjectID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+settings.AsanaPAT)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("asana API error: %d - %s", resp.StatusCode, string(body))
	}

	var asanaResp AsanaResponse
	if err := json.NewDecoder(resp.Body).Decode(&asanaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	fmt.Printf("Retrieved %d Asana tasks for user %d\n", len(asanaResp.Data), userID)
	return asanaResp.Data, nil
}

// DeleteTask deletes an Asana task
func (s *AsanaService) DeleteTask(userID int, taskID string) error {
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return fmt.Errorf("failed to get user settings: %w", err)
	}

	if settings.AsanaPAT == "" {
		return fmt.Errorf("asana PAT not configured")
	}

	url := fmt.Sprintf("https://app.asana.com/api/1.0/tasks/%s", taskID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+settings.AsanaPAT)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("delete request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("asana delete error: %d - %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Successfully deleted Asana task: %s for user %d\n", taskID, userID)
	return nil
}

// GetTags extracts tags from an Asana task
func (s *AsanaService) GetTags(task AsanaTask) []string {
	var tags []string
	for _, tag := range task.Tags {
		if tag.Name != "" {
			tags = append(tags, tag.Name)
		}
	}
	return tags
}

// GetSectionName returns the section name of a task
func (s *AsanaService) GetSectionName(task AsanaTask) string {
	if len(task.Memberships) == 0 {
		return "No Section"
	}
	return strings.ToLower(task.Memberships[0].Section.Name)
}

// MapStateToYouTrack maps Asana section to YouTrack state
func (s *AsanaService) MapStateToYouTrack(task AsanaTask) string {
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

// FilterTasksByColumns filters Asana tasks by specified columns
func (s *AsanaService) FilterTasksByColumns(tasks []AsanaTask, selectedColumns []string) []AsanaTask {
	if len(selectedColumns) == 0 {
		fmt.Printf("FILTER DEBUG: No columns specified, returning all %d tasks\n", len(tasks))
		return tasks
	}

	fmt.Printf("FILTER DEBUG: Filtering %d tasks by columns: %v\n", len(tasks), selectedColumns)
	filtered := []AsanaTask{}

	for i, task := range tasks {
		if len(task.Memberships) > 0 {
			sectionName := strings.ToLower(strings.TrimSpace(task.Memberships[0].Section.Name))

			// Debug first few tasks
			if i < 5 {
				fmt.Printf("FILTER DEBUG: Task %d '%s' is in section '%s'\n", i, task.Name, sectionName)
			}

			matchFound := false
			for _, selectedCol := range selectedColumns {
				selectedColLower := strings.ToLower(strings.TrimSpace(selectedCol))

				var matches bool
				switch selectedColLower {
				case "backlog":
					matches = strings.Contains(sectionName, "backlog") &&
						!strings.Contains(sectionName, "dev") &&
						!strings.Contains(sectionName, "stage") &&
						!strings.Contains(sectionName, "blocked") &&
						!strings.Contains(sectionName, "progress")
				case "in progress":
					matches = strings.Contains(sectionName, "in progress") ||
						(strings.Contains(sectionName, "progress") && !strings.Contains(sectionName, "backlog"))
				case "dev":
					matches = strings.Contains(sectionName, "dev") &&
						!strings.Contains(sectionName, "ready")
				case "stage":
					matches = strings.Contains(sectionName, "stage") &&
						!strings.Contains(sectionName, "ready")
				case "blocked":
					matches = strings.Contains(sectionName, "blocked")
				case "ready for stage":
					matches = strings.Contains(sectionName, "ready") && strings.Contains(sectionName, "stage")
				case "findings":
					matches = strings.Contains(sectionName, "findings")
				default:
					matches = strings.Contains(sectionName, selectedColLower)
				}

				if matches {
					matchFound = true
					if i < 10 {
						fmt.Printf("FILTER DEBUG: ✓ Task '%s' matches column '%s'\n", task.Name, selectedColLower)
					}
					break
				}
			}

			if matchFound {
				filtered = append(filtered, task)
			}
		}
	}

	fmt.Printf("FILTER DEBUG: Filtered %d tasks from %d total for columns: %v\n", len(filtered), len(tasks), selectedColumns)
	return filtered
}
