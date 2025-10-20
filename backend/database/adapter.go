package database

import (
	"fmt"
	"log"
	"os"
)

// DBAdapter provides a unified interface for both file-based and PostgreSQL databases
// This makes it easy to switch between the two without changing code everywhere
type DBAdapter struct {
	isPostgres bool
	fileDB     *DB
	postgresDB *PostgresDB
}

var globalAdapter *DBAdapter

// InitializeDatabase initializes either PostgreSQL or file-based database
// based on the DATABASE_URL environment variable
func InitializeDatabase(dbPath string) (*DBAdapter, error) {
	adapter := &DBAdapter{}

	// Check if DATABASE_URL is set (use PostgreSQL)
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		log.Println("🐘 Using PostgreSQL database")
		pgDB, err := InitPostgres()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize PostgreSQL: %w", err)
		}
		adapter.isPostgres = true
		adapter.postgresDB = pgDB
	} else {
		log.Println("📁 Using file-based database (local development)")
		fileDB, err := InitDB(dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize file database: %w", err)
		}
		adapter.isPostgres = false
		adapter.fileDB = fileDB
	}

	globalAdapter = adapter
	return adapter, nil
}

// GetAdapter returns the global database adapter
func GetAdapter() *DBAdapter {
	return globalAdapter
}

// IsPostgres returns true if using PostgreSQL
func (a *DBAdapter) IsPostgres() bool {
	return a.isPostgres
}

// GetFileDB returns the file-based database (nil if using PostgreSQL)
func (a *DBAdapter) GetFileDB() *DB {
	return a.fileDB
}

// GetPostgresDB returns the PostgreSQL database (nil if using file-based)
func (a *DBAdapter) GetPostgresDB() *PostgresDB {
	return a.postgresDB
}

// Close closes the active database connection
func (a *DBAdapter) Close() error {
	if a.isPostgres {
		return a.postgresDB.Close()
	}
	return nil // File DB doesn't need explicit closing
}

// =============================================================================
// UNIFIED INTERFACE METHODS
// These methods work with both database types
// =============================================================================

// CreateUser creates a user in whichever database is active
func (a *DBAdapter) CreateUser(username, email, passwordHash string) (*User, error) {
	if a.isPostgres {
		return a.postgresDB.CreateUser(username, email, passwordHash)
	}
	return a.fileDB.CreateUser(username, email, passwordHash)
}

// GetUserByEmail retrieves a user by email
func (a *DBAdapter) GetUserByEmail(email string) (*User, error) {
	if a.isPostgres {
		return a.postgresDB.GetUserByEmail(email)
	}
	return a.fileDB.GetUserByEmail(email)
}

// GetUserByID retrieves a user by ID
func (a *DBAdapter) GetUserByID(id int) (*User, error) {
	if a.isPostgres {
		return a.postgresDB.GetUserByID(id)
	}
	return a.fileDB.GetUserByID(id)
}

// UpdateUserPassword updates a user's password
func (a *DBAdapter) UpdateUserPassword(userID int, passwordHash string) error {
	if a.isPostgres {
		return a.postgresDB.UpdateUserPassword(userID, passwordHash)
	}
	return a.fileDB.UpdateUserPassword(userID, passwordHash)
}

// DeleteUser deletes a user
func (a *DBAdapter) DeleteUser(userID int) error {
	if a.isPostgres {
		return a.postgresDB.DeleteUser(userID)
	}
	return a.fileDB.DeleteUser(userID)
}

// GetUserSettings retrieves user settings
func (a *DBAdapter) GetUserSettings(userID int) (*UserSettings, error) {
	if a.isPostgres {
		return a.postgresDB.GetUserSettings(userID)
	}
	return a.fileDB.GetUserSettings(userID)
}

// UpdateUserSettings updates user settings
func (a *DBAdapter) UpdateUserSettings(settings *UserSettings) error {
	if a.isPostgres {
		return a.postgresDB.UpdateUserSettings(settings)
	}
	return a.fileDB.UpdateUserSettings(settings)
}

// CreateTicketMapping creates a ticket mapping
func (a *DBAdapter) CreateTicketMapping(userID int, asanaProjectID, asanaTaskID, youtrackProjectID, youtrackIssueID string) (*TicketMapping, error) {
	if a.isPostgres {
		return a.postgresDB.CreateTicketMapping(userID, asanaProjectID, asanaTaskID, youtrackProjectID, youtrackIssueID)
	}
	return a.fileDB.CreateTicketMapping(userID, asanaProjectID, asanaTaskID, youtrackProjectID, youtrackIssueID)
}

// GetAllTicketMappings retrieves all ticket mappings for a user
func (a *DBAdapter) GetAllTicketMappings(userID int) ([]*TicketMapping, error) {
	if a.isPostgres {
		return a.postgresDB.GetAllTicketMappings(userID)
	}
	return a.fileDB.GetAllTicketMappings(userID)
}

// FindMappingByAsanaID finds a mapping by Asana task ID
func (a *DBAdapter) FindMappingByAsanaID(userID int, asanaTaskID string) (*TicketMapping, error) {
	if a.isPostgres {
		return a.postgresDB.FindMappingByAsanaID(userID, asanaTaskID)
	}
	return a.fileDB.FindMappingByAsanaID(userID, asanaTaskID)
}

// FindMappingByYouTrackID finds a mapping by YouTrack issue ID
func (a *DBAdapter) FindMappingByYouTrackID(userID int, youtrackIssueID string) (*TicketMapping, error) {
	if a.isPostgres {
		return a.postgresDB.FindMappingByYouTrackID(userID, youtrackIssueID)
	}
	return a.fileDB.FindMappingByYouTrackID(userID, youtrackIssueID)
}

// DeleteTicketMapping deletes a ticket mapping
func (a *DBAdapter) DeleteTicketMapping(id int) error {
	if a.isPostgres {
		return a.postgresDB.DeleteTicketMapping(id)
	}
	return a.fileDB.DeleteTicketMapping(id)
}

// AddIgnoredTicket adds a ticket to the ignore list
func (a *DBAdapter) AddIgnoredTicket(userID int, asanaProjectID, ticketID, ignoreType string) error {
	if a.isPostgres {
		return a.postgresDB.AddIgnoredTicket(userID, asanaProjectID, ticketID, ignoreType)
	}
	return a.fileDB.AddIgnoredTicket(userID, asanaProjectID, ticketID, ignoreType)
}

// RemoveIgnoredTicket removes a ticket from the ignore list
func (a *DBAdapter) RemoveIgnoredTicket(userID int, asanaProjectID, ticketID string) error {
	if a.isPostgres {
		return a.postgresDB.RemoveIgnoredTicket(userID, asanaProjectID, ticketID)
	}
	return a.fileDB.RemoveIgnoredTicket(userID, asanaProjectID, ticketID)
}

// GetIgnoredTickets retrieves all ignored tickets for a project
func (a *DBAdapter) GetIgnoredTickets(userID int, asanaProjectID string) ([]string, error) {
	if a.isPostgres {
		return a.postgresDB.GetIgnoredTickets(userID, asanaProjectID)
	}
	return a.fileDB.GetIgnoredTickets(userID, asanaProjectID)
}

// IsTicketIgnored checks if a ticket is ignored
func (a *DBAdapter) IsTicketIgnored(userID int, asanaProjectID, ticketID string) (bool, error) {
	if a.isPostgres {
		return a.postgresDB.IsTicketIgnored(userID, asanaProjectID, ticketID)
	}
	return a.fileDB.IsTicketIgnored(userID, asanaProjectID, ticketID)
}

// ClearTemporaryIgnores clears all temporary ignores for a project
func (a *DBAdapter) ClearTemporaryIgnores(userID int, asanaProjectID string) error {
	if a.isPostgres {
		return a.postgresDB.ClearTemporaryIgnores(userID, asanaProjectID)
	}
	return a.fileDB.ClearTemporaryIgnores(userID, asanaProjectID)
}

// CreateSyncOperation creates a sync operation
func (a *DBAdapter) CreateSyncOperation(userID int, operationType string, operationData map[string]interface{}) (*SyncOperation, error) {
	if a.isPostgres {
		return a.postgresDB.CreateSyncOperation(userID, operationType, operationData)
	}
	return a.fileDB.CreateSyncOperation(userID, operationType, operationData)
}

// GetSyncOperation retrieves a sync operation by ID
func (a *DBAdapter) GetSyncOperation(id int) (*SyncOperation, error) {
	if a.isPostgres {
		return a.postgresDB.GetSyncOperation(id)
	}
	return a.fileDB.GetSyncOperation(id)
}

// GetUserSyncOperations retrieves sync operations for a user
func (a *DBAdapter) GetUserSyncOperations(userID int, limit int) ([]*SyncOperation, error) {
	if a.isPostgres {
		return a.postgresDB.GetUserSyncOperations(userID, limit)
	}
	return a.fileDB.GetUserSyncOperations(userID, limit)
}

// UpdateSyncOperationStatus updates the status of a sync operation
func (a *DBAdapter) UpdateSyncOperationStatus(id int, status string, errorMsg *string) error {
	if a.isPostgres {
		return a.postgresDB.UpdateSyncOperationStatus(id, status, errorMsg)
	}
	return a.fileDB.UpdateSyncOperationStatus(id, status, errorMsg)
}

// CreateRollbackSnapshot creates a rollback snapshot
func (a *DBAdapter) CreateRollbackSnapshot(operationID, userID int, snapshotData SnapshotData) (*RollbackSnapshot, error) {
	if a.isPostgres {
		return a.postgresDB.CreateRollbackSnapshot(operationID, userID, snapshotData)
	}
	return a.fileDB.CreateRollbackSnapshot(operationID, userID, snapshotData)
}

// GetRollbackSnapshot retrieves a rollback snapshot
func (a *DBAdapter) GetRollbackSnapshot(operationID int) (*RollbackSnapshot, error) {
	if a.isPostgres {
		return a.postgresDB.GetRollbackSnapshot(operationID)
	}
	return a.fileDB.GetRollbackSnapshot(operationID)
}

// DeleteRollbackSnapshot deletes a rollback snapshot
func (a *DBAdapter) DeleteRollbackSnapshot(operationID int) error {
	if a.isPostgres {
		return a.postgresDB.DeleteRollbackSnapshot(operationID)
	}
	return a.fileDB.DeleteRollbackSnapshot(operationID)
}

// CleanupExpiredSnapshots removes expired snapshots
func (a *DBAdapter) CleanupExpiredSnapshots() (int, error) {
	if a.isPostgres {
		return a.postgresDB.CleanupExpiredSnapshots()
	}
	return a.fileDB.CleanupExpiredSnapshots()
}

// CreateAuditLog creates an audit log entry
func (a *DBAdapter) CreateAuditLog(entry *AuditLogEntry) error {
	if a.isPostgres {
		return a.postgresDB.CreateAuditLog(entry)
	}
	return a.fileDB.CreateAuditLog(entry)
}

// GetAuditLogs retrieves audit logs with filtering
func (a *DBAdapter) GetAuditLogs(filter *AuditLogFilter) ([]*AuditLogEntry, error) {
	if a.isPostgres {
		return a.postgresDB.GetAuditLogs(filter)
	}
	return a.fileDB.GetAuditLogs(filter)
}

// GetOperationAuditLogs retrieves audit logs for a specific operation
func (a *DBAdapter) GetOperationAuditLogs(operationID int) ([]*AuditLogEntry, error) {
	if a.isPostgres {
		return a.postgresDB.GetOperationAuditLogs(operationID)
	}
	return a.fileDB.GetOperationAuditLogs(operationID)
}

// GetTicketAuditHistory retrieves audit history for a specific ticket
func (a *DBAdapter) GetTicketAuditHistory(ticketID string) ([]*AuditLogEntry, error) {
	if a.isPostgres {
		return a.postgresDB.GetTicketAuditHistory(ticketID)
	}
	return a.fileDB.GetTicketAuditHistory(ticketID)
}
