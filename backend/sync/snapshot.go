package sync

import (
	"time"
)

// RollbackSnapshot represents a complete snapshot before sync operations
type RollbackSnapshot struct {
	ID           int          `json:"id"`
	OperationID  int          `json:"operation_id"`
	UserID       int          `json:"user_id"`
	SnapshotData SnapshotData `json:"snapshot_data"`
	CreatedAt    time.Time    `json:"created_at"`
	ExpiresAt    time.Time    `json:"expires_at"`
}

// SnapshotData contains all data needed for rollback
type SnapshotData struct {
	OriginalTickets  []TicketState    `json:"original_tickets"`
	CreatedTickets   []CreatedTicket  `json:"created_tickets"`
	UpdatedMappings  []MappingChange  `json:"updated_mappings"`
	IgnoreChanges    []IgnoreChange   `json:"ignore_changes"`
	ColumnMappings   interface{}      `json:"column_mappings"` // Settings at sync time
}

// TicketState represents the state of a ticket before changes
type TicketState struct {
	Platform       string                 `json:"platform"`        // "asana" or "youtrack"
	TicketID       string                 `json:"ticket_id"`
	OriginalStatus string                 `json:"original_status"`
	NewStatus      string                 `json:"new_status,omitempty"`
	OriginalData   map[string]interface{} `json:"original_data"` // Full ticket snapshot
}

// CreatedTicket represents a ticket created during sync
type CreatedTicket struct {
	Platform  string `json:"platform"`             // "asana" or "youtrack"
	TicketID  string `json:"ticket_id"`
	MappingID int    `json:"mapping_id,omitempty"` // Associated mapping if created
}

// MappingChange represents changes to ticket mappings
type MappingChange struct {
	MappingID  int         `json:"mapping_id"`
	Action     string      `json:"action"` // "created", "updated", "deleted"
	OldMapping interface{} `json:"old_mapping,omitempty"`
	NewMapping interface{} `json:"new_mapping,omitempty"`
}

// IgnoreChange represents changes to ignore status
type IgnoreChange struct {
	TicketID      string `json:"ticket_id"`
	OldIgnoreType string `json:"old_ignore_type"` // "none", "temporary", "forever"
	NewIgnoreType string `json:"new_ignore_type"`
}

// AuditLogEntry represents a detailed log entry for each ticket change
type AuditLogEntry struct {
	ID          int       `json:"id"`
	OperationID int       `json:"operation_id"` // Links to SyncOperation
	TicketID    string    `json:"ticket_id"`    // "ARD-123" or Asana GID
	Platform    string    `json:"platform"`     // "youtrack" or "asana"
	ActionType  string    `json:"action_type"`  // "created", "updated", "status_changed", "ignored", "deleted"
	UserEmail   string    `json:"user_email"`
	OldValue    string    `json:"old_value,omitempty"`
	NewValue    string    `json:"new_value,omitempty"`
	FieldName   string    `json:"field_name,omitempty"` // e.g., "status", "assignee", "description"
	Timestamp   time.Time `json:"timestamp"`
}

