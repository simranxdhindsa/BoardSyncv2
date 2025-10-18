package database

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// User represents a user in the system
type User struct {
	ID           int       `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"password_hash" db:"password_hash"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// UserSettings represents user configuration
type UserSettings struct {
	ID                  int                 `json:"id" db:"id"`
	UserID              int                 `json:"user_id" db:"user_id"`
	AsanaPAT            string              `json:"asana_pat" db:"asana_pat"`
	YouTrackBaseURL     string              `json:"youtrack_base_url" db:"youtrack_base_url"`
	YouTrackToken       string              `json:"youtrack_token" db:"youtrack_token"`
	AsanaProjectID      string              `json:"asana_project_id" db:"asana_project_id"`
	YouTrackProjectID   string              `json:"youtrack_project_id" db:"youtrack_project_id"`
	YouTrackBoardID     string              `json:"youtrack_board_id" db:"youtrack_board_id"`
	CustomFieldMappings CustomFieldMappings `json:"custom_field_mappings" db:"custom_field_mappings"`
	ColumnMappings      ColumnMappings      `json:"column_mappings" db:"column_mappings"`
	CreatedAt           time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time           `json:"updated_at" db:"updated_at"`
}

// CustomFieldMappings represents custom field mapping configuration
type CustomFieldMappings struct {
	TagMapping      map[string]string `json:"tag_mapping"`
	PriorityMapping map[string]string `json:"priority_mapping"`
	StatusMapping   map[string]string `json:"status_mapping"`
	CustomFields    map[string]string `json:"custom_fields"`
}

// Value implements the driver.Valuer interface for JSON storage
func (cfm CustomFieldMappings) Value() (driver.Value, error) {
	return json.Marshal(cfm)
}

// Scan implements the sql.Scanner interface for JSON retrieval
func (cfm *CustomFieldMappings) Scan(value interface{}) error {
	if value == nil {
		*cfm = CustomFieldMappings{}
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, cfm)
	case string:
		return json.Unmarshal([]byte(v), cfm)
	default:
		*cfm = CustomFieldMappings{}
		return nil
	}
}

// ColumnMapping represents a single column mapping configuration
type ColumnMapping struct {
	AsanaColumn    string `json:"asana_column"`
	YouTrackStatus string `json:"youtrack_status"`
	DisplayOnly    bool   `json:"display_only"`
}

// ColumnMappings represents bidirectional column mappings
type ColumnMappings struct {
	AsanaToYouTrack []ColumnMapping `json:"asana_to_youtrack"`
	YouTrackToAsana []ColumnMapping `json:"youtrack_to_asana"` // For future bidirectional sync
}

// Value implements the driver.Valuer interface for JSON storage
func (cm ColumnMappings) Value() (driver.Value, error) {
	return json.Marshal(cm)
}

// Scan implements the sql.Scanner interface for JSON retrieval
func (cm *ColumnMappings) Scan(value interface{}) error {
	if value == nil {
		*cm = ColumnMappings{
			AsanaToYouTrack: []ColumnMapping{},
			YouTrackToAsana: []ColumnMapping{},
		}
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, cm)
	case string:
		return json.Unmarshal([]byte(v), cm)
	default:
		*cm = ColumnMappings{
			AsanaToYouTrack: []ColumnMapping{},
			YouTrackToAsana: []ColumnMapping{},
		}
		return nil
	}
}

// SyncOperation represents a sync operation record
type SyncOperation struct {
	ID            int                    `json:"id" db:"id"`
	UserID        int                    `json:"user_id" db:"user_id"`
	OperationType string                 `json:"operation_type" db:"operation_type"`
	OperationData map[string]interface{} `json:"operation_data" db:"operation_data"`
	Status        string                 `json:"status" db:"status"`
	ErrorMessage  *string                `json:"error_message,omitempty" db:"error_message"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
}

// OperationData represents the operation data as JSON
type OperationData map[string]interface{}

// Value implements the driver.Valuer interface for JSON storage
func (od OperationData) Value() (driver.Value, error) {
	if len(od) == 0 {
		return "{}", nil
	}
	return json.Marshal(od)
}

// Scan implements the sql.Scanner interface for JSON retrieval
func (od *OperationData) Scan(value interface{}) error {
	if value == nil {
		*od = make(OperationData)
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, od)
	case string:
		return json.Unmarshal([]byte(v), od)
	default:
		*od = make(OperationData)
		return nil
	}
}

// IgnoredTicket represents an ignored ticket for a user's project
type IgnoredTicket struct {
	ID             int       `json:"id" db:"id"`
	UserID         int       `json:"user_id" db:"user_id"`
	AsanaProjectID string    `json:"asana_project_id" db:"asana_project_id"`
	TicketID       string    `json:"ticket_id" db:"ticket_id"`
	IgnoreType     string    `json:"ignore_type" db:"ignore_type"` // "temp" or "forever"
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// TicketMapping represents a manual mapping between Asana task and YouTrack issue
type TicketMapping struct {
	ID                int       `json:"id" db:"id"`
	UserID            int       `json:"user_id" db:"user_id"`
	AsanaProjectID    string    `json:"asana_project_id" db:"asana_project_id"`
	AsanaTaskID       string    `json:"asana_task_id" db:"asana_task_id"`
	YouTrackProjectID string    `json:"youtrack_project_id" db:"youtrack_project_id"`
	YouTrackIssueID   string    `json:"youtrack_issue_id" db:"youtrack_issue_id"` // e.g., "ARD-340"
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
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
}

// YouTrackBoard represents a YouTrack agile board
type YouTrackBoard struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AsanaSection represents an Asana section (column)
type AsanaSection struct {
	GID  string `json:"gid"`
	Name string `json:"name"`
}

// YouTrackState represents a YouTrack workflow state
type YouTrackState struct {
	Name string `json:"name"`
}
