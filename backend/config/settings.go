package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"asana-youtrack-sync/database"
)

// UserSettings represents user configuration
type UserSettings struct {
<<<<<<< HEAD
	ID                  int                 `json:"id"`
	UserID              int                 `json:"user_id"`
	AsanaPAT            string              `json:"asana_pat"`
	YouTrackBaseURL     string              `json:"youtrack_base_url"`
	YouTrackToken       string              `json:"youtrack_token"`
	AsanaProjectID      string              `json:"asana_project_id"`
	YouTrackProjectID   string              `json:"youtrack_project_id"`
	YouTrackBoardID     string              `json:"youtrack_board_id"`
	CustomFieldMappings CustomFieldMappings `json:"custom_field_mappings"`
	CreatedAt           time.Time           `json:"created_at"`
	UpdatedAt           time.Time           `json:"updated_at"`
=======
	ID                  int                        `json:"id"`
	UserID              int                        `json:"user_id"`
	AsanaPAT            string                     `json:"asana_pat"`
	YouTrackBaseURL     string                     `json:"youtrack_base_url"`
	YouTrackToken       string                     `json:"youtrack_token"`
	AsanaProjectID      string                     `json:"asana_project_id"`
	YouTrackProjectID   string                     `json:"youtrack_project_id"`
	YouTrackBoardID     string                     `json:"youtrack_board_id"`
	CustomFieldMappings CustomFieldMappings        `json:"custom_field_mappings"`
	ColumnMappings      database.ColumnMappings    `json:"column_mappings"`
	CreatedAt           time.Time                  `json:"created_at"`
	UpdatedAt           time.Time                  `json:"updated_at"`
>>>>>>> features
}

// CustomFieldMappings represents custom field mapping configuration
type CustomFieldMappings struct {
	TagMapping      map[string]string `json:"tag_mapping"`
	PriorityMapping map[string]string `json:"priority_mapping"`
	StatusMapping   map[string]string `json:"status_mapping"`
	CustomFields    map[string]string `json:"custom_fields"`
}

// UpdateSettingsRequest represents a settings update request
type UpdateSettingsRequest struct {
<<<<<<< HEAD
	AsanaPAT            string              `json:"asana_pat"`
	YouTrackBaseURL     string              `json:"youtrack_base_url"`
	YouTrackToken       string              `json:"youtrack_token"`
	AsanaProjectID      string              `json:"asana_project_id"`
	YouTrackProjectID   string              `json:"youtrack_project_id"`
	YouTrackBoardID     string              `json:"youtrack_board_id"`
	CustomFieldMappings CustomFieldMappings `json:"custom_field_mappings"`
=======
	AsanaPAT            string                   `json:"asana_pat"`
	YouTrackBaseURL     string                   `json:"youtrack_base_url"`
	YouTrackToken       string                   `json:"youtrack_token"`
	AsanaProjectID      string                   `json:"asana_project_id"`
	YouTrackProjectID   string                   `json:"youtrack_project_id"`
	YouTrackBoardID     string                   `json:"youtrack_board_id"`
	CustomFieldMappings CustomFieldMappings      `json:"custom_field_mappings"`
	ColumnMappings      database.ColumnMappings  `json:"column_mappings"`
>>>>>>> features
}

// Project represents project information for dropdowns
type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AsanaProject represents an Asana project
type AsanaProject struct {
	GID  string `json:"gid"`
	Name string `json:"name"`
}

// YouTrackProject represents a YouTrack project
type YouTrackProject struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
	Archived  bool   `json:"archived"`
	RingId    string `json:"ringId"`
}

// Service handles settings management
type Service struct {
	db *database.DB
}

// NewService creates a new settings service
func NewService(db *database.DB) *Service {
	return &Service{db: db}
}

// GetSettings retrieves user settings
func (s *Service) GetSettings(userID int) (*UserSettings, error) {
	settings, err := s.db.GetUserSettings(userID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	return &UserSettings{
		ID:                settings.ID,
		UserID:            settings.UserID,
		AsanaPAT:          settings.AsanaPAT,
		YouTrackBaseURL:   settings.YouTrackBaseURL,
		YouTrackToken:     settings.YouTrackToken,
		AsanaProjectID:    settings.AsanaProjectID,
		YouTrackProjectID: settings.YouTrackProjectID,
		YouTrackBoardID:   settings.YouTrackBoardID,
		CustomFieldMappings: CustomFieldMappings{
			TagMapping:      settings.CustomFieldMappings.TagMapping,
			PriorityMapping: settings.CustomFieldMappings.PriorityMapping,
			StatusMapping:   settings.CustomFieldMappings.StatusMapping,
			CustomFields:    settings.CustomFieldMappings.CustomFields,
		},
		ColumnMappings: settings.ColumnMappings,
		CreatedAt:      settings.CreatedAt,
		UpdatedAt:      settings.UpdatedAt,
	}, nil
}

// UpdateSettings updates user settings
func (s *Service) UpdateSettings(userID int, req UpdateSettingsRequest) (*UserSettings, error) {
	// Initialize mappings if nil
	if req.CustomFieldMappings.TagMapping == nil {
		req.CustomFieldMappings.TagMapping = make(map[string]string)
	}
	if req.CustomFieldMappings.PriorityMapping == nil {
		req.CustomFieldMappings.PriorityMapping = make(map[string]string)
	}
	if req.CustomFieldMappings.StatusMapping == nil {
		req.CustomFieldMappings.StatusMapping = make(map[string]string)
	}
	if req.CustomFieldMappings.CustomFields == nil {
		req.CustomFieldMappings.CustomFields = make(map[string]string)
	}

	updatedSettings, err := s.db.UpdateUserSettings(
		userID,
		req.AsanaPAT,
		req.YouTrackBaseURL,
		req.YouTrackToken,
		req.AsanaProjectID,
		req.YouTrackProjectID,
		req.YouTrackBoardID,
		database.CustomFieldMappings{
			TagMapping:      req.CustomFieldMappings.TagMapping,
			PriorityMapping: req.CustomFieldMappings.PriorityMapping,
			StatusMapping:   req.CustomFieldMappings.StatusMapping,
			CustomFields:    req.CustomFieldMappings.CustomFields,
		},
		req.ColumnMappings,
	)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	return &UserSettings{
<<<<<<< HEAD
		ID:                updatedSettings.ID,
		UserID:            updatedSettings.UserID,
		AsanaPAT:          updatedSettings.AsanaPAT,
		YouTrackBaseURL:   updatedSettings.YouTrackBaseURL,
		YouTrackToken:     updatedSettings.YouTrackToken,
		AsanaProjectID:    updatedSettings.AsanaProjectID,
		YouTrackProjectID: updatedSettings.YouTrackProjectID,
		YouTrackBoardID:   updatedSettings.YouTrackBoardID,
=======
		ID:                  updatedSettings.ID,
		UserID:              updatedSettings.UserID,
		AsanaPAT:            updatedSettings.AsanaPAT,
		YouTrackBaseURL:     updatedSettings.YouTrackBaseURL,
		YouTrackToken:       updatedSettings.YouTrackToken,
		AsanaProjectID:      updatedSettings.AsanaProjectID,
		YouTrackProjectID:   updatedSettings.YouTrackProjectID,
		YouTrackBoardID:     updatedSettings.YouTrackBoardID,
>>>>>>> features
		CustomFieldMappings: CustomFieldMappings{
			TagMapping:      updatedSettings.CustomFieldMappings.TagMapping,
			PriorityMapping: updatedSettings.CustomFieldMappings.PriorityMapping,
			StatusMapping:   updatedSettings.CustomFieldMappings.StatusMapping,
			CustomFields:    updatedSettings.CustomFieldMappings.CustomFields,
		},
		ColumnMappings: updatedSettings.ColumnMappings,
		CreatedAt:      updatedSettings.CreatedAt,
		UpdatedAt:      updatedSettings.UpdatedAt,
	}, nil
}

// GetAsanaProjects fetches Asana projects using user's PAT
func (s *Service) GetAsanaProjects(userID int) ([]Project, error) {
	settings, err := s.GetSettings(userID)
	if err != nil {
		return nil, err
	}

	if settings.AsanaPAT == "" {
		return nil, fmt.Errorf("Asana PAT not configured")
	}

	req, err := http.NewRequest("GET", "https://app.asana.com/api/1.0/projects", nil)
	if err != nil {
		return nil, fmt.Errorf("request creation error: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+settings.AsanaPAT)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Asana API error: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("response read error: %w", err)
	}

	var response struct {
		Data []AsanaProject `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("JSON unmarshal error: %w", err)
	}

	projects := make([]Project, len(response.Data))
	for i, project := range response.Data {
		projects[i] = Project{
			ID:   project.GID,
			Name: project.Name,
		}
	}

	return projects, nil
}

// GetYouTrackProjects fetches YouTrack projects using user's token
func (s *Service) GetYouTrackProjects(userID int) ([]Project, error) {
	settings, err := s.GetSettings(userID)
	if err != nil {
		return nil, err
	}

	if settings.YouTrackBaseURL == "" || settings.YouTrackToken == "" {
		return nil, fmt.Errorf("YouTrack credentials not configured")
	}

	// Try multiple API endpoints
	endpoints := []string{
		fmt.Sprintf("%s/api/admin/projects?fields=id,name,shortName,archived&$top=50&archived=false", settings.YouTrackBaseURL),
		fmt.Sprintf("%s/api/projects?fields=id,name,shortName,archived&$top=50", settings.YouTrackBaseURL),
		fmt.Sprintf("%s/api/admin/projects?fields=shortName,name&$top=50", settings.YouTrackBaseURL),
		fmt.Sprintf("%s/api/admin/projects", settings.YouTrackBaseURL),
	}

	var lastError error

	for _, url := range endpoints {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			lastError = err
			continue
		}

		req.Header.Set("Authorization", "Bearer "+settings.YouTrackToken)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Cache-Control", "no-cache")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastError = err
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastError = err
			continue
		}

		if resp.StatusCode == http.StatusOK {
			var projects []YouTrackProject
			if err := json.Unmarshal(body, &projects); err != nil {
				lastError = err
				continue
			}

			result := make([]Project, len(projects))
			for i, project := range projects {
				projectKey := project.ShortName
				if projectKey == "" {
					projectKey = project.ID
				}

				displayName := project.Name
				if project.ShortName != "" {
					displayName = fmt.Sprintf("%s (%s)", project.Name, project.ShortName)
				}

				result[i] = Project{
					ID:   projectKey,
					Name: displayName,
				}
			}

			return result, nil
		} else {
			switch resp.StatusCode {
			case 401:
				return nil, fmt.Errorf("YouTrack authentication failed. Please check your token")
			case 403:
				return nil, fmt.Errorf("YouTrack access forbidden. Your token may not have sufficient permissions")
			case 404:
				lastError = fmt.Errorf("endpoint not found")
				continue
			default:
				lastError = fmt.Errorf("YouTrack API error: status %d", resp.StatusCode)
			}
		}
	}

	return nil, fmt.Errorf("all YouTrack endpoints failed: %v", lastError)
}

// GetYouTrackBoards fetches available YouTrack agile boards
func (s *Service) GetYouTrackBoards(userID int) ([]database.YouTrackBoard, error) {
	settings, err := s.GetSettings(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	if settings.YouTrackBaseURL == "" || settings.YouTrackToken == "" {
		return nil, fmt.Errorf("YouTrack credentials not configured")
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

// TestConnections tests API connections with current settings
func (s *Service) TestConnections(userID int) (map[string]bool, error) {
	results := make(map[string]bool)

	// Test Asana connection
	_, err := s.GetAsanaProjects(userID)
	results["asana"] = err == nil

	// Test YouTrack connection
	_, err = s.GetYouTrackProjects(userID)
	results["youtrack"] = err == nil

	return results, nil
}

// GetAsanaSections fetches Asana sections (columns) for the configured project
func (s *Service) GetAsanaSections(userID int) ([]database.AsanaSection, error) {
	settings, err := s.GetSettings(userID)
	if err != nil {
		return nil, err
	}

	if settings.AsanaPAT == "" || settings.AsanaProjectID == "" {
		return nil, fmt.Errorf("asana credentials not configured")
	}

	url := fmt.Sprintf("https://app.asana.com/api/1.0/projects/%s/sections", settings.AsanaProjectID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation error: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+settings.AsanaPAT)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("asana API error: %d - %s", resp.StatusCode, string(body))
	}

	var response struct {
		Data []database.AsanaSection `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("JSON decode error: %w", err)
	}

	return response.Data, nil
}

// GetYouTrackStates fetches YouTrack workflow states for the configured project
func (s *Service) GetYouTrackStates(userID int) ([]database.YouTrackState, error) {
	settings, err := s.GetSettings(userID)
	if err != nil {
		return nil, err
	}

	if settings.YouTrackBaseURL == "" || settings.YouTrackToken == "" || settings.YouTrackProjectID == "" {
		return nil, fmt.Errorf("youtrack credentials not configured")
	}

	url := fmt.Sprintf("%s/api/admin/projects/%s/customFields?fields=field(name,fieldType(id)),bundle(values(name))",
		settings.YouTrackBaseURL, settings.YouTrackProjectID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation error: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+settings.YouTrackToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request error: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("youtrack API error: %d - %s", resp.StatusCode, string(body))
	}

	var customFields []map[string]interface{}
	if err := json.Unmarshal(body, &customFields); err != nil {
		return nil, fmt.Errorf("JSON unmarshal error: %w", err)
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

	return states, nil
}

// GetYouTrackBoards fetches all agile boards from YouTrack
func (s *Service) GetYouTrackBoards(userID int) ([]database.YouTrackBoard, error) {
	settings, err := s.GetSettings(userID)
	if err != nil {
		return nil, err
	}

	if settings.YouTrackBaseURL == "" || settings.YouTrackToken == "" {
		return nil, fmt.Errorf("youtrack credentials not configured")
	}

	url := fmt.Sprintf("%s/api/agiles?$top=-1&fields=id,name,sprintsSettings(disableSprints),projects(id)",
		settings.YouTrackBaseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation error: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+settings.YouTrackToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request error: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("youtrack API error: %d - %s", resp.StatusCode, string(body))
	}

	var rawBoards []map[string]interface{}
	if err := json.Unmarshal(body, &rawBoards); err != nil {
		return nil, fmt.Errorf("JSON unmarshal error: %w", err)
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
