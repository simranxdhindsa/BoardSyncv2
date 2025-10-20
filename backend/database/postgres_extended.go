package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// =============================================================================
// SYNC OPERATIONS
// =============================================================================

func (db *PostgresDB) CreateSyncOperation(userID int, operationType string, operationData map[string]interface{}) (*SyncOperation, error) {
	query := `
		INSERT INTO sync_operations (user_id, operation_type, operation_data, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, operation_type, operation_data, status, error_message, created_at, completed_at
	`

	op := &SyncOperation{
		UserID:        userID,
		OperationType: operationType,
		OperationData: operationData,
		Status:        "pending",
		CreatedAt:     time.Now(),
	}

	err := db.conn.QueryRow(query, userID, operationType, OperationData(operationData), "pending", op.CreatedAt).Scan(
		&op.ID,
		&op.UserID,
		&op.OperationType,
		&op.OperationData,
		&op.Status,
		&op.ErrorMessage,
		&op.CreatedAt,
		&op.CompletedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create sync operation: %w", err)
	}

	log.Printf("DB: Created sync operation ID: %d\n", op.ID)
	return op, nil
}

func (db *PostgresDB) GetSyncOperation(id int) (*SyncOperation, error) {
	query := `
		SELECT id, user_id, operation_type, operation_data, status, error_message, created_at, completed_at
		FROM sync_operations
		WHERE id = $1
	`

	op := &SyncOperation{}
	err := db.conn.QueryRow(query, id).Scan(
		&op.ID,
		&op.UserID,
		&op.OperationType,
		&op.OperationData,
		&op.Status,
		&op.ErrorMessage,
		&op.CreatedAt,
		&op.CompletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("operation not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get operation: %w", err)
	}

	return op, nil
}

func (db *PostgresDB) GetUserSyncOperations(userID int, limit int) ([]*SyncOperation, error) {
	query := `
		SELECT id, user_id, operation_type, operation_data, status, error_message, created_at, completed_at
		FROM sync_operations
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := db.conn.Query(query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get operations: %w", err)
	}
	defer rows.Close()

	operations := []*SyncOperation{}
	for rows.Next() {
		op := &SyncOperation{}
		err := rows.Scan(
			&op.ID,
			&op.UserID,
			&op.OperationType,
			&op.OperationData,
			&op.Status,
			&op.ErrorMessage,
			&op.CreatedAt,
			&op.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan operation: %w", err)
		}
		operations = append(operations, op)
	}

	return operations, nil
}

func (db *PostgresDB) UpdateSyncOperationStatus(id int, status string, errorMsg *string) error {
	query := `
		UPDATE sync_operations
		SET status = $1, error_message = $2, completed_at = $3
		WHERE id = $4
	`

	var completedAt *time.Time
	if status == "completed" || status == "failed" || status == "rolled_back" {
		now := time.Now()
		completedAt = &now
	}

	_, err := db.conn.Exec(query, status, errorMsg, completedAt, id)
	if err != nil {
		return fmt.Errorf("failed to update operation status: %w", err)
	}

	log.Printf("DB: Updated operation %d status to: %s\n", id, status)
	return nil
}

// =============================================================================
// ROLLBACK SNAPSHOTS
// =============================================================================

func (db *PostgresDB) CreateRollbackSnapshot(operationID, userID int, snapshotData SnapshotData) (*RollbackSnapshot, error) {
	query := `
		INSERT INTO rollback_snapshots (operation_id, user_id, snapshot_data, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, operation_id, user_id, snapshot_data, created_at, expires_at
	`

	now := time.Now()
	expiresAt := now.Add(30 * 24 * time.Hour) // 30 days

	snapshot := &RollbackSnapshot{
		OperationID:  operationID,
		UserID:       userID,
		SnapshotData: snapshotData,
		CreatedAt:    now,
		ExpiresAt:    expiresAt,
	}

	// Convert SnapshotData to JSON
	snapshotJSON, err := json.Marshal(snapshotData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snapshot data: %w", err)
	}

	err = db.conn.QueryRow(query, operationID, userID, snapshotJSON, now, expiresAt).Scan(
		&snapshot.ID,
		&snapshot.OperationID,
		&snapshot.UserID,
		&snapshotJSON,
		&snapshot.CreatedAt,
		&snapshot.ExpiresAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}

	log.Printf("DB: Created snapshot ID: %d for operation %d\n", snapshot.ID, operationID)
	return snapshot, nil
}

func (db *PostgresDB) GetRollbackSnapshot(operationID int) (*RollbackSnapshot, error) {
	query := `
		SELECT id, operation_id, user_id, snapshot_data, created_at, expires_at
		FROM rollback_snapshots
		WHERE operation_id = $1
	`

	snapshot := &RollbackSnapshot{}
	var snapshotJSON []byte

	err := db.conn.QueryRow(query, operationID).Scan(
		&snapshot.ID,
		&snapshot.OperationID,
		&snapshot.UserID,
		&snapshotJSON,
		&snapshot.CreatedAt,
		&snapshot.ExpiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("snapshot not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	// Unmarshal snapshot data
	if err := json.Unmarshal(snapshotJSON, &snapshot.SnapshotData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot data: %w", err)
	}

	return snapshot, nil
}

func (db *PostgresDB) DeleteRollbackSnapshot(operationID int) error {
	query := `DELETE FROM rollback_snapshots WHERE operation_id = $1`

	_, err := db.conn.Exec(query, operationID)
	if err != nil {
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}

	log.Printf("DB: Deleted snapshot for operation %d\n", operationID)
	return nil
}

func (db *PostgresDB) CleanupExpiredSnapshots() (int, error) {
	query := `DELETE FROM rollback_snapshots WHERE expires_at < $1`

	result, err := db.conn.Exec(query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired snapshots: %w", err)
	}

	count, _ := result.RowsAffected()
	if count > 0 {
		log.Printf("DB: Cleaned up %d expired snapshots\n", count)
	}
	return int(count), nil
}

// =============================================================================
// AUDIT LOGS
// =============================================================================

func (db *PostgresDB) CreateAuditLog(entry *AuditLogEntry) error {
	query := `
		INSERT INTO audit_logs (operation_id, ticket_id, platform, action_type, user_email, old_value, new_value, field_name, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	err := db.conn.QueryRow(
		query,
		entry.OperationID,
		entry.TicketID,
		entry.Platform,
		entry.ActionType,
		entry.UserEmail,
		entry.OldValue,
		entry.NewValue,
		entry.FieldName,
		entry.Timestamp,
	).Scan(&entry.ID)

	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

func (db *PostgresDB) GetAuditLogs(filter *AuditLogFilter) ([]*AuditLogEntry, error) {
	query := `
		SELECT id, operation_id, ticket_id, platform, action_type, user_email, old_value, new_value, field_name, timestamp
		FROM audit_logs
		WHERE 1=1
	`
	args := []interface{}{}
	argNum := 1

	if filter.UserEmail != "" {
		query += fmt.Sprintf(" AND user_email = $%d", argNum)
		args = append(args, filter.UserEmail)
		argNum++
	}

	if filter.TicketID != "" {
		query += fmt.Sprintf(" AND ticket_id = $%d", argNum)
		args = append(args, filter.TicketID)
		argNum++
	}

	if filter.Platform != "" {
		query += fmt.Sprintf(" AND platform = $%d", argNum)
		args = append(args, filter.Platform)
		argNum++
	}

	if filter.ActionType != "" {
		query += fmt.Sprintf(" AND action_type = $%d", argNum)
		args = append(args, filter.ActionType)
		argNum++
	}

	if !filter.StartDate.IsZero() {
		query += fmt.Sprintf(" AND timestamp >= $%d", argNum)
		args = append(args, filter.StartDate)
		argNum++
	}

	if !filter.EndDate.IsZero() {
		query += fmt.Sprintf(" AND timestamp <= $%d", argNum)
		args = append(args, filter.EndDate)
		argNum++
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, filter.Limit)
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs: %w", err)
	}
	defer rows.Close()

	logs := []*AuditLogEntry{}
	for rows.Next() {
		entry := &AuditLogEntry{}
		err := rows.Scan(
			&entry.ID,
			&entry.OperationID,
			&entry.TicketID,
			&entry.Platform,
			&entry.ActionType,
			&entry.UserEmail,
			&entry.OldValue,
			&entry.NewValue,
			&entry.FieldName,
			&entry.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, entry)
	}

	return logs, nil
}

func (db *PostgresDB) GetOperationAuditLogs(operationID int) ([]*AuditLogEntry, error) {
	query := `
		SELECT id, operation_id, ticket_id, platform, action_type, user_email, old_value, new_value, field_name, timestamp
		FROM audit_logs
		WHERE operation_id = $1
		ORDER BY timestamp ASC
	`

	rows, err := db.conn.Query(query, operationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get operation audit logs: %w", err)
	}
	defer rows.Close()

	logs := []*AuditLogEntry{}
	for rows.Next() {
		entry := &AuditLogEntry{}
		err := rows.Scan(
			&entry.ID,
			&entry.OperationID,
			&entry.TicketID,
			&entry.Platform,
			&entry.ActionType,
			&entry.UserEmail,
			&entry.OldValue,
			&entry.NewValue,
			&entry.FieldName,
			&entry.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, entry)
	}

	return logs, nil
}

func (db *PostgresDB) GetTicketAuditHistory(ticketID string) ([]*AuditLogEntry, error) {
	query := `
		SELECT id, operation_id, ticket_id, platform, action_type, user_email, old_value, new_value, field_name, timestamp
		FROM audit_logs
		WHERE ticket_id = $1
		ORDER BY timestamp DESC
	`

	rows, err := db.conn.Query(query, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket history: %w", err)
	}
	defer rows.Close()

	logs := []*AuditLogEntry{}
	for rows.Next() {
		entry := &AuditLogEntry{}
		err := rows.Scan(
			&entry.ID,
			&entry.OperationID,
			&entry.TicketID,
			&entry.Platform,
			&entry.ActionType,
			&entry.UserEmail,
			&entry.OldValue,
			&entry.NewValue,
			&entry.FieldName,
			&entry.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, entry)
	}

	return logs, nil
}
