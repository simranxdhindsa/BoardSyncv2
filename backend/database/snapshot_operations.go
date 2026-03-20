package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// ─── Rollback Snapshot Operations ────────────────────────────────────────────

func (db *DB) CreateRollbackSnapshot(snapshot *RollbackSnapshot) (*RollbackSnapshot, error) {
	ctx := context.Background()

	snapshot.CreatedAt = time.Now()
	snapshot.ExpiresAt = time.Now().Add(24 * time.Hour)

	dataJSON, err := json.Marshal(snapshot.SnapshotData)
	if err != nil {
		return nil, err
	}

	err = db.pool.QueryRow(ctx,
		`INSERT INTO rollback_snapshots (operation_id, user_id, snapshot_data, created_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		snapshot.OperationID, snapshot.UserID, dataJSON, snapshot.CreatedAt, snapshot.ExpiresAt,
	).Scan(&snapshot.ID)
	if err != nil {
		return nil, err
	}

	// Enforce 15-snapshot limit per user (delete oldest beyond limit)
	db.enforceSnapshotLimit(ctx, snapshot.UserID, 15)

	log.Printf("DB: Created rollback snapshot ID %d for operation %d\n", snapshot.ID, snapshot.OperationID)
	return snapshot, nil
}

func (db *DB) enforceSnapshotLimit(ctx context.Context, userID int, maxSnapshots int) {
	_, err := db.pool.Exec(ctx,
		`DELETE FROM rollback_snapshots
		 WHERE id IN (
		   SELECT id FROM rollback_snapshots
		   WHERE user_id = $1
		   ORDER BY created_at DESC
		   OFFSET $2
		 )`,
		userID, maxSnapshots,
	)
	if err != nil {
		log.Printf("DB: Warning: failed to enforce snapshot limit: %v\n", err)
	}
}

func (db *DB) GetSnapshotByOperationID(operationID int) (*RollbackSnapshot, error) {
	ctx := context.Background()
	s := &RollbackSnapshot{}
	var dataJSON []byte
	err := db.pool.QueryRow(ctx,
		`SELECT id, operation_id, user_id, snapshot_data, created_at, expires_at
		 FROM rollback_snapshots WHERE operation_id=$1`,
		operationID,
	).Scan(&s.ID, &s.OperationID, &s.UserID, &dataJSON, &s.CreatedAt, &s.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("snapshot not found for operation %d", operationID)
	}
	json.Unmarshal(dataJSON, &s.SnapshotData)
	return s, nil
}

func (db *DB) GetSnapshotByID(snapshotID int) (*RollbackSnapshot, error) {
	ctx := context.Background()
	s := &RollbackSnapshot{}
	var dataJSON []byte
	err := db.pool.QueryRow(ctx,
		`SELECT id, operation_id, user_id, snapshot_data, created_at, expires_at
		 FROM rollback_snapshots WHERE id=$1`,
		snapshotID,
	).Scan(&s.ID, &s.OperationID, &s.UserID, &dataJSON, &s.CreatedAt, &s.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("snapshot not found")
	}
	json.Unmarshal(dataJSON, &s.SnapshotData)
	return s, nil
}

func (db *DB) GetUserSnapshots(userID int, limit int) ([]*RollbackSnapshot, error) {
	ctx := context.Background()
	rows, err := db.pool.Query(ctx,
		`SELECT id, operation_id, user_id, snapshot_data, created_at, expires_at
		 FROM rollback_snapshots WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []*RollbackSnapshot
	for rows.Next() {
		s := &RollbackSnapshot{}
		var dataJSON []byte
		if err := rows.Scan(&s.ID, &s.OperationID, &s.UserID, &dataJSON, &s.CreatedAt, &s.ExpiresAt); err != nil {
			continue
		}
		json.Unmarshal(dataJSON, &s.SnapshotData)
		snapshots = append(snapshots, s)
	}
	return snapshots, nil
}

func (db *DB) UpdateRollbackSnapshot(snapshot *RollbackSnapshot) error {
	ctx := context.Background()
	dataJSON, err := json.Marshal(snapshot.SnapshotData)
	if err != nil {
		return err
	}
	result, err := db.pool.Exec(ctx,
		`UPDATE rollback_snapshots SET snapshot_data=$1, expires_at=$2 WHERE id=$3`,
		dataJSON, snapshot.ExpiresAt, snapshot.ID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("snapshot not found")
	}
	log.Printf("DB: Updated rollback snapshot ID %d\n", snapshot.ID)
	return nil
}

func (db *DB) DeleteSnapshot(snapshotID int) error {
	ctx := context.Background()
	result, err := db.pool.Exec(ctx, `DELETE FROM rollback_snapshots WHERE id=$1`, snapshotID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("snapshot not found")
	}
	log.Printf("DB: Deleted rollback snapshot ID %d\n", snapshotID)
	return nil
}

func (db *DB) CleanupExpiredSnapshots() error {
	ctx := context.Background()
	result, err := db.pool.Exec(ctx, `DELETE FROM rollback_snapshots WHERE expires_at < NOW()`)
	if err != nil {
		return err
	}
	if result.RowsAffected() > 0 {
		log.Printf("DB: Cleaned up %d expired snapshots\n", result.RowsAffected())
	}
	return nil
}

// ─── Audit Log Operations ─────────────────────────────────────────────────────

func (db *DB) CreateAuditLogEntry(entry *AuditLogEntry) (*AuditLogEntry, error) {
	ctx := context.Background()
	entry.Timestamp = time.Now()
	err := db.pool.QueryRow(ctx,
		`INSERT INTO audit_logs (operation_id, ticket_id, platform, action_type, user_email, old_value, new_value, field_name, timestamp)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 RETURNING id`,
		entry.OperationID, entry.TicketID, entry.Platform, entry.ActionType,
		entry.UserEmail, entry.OldValue, entry.NewValue, entry.FieldName, entry.Timestamp,
	).Scan(&entry.ID)
	if err != nil {
		return nil, err
	}
	log.Printf("DB: Created audit log entry ID %d for ticket %s (action: %s)\n", entry.ID, entry.TicketID, entry.ActionType)
	return entry, nil
}

func (db *DB) GetAuditLogsByOperationID(operationID int) ([]*AuditLogEntry, error) {
	ctx := context.Background()
	rows, err := db.pool.Query(ctx,
		`SELECT id, operation_id, ticket_id, platform, action_type, user_email, old_value, new_value, field_name, timestamp
		 FROM audit_logs WHERE operation_id=$1 ORDER BY timestamp ASC`,
		operationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAuditLogs(rows)
}

func (db *DB) GetAuditLogsByTicketID(ticketID string) ([]*AuditLogEntry, error) {
	ctx := context.Background()
	rows, err := db.pool.Query(ctx,
		`SELECT id, operation_id, ticket_id, platform, action_type, user_email, old_value, new_value, field_name, timestamp
		 FROM audit_logs WHERE ticket_id=$1 ORDER BY timestamp DESC`,
		ticketID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAuditLogs(rows)
}

func (db *DB) GetAuditLogsWithFilter(filter AuditLogFilter) ([]*AuditLogEntry, error) {
	ctx := context.Background()

	query := `SELECT id, operation_id, ticket_id, platform, action_type, user_email, old_value, new_value, field_name, timestamp
	          FROM audit_logs WHERE 1=1`
	args := []interface{}{}
	n := 1

	if filter.UserEmail != "" {
		query += fmt.Sprintf(` AND user_email=$%d`, n)
		args = append(args, filter.UserEmail)
		n++
	}
	if filter.TicketID != "" {
		query += fmt.Sprintf(` AND ticket_id=$%d`, n)
		args = append(args, filter.TicketID)
		n++
	}
	if filter.Platform != "" {
		query += fmt.Sprintf(` AND platform=$%d`, n)
		args = append(args, filter.Platform)
		n++
	}
	if filter.ActionType != "" {
		query += fmt.Sprintf(` AND action_type=$%d`, n)
		args = append(args, filter.ActionType)
		n++
	}
	if !filter.StartDate.IsZero() {
		query += fmt.Sprintf(` AND timestamp>=$%d`, n)
		args = append(args, filter.StartDate)
		n++
	}
	if !filter.EndDate.IsZero() {
		query += fmt.Sprintf(` AND timestamp<=$%d`, n)
		args = append(args, filter.EndDate)
		n++
	}
	query += ` ORDER BY timestamp DESC`
	if filter.Limit > 0 {
		query += fmt.Sprintf(` LIMIT $%d`, n)
		args = append(args, filter.Limit)
	}

	rows, err := db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAuditLogs(rows)
}

func (db *DB) GetRecentAuditLogs(limit int) ([]*AuditLogEntry, error) {
	ctx := context.Background()
	rows, err := db.pool.Query(ctx,
		`SELECT id, operation_id, ticket_id, platform, action_type, user_email, old_value, new_value, field_name, timestamp
		 FROM audit_logs ORDER BY timestamp DESC LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAuditLogs(rows)
}

// scanAuditLogs is a shared helper to scan audit log rows.
func scanAuditLogs(rows interface{ Next() bool; Scan(...interface{}) error }) ([]*AuditLogEntry, error) {
	var logs []*AuditLogEntry
	for rows.Next() {
		e := &AuditLogEntry{}
		if err := rows.Scan(&e.ID, &e.OperationID, &e.TicketID, &e.Platform, &e.ActionType,
			&e.UserEmail, &e.OldValue, &e.NewValue, &e.FieldName, &e.Timestamp); err != nil {
			continue
		}
		logs = append(logs, e)
	}
	return logs, nil
}
