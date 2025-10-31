// backend/legacy/types.go - ENHANCED VERSION
package legacy

import (
	"time"

	"asana-youtrack-sync/database"
)

// Asana data structures
type AsanaTask struct {
	GID         string `json:"gid"`
	Name        string `json:"name"`
	Notes       string `json:"notes"`
	HTMLNotes   string `json:"html_notes"`
	CompletedAt string `json:"completed_at"`
	CreatedAt   string `json:"created_at"`
	ModifiedAt  string `json:"modified_at"`
	Assignee    struct {
		GID  string `json:"gid"`
		Name string `json:"name"`
	} `json:"assignee"`
	Memberships []struct {
		Section struct {
			GID  string `json:"gid"`
			Name string `json:"name"`
		} `json:"section"`
	} `json:"memberships"`
	Tags []struct {
		GID  string `json:"gid"`
		Name string `json:"name"`
	} `json:"tags"`
	CustomFields []struct {
		GID          string `json:"gid"`
		Name         string `json:"name"`
		DisplayValue string `json:"display_value"`
		TextValue    string `json:"text_value"`
		NumberValue  int    `json:"number_value"`
		EnumValue    struct {
			GID  string `json:"gid"`
			Name string `json:"name"`
		} `json:"enum_value"`
	} `json:"custom_fields"`
	Attachments []struct {
		GID          string `json:"gid"`
		Name         string `json:"name"`
		DownloadURL  string `json:"download_url"`
		ViewURL      string `json:"view_url"`
		ResourceType string `json:"resource_type"` // "image", "pdf", "document", etc.
		Host         string `json:"host"`          // "asana", "dropbox", "google", etc.
		Size         int64  `json:"size"`          // File size in bytes
	} `json:"attachments"`
}

type AsanaResponse struct {
	Data []AsanaTask `json:"data"`
}

// YouTrack data structures
type YouTrackIssue struct {
	ID           string                 `json:"id"`
	Summary      string                 `json:"summary"`
	Description  string                 `json:"description"`
	Created      int64                  `json:"created"`
	Updated      int64                  `json:"updated"`
	State        string                 `json:"state"`
	Subsystem    string                 `json:"subsystem"`
	CreatedBy    string                 `json:"created_by"`
	Attachments  []YouTrackAttachment   `json:"attachments"`
	CustomFields []struct {
		Name  string      `json:"name"`
		Value interface{} `json:"value"`
	} `json:"customFields"`
	Project struct {
		ShortName string `json:"shortName"`
	} `json:"project"`
}

type YouTrackAttachment struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	MimeType  string `json:"mimeType"`
	URL       string `json:"url"`
	Extension string `json:"extension"`
}

type YouTrackUser struct {
	ID       string `json:"id"`
	Login    string `json:"login"`
	FullName string `json:"fullName"`
	Email    string `json:"email"`
}

// Analysis result structures
type TicketAnalysis struct {
	SelectedColumn   string             `json:"selected_column"`
	Matched          []MatchedTicket    `json:"matched"`
	Mismatched       []MismatchedTicket `json:"mismatched"`
	MissingYouTrack  []AsanaTask        `json:"missing_youtrack"`
	FindingsTickets  []AsanaTask        `json:"findings_tickets"`
	FindingsAlerts   []FindingsAlert    `json:"findings_alerts"`
	ReadyForStage    []AsanaTask        `json:"ready_for_stage"`
	BlockedTickets   []MatchedTicket    `json:"blocked_tickets"`
	OrphanedYouTrack []YouTrackIssue    `json:"orphaned_youtrack"`
	Ignored          []string           `json:"ignored"`
}

type MatchedTicket struct {
	AsanaTask         AsanaTask     `json:"asana_task"`
	YouTrackIssue     YouTrackIssue `json:"youtrack_issue"`
	Status            string        `json:"status"`
	AsanaTags         []string      `json:"asana_tags"`
	YouTrackSubsystem string        `json:"youtrack_subsystem"`
	TagMismatch       bool          `json:"tag_mismatch"`
	// Enhanced fields
	AssigneeName string    `json:"assignee_name"`
	Priority     string    `json:"priority"`
	CreatedAt    time.Time `json:"created_at"`
}

type MismatchedTicket struct {
	AsanaTask         AsanaTask     `json:"asana_task"`
	YouTrackIssue     YouTrackIssue `json:"youtrack_issue"`
	AsanaStatus       string        `json:"asana_status"`
	YouTrackStatus    string        `json:"youtrack_status"`
	AsanaTags         []string      `json:"asana_tags"`
	YouTrackSubsystem string        `json:"youtrack_subsystem"`
	TagMismatch       bool          `json:"tag_mismatch"`
	// Enhanced fields
	AssigneeName      string    `json:"assignee_name"`
	Priority          string    `json:"priority"`
	CreatedAt         time.Time `json:"created_at"`
}

type FindingsAlert struct {
	AsanaTask      AsanaTask     `json:"asana_task"`
	YouTrackIssue  YouTrackIssue `json:"youtrack_issue"`
	YouTrackStatus string        `json:"youtrack_status"`
	AlertMessage   string        `json:"alert_message"`
}

// API request structures
type SyncRequest struct {
	TicketID string `json:"ticket_id"`
	Action   string `json:"action"`
}

type CreateSingleRequest struct {
	TaskID string `json:"task_id"`
}

type IgnoreRequest struct {
	TicketID string `json:"ticket_id"`
	Action   string `json:"action"`
	Type     string `json:"type"`
}

// Sorting and Filtering structures
type TicketFilter struct {
	Assignees  []string  `json:"assignees"`  // Filter by assignee names
	StartDate  time.Time `json:"start_date"` // Filter by created date range
	EndDate    time.Time `json:"end_date"`
	Priority   []string  `json:"priority"`   // Filter by priority values
}

type TicketSortOptions struct {
	SortBy    string `json:"sort_by"`    // "created_at", "assignee", "priority"
	SortOrder string `json:"sort_order"` // "asc" or "desc"
}

// Delete request structures
type DeleteTicketsRequest struct {
	TicketIDs []string `json:"ticket_ids"`
	Source    string   `json:"source"` // "asana", "youtrack", "both"
}

type DeleteResult struct {
	TicketID       string `json:"ticket_id"`
	TicketName     string `json:"ticket_name"`
	Status         string `json:"status"` // "success", "failed", "partial"
	AsanaResult    string `json:"asana_result,omitempty"`
	YouTrackResult string `json:"youtrack_result,omitempty"`
	Error          string `json:"error,omitempty"`
}

type DeleteResponse struct {
	Status         string         `json:"status"`
	Source         string         `json:"source"`
	RequestedCount int            `json:"requested_count"`
	SuccessCount   int            `json:"success_count"`
	FailureCount   int            `json:"failure_count"`
	Results        []DeleteResult `json:"results"`
	Summary        string         `json:"summary"`
}

// Auto-sync control structures
type AutoSyncRequest struct {
	Action   string `json:"action"`   // "start" or "stop"
	Interval int    `json:"interval"` // interval in seconds (optional, defaults to 15)
}

type AutoSyncStatus struct {
	Running      bool      `json:"running"`
	Interval     int       `json:"interval"`
	LastSync     time.Time `json:"last_sync"`
	NextSync     time.Time `json:"next_sync"`
	SyncCount    int       `json:"sync_count"`
	LastSyncInfo string    `json:"last_sync_info"`
}

// Auto-create control structures
type AutoCreateRequest struct {
	Action   string `json:"action"`   // "start" or "stop"
	Interval int    `json:"interval"` // interval in seconds (optional, defaults to 15)
}

type AutoCreateStatus struct {
	Running        bool      `json:"running"`
	Interval       int       `json:"interval"`
	LastCreate     time.Time `json:"last_create"`
	NextCreate     time.Time `json:"next_create"`
	CreateCount    int       `json:"create_count"`
	LastCreateInfo string    `json:"last_create_info"`
}

// Ticket details request
type TicketsRequest struct {
	Type   string `json:"type"`   // "matched", "mismatched", "missing", "ignored", etc.
	Column string `json:"column"` // column filter
	Filter TicketFilter      `json:"filter"` // filtering options
	Sort   TicketSortOptions `json:"sort"`   // sorting options
}

// Tag mapping configuration
type TagMapping struct {
	AsanaTag          string `json:"asana_tag"`
	YouTrackSubsystem string `json:"youtrack_subsystem"`
}

// Column definitions
var SyncableColumns = []string{"backlog", "in progress", "dev", "stage", "blocked", "ready for stage"}
var DisplayOnlyColumns = []string{"findings"}
var AllColumns = append(SyncableColumns, DisplayOnlyColumns...)

// Default tag-to-subsystem mapping
var DefaultTagMapping = map[string]string{
	"Mobile":      "mobile",
	"Web":         "web",
	"API":         "backend",
	"Frontend":    "frontend",
	"Backend":     "backend",
	"iOS":         "mobile",
	"Android":     "mobile",
	"Desktop":     "desktop",
	"Database":    "backend",
	"UI/UX":       "frontend",
	"DevOps":      "infrastructure",
	"QA":          "testing",
	"Testing":     "testing",
	"Security":    "security",
	"Performance": "performance",
}

// Helper functions for columns
func IsSyncableColumn(sectionName string) bool {
	for _, col := range SyncableColumns {
		if col == sectionName {
			return true
		}
	}
	return false
}

func IsDisplayOnlyColumn(sectionName string) bool {
	for _, col := range DisplayOnlyColumns {
		if col == sectionName {
			return true
		}
	}
	return false
}

func IsActiveYouTrackStatus(status string) bool {
	activeStatuses := []string{"Backlog", "In Progress", "DEV", "STAGE", "Blocked"}
	for _, activeStatus := range activeStatuses {
		if status == activeStatus {
			return true
		}
	}
	return false
}

// Reverse Sync (YouTrack -> Asana) data structures
type ReverseTicketAnalysis struct {
	Matched      []ReverseMatchedTicket `json:"matched"`
	MissingAsana []YouTrackIssue        `json:"missing_asana"`
}

type ReverseMatchedTicket struct {
	YouTrackIssue YouTrackIssue `json:"youtrack_issue"`
	AsanaTaskID   string        `json:"asana_task_id"`
}

type ReverseSyncResult struct {
	TotalTickets    int                        `json:"total_tickets"`
	SuccessCount    int                        `json:"success_count"`
	FailedCount     int                        `json:"failed_count"`
	FailedTickets   []FailedTicket             `json:"failed_tickets"`
	CreatedMappings []*database.TicketMapping `json:"created_mappings"`
}

type FailedTicket struct {
	IssueID string `json:"issue_id"`
	Title   string `json:"title"`
	Error   string `json:"error"`
}

type ReverseAnalysisRequest struct{
	CreatorFilter string `json:"creator_filter"` // "All" or specific user name
}

type ReverseCreateRequest struct {
	SelectedIssueIDs []string `json:"selected_issue_ids"` // Empty array means create all
}