package sync

import (
	"asana-youtrack-sync/database"
	"fmt"
	"log"
)

// AuditService handles audit logging operations
type AuditService struct {
	db *database.DB
}

// NewAuditService creates a new audit service
func NewAuditService(db *database.DB) *AuditService {
	return &AuditService{
		db: db,
	}
}

// Action types for audit log
const (
	ActionCreated       = "created"
	ActionUpdated       = "updated"
	ActionStatusChanged = "status_changed"
	ActionIgnored       = "ignored"
	ActionDeleted       = "deleted"
	ActionRolledBack    = "rolled_back"
	ActionMappingAdded  = "mapping_added"
)

// LogTicketCreated logs a ticket creation event
func (as *AuditService) LogTicketCreated(operationID int, userEmail, ticketID, platform, initialStatus string) error {
	entry := &database.AuditLogEntry{
		OperationID: operationID,
		TicketID:    ticketID,
		Platform:    platform,
		ActionType:  ActionCreated,
		UserEmail:   userEmail,
		OldValue:    "",
		NewValue:    initialStatus,
		FieldName:   "status",
	}

	_, err := as.db.CreateAuditLogEntry(entry)
	if err != nil {
		return fmt.Errorf("failed to log ticket creation: %w", err)
	}

	log.Printf("AuditService: Logged ticket creation: %s %s by %s\n", platform, ticketID, userEmail)
	return nil
}

// LogStatusChange logs a status change event
func (as *AuditService) LogStatusChange(operationID int, userEmail, ticketID, platform, oldStatus, newStatus string) error {
	entry := &database.AuditLogEntry{
		OperationID: operationID,
		TicketID:    ticketID,
		Platform:    platform,
		ActionType:  ActionStatusChanged,
		UserEmail:   userEmail,
		OldValue:    oldStatus,
		NewValue:    newStatus,
		FieldName:   "status",
	}

	_, err := as.db.CreateAuditLogEntry(entry)
	if err != nil {
		return fmt.Errorf("failed to log status change: %w", err)
	}

	log.Printf("AuditService: Logged status change: %s %s (%s -> %s) by %s\n",
		platform, ticketID, oldStatus, newStatus, userEmail)
	return nil
}

// LogFieldUpdate logs a field update event
func (as *AuditService) LogFieldUpdate(operationID int, userEmail, ticketID, platform, fieldName, oldValue, newValue string) error {
	entry := &database.AuditLogEntry{
		OperationID: operationID,
		TicketID:    ticketID,
		Platform:    platform,
		ActionType:  ActionUpdated,
		UserEmail:   userEmail,
		OldValue:    oldValue,
		NewValue:    newValue,
		FieldName:   fieldName,
	}

	_, err := as.db.CreateAuditLogEntry(entry)
	if err != nil {
		return fmt.Errorf("failed to log field update: %w", err)
	}

	log.Printf("AuditService: Logged field update: %s %s.%s (%s -> %s) by %s\n",
		platform, ticketID, fieldName, oldValue, newValue, userEmail)
	return nil
}

// LogTicketIgnored logs when a ticket is ignored
func (as *AuditService) LogTicketIgnored(operationID int, userEmail, ticketID, platform, ignoreType string) error {
	entry := &database.AuditLogEntry{
		OperationID: operationID,
		TicketID:    ticketID,
		Platform:    platform,
		ActionType:  ActionIgnored,
		UserEmail:   userEmail,
		OldValue:    "",
		NewValue:    ignoreType,
		FieldName:   "ignore_type",
	}

	_, err := as.db.CreateAuditLogEntry(entry)
	if err != nil {
		return fmt.Errorf("failed to log ticket ignored: %w", err)
	}

	log.Printf("AuditService: Logged ticket ignored: %s %s (%s) by %s\n",
		platform, ticketID, ignoreType, userEmail)
	return nil
}

// LogTicketDeleted logs when a ticket is deleted (during rollback)
func (as *AuditService) LogTicketDeleted(operationID int, userEmail, ticketID, platform string) error {
	entry := &database.AuditLogEntry{
		OperationID: operationID,
		TicketID:    ticketID,
		Platform:    platform,
		ActionType:  ActionDeleted,
		UserEmail:   userEmail,
		OldValue:    "",
		NewValue:    "",
		FieldName:   "",
	}

	_, err := as.db.CreateAuditLogEntry(entry)
	if err != nil {
		return fmt.Errorf("failed to log ticket deletion: %w", err)
	}

	log.Printf("AuditService: Logged ticket deletion: %s %s by %s\n", platform, ticketID, userEmail)
	return nil
}

// LogMappingCreated logs when a ticket mapping is created
func (as *AuditService) LogMappingCreated(operationID int, userEmail, asanaTaskID, youtrackIssueID string) error {
	entry := &database.AuditLogEntry{
		OperationID: operationID,
		TicketID:    fmt.Sprintf("%s <-> %s", asanaTaskID, youtrackIssueID),
		Platform:    "mapping",
		ActionType:  ActionMappingAdded,
		UserEmail:   userEmail,
		OldValue:    "",
		NewValue:    fmt.Sprintf("Asana: %s, YouTrack: %s", asanaTaskID, youtrackIssueID),
		FieldName:   "mapping",
	}

	_, err := as.db.CreateAuditLogEntry(entry)
	if err != nil {
		return fmt.Errorf("failed to log mapping creation: %w", err)
	}

	log.Printf("AuditService: Logged mapping creation: %s <-> %s by %s\n",
		asanaTaskID, youtrackIssueID, userEmail)
	return nil
}

// LogRollback logs when an operation is rolled back
func (as *AuditService) LogRollback(operationID int, userEmail string, rollbackDetails string) error {
	entry := &database.AuditLogEntry{
		OperationID: operationID,
		TicketID:    fmt.Sprintf("operation_%d", operationID),
		Platform:    "system",
		ActionType:  ActionRolledBack,
		UserEmail:   userEmail,
		OldValue:    "",
		NewValue:    rollbackDetails,
		FieldName:   "rollback",
	}

	_, err := as.db.CreateAuditLogEntry(entry)
	if err != nil {
		return fmt.Errorf("failed to log rollback: %w", err)
	}

	log.Printf("AuditService: Logged rollback: operation %d by %s\n", operationID, userEmail)
	return nil
}

// GetTicketHistory retrieves all audit logs for a specific ticket
func (as *AuditService) GetTicketHistory(ticketID string) ([]*database.AuditLogEntry, error) {
	logs, err := as.db.GetAuditLogsByTicketID(ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket history: %w", err)
	}

	return logs, nil
}

// GetOperationHistory retrieves all audit logs for a specific operation
func (as *AuditService) GetOperationHistory(operationID int) ([]*database.AuditLogEntry, error) {
	logs, err := as.db.GetAuditLogsByOperationID(operationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get operation history: %w", err)
	}

	return logs, nil
}

// GetFilteredAuditLogs retrieves audit logs with advanced filtering
func (as *AuditService) GetFilteredAuditLogs(filter database.AuditLogFilter) ([]*database.AuditLogEntry, error) {
	logs, err := as.db.GetAuditLogsWithFilter(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get filtered audit logs: %w", err)
	}

	return logs, nil
}

// GetRecentAuditLogs retrieves the most recent audit logs
func (as *AuditService) GetRecentAuditLogs(limit int) ([]*database.AuditLogEntry, error) {
	logs, err := as.db.GetRecentAuditLogs(limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent audit logs: %w", err)
	}

	return logs, nil
}

// ExportAuditLogsToCSV exports audit logs to CSV format
func (as *AuditService) ExportAuditLogsToCSV(filter database.AuditLogFilter) (string, error) {
	logs, err := as.GetFilteredAuditLogs(filter)
	if err != nil {
		return "", fmt.Errorf("failed to get audit logs for export: %w", err)
	}

	// Build CSV content
	csv := "Timestamp,User Email,Ticket ID,Platform,Action Type,Field Name,Old Value,New Value\n"

	for _, log := range logs {
		csv += fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s\n",
			log.Timestamp.Format("2006-01-02 15:04:05"),
			log.UserEmail,
			log.TicketID,
			log.Platform,
			log.ActionType,
			log.FieldName,
			log.OldValue,
			log.NewValue,
		)
	}

	return csv, nil
}
