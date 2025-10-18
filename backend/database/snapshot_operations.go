package database

import (
	"fmt"
	"log"
	"time"
)

// Rollback Snapshot Operations

// CreateRollbackSnapshot creates a new rollback snapshot
func (db *DB) CreateRollbackSnapshot(snapshot *RollbackSnapshot) (*RollbackSnapshot, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	snapshot.ID = db.nextSnapshotID
	snapshot.CreatedAt = time.Now()
	snapshot.ExpiresAt = time.Now().Add(24 * time.Hour) // 24-hour expiration

	db.rollbackSnapshots[snapshot.ID] = snapshot
	db.nextSnapshotID++

	// Enforce maximum 15 snapshots per user
	db.enforceSnapshotLimit(snapshot.UserID, 15)

	if err := db.saveData(); err != nil {
		return nil, err
	}

	log.Printf("DB: Created rollback snapshot ID %d for operation %d\n", snapshot.ID, snapshot.OperationID)
	return snapshot, nil
}

// enforceSnapshotLimit ensures max 15 snapshots per user (FIFO deletion)
func (db *DB) enforceSnapshotLimit(userID int, maxSnapshots int) {
	// Get all snapshots for this user sorted by creation time
	userSnapshots := []*RollbackSnapshot{}
	for _, snapshot := range db.rollbackSnapshots {
		if snapshot.UserID == userID {
			userSnapshots = append(userSnapshots, snapshot)
		}
	}

	// If we're over the limit, delete oldest ones
	if len(userSnapshots) > maxSnapshots {
		// Sort by CreatedAt (oldest first)
		for i := 0; i < len(userSnapshots)-1; i++ {
			for j := i + 1; j < len(userSnapshots); j++ {
				if userSnapshots[i].CreatedAt.After(userSnapshots[j].CreatedAt) {
					userSnapshots[i], userSnapshots[j] = userSnapshots[j], userSnapshots[i]
				}
			}
		}

		// Delete oldest snapshots
		deleteCount := len(userSnapshots) - maxSnapshots
		for i := 0; i < deleteCount; i++ {
			delete(db.rollbackSnapshots, userSnapshots[i].ID)
			log.Printf("DB: Deleted old snapshot ID %d (FIFO limit enforcement)\n", userSnapshots[i].ID)
		}
	}
}

// GetSnapshotByOperationID retrieves a snapshot by operation ID
func (db *DB) GetSnapshotByOperationID(operationID int) (*RollbackSnapshot, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	for _, snapshot := range db.rollbackSnapshots {
		if snapshot.OperationID == operationID {
			return snapshot, nil
		}
	}

	return nil, fmt.Errorf("snapshot not found for operation %d", operationID)
}

// GetSnapshotByID retrieves a snapshot by its ID
func (db *DB) GetSnapshotByID(snapshotID int) (*RollbackSnapshot, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	if snapshot, exists := db.rollbackSnapshots[snapshotID]; exists {
		return snapshot, nil
	}

	return nil, fmt.Errorf("snapshot not found")
}

// GetUserSnapshots retrieves all snapshots for a user (most recent first)
func (db *DB) GetUserSnapshots(userID int, limit int) ([]*RollbackSnapshot, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	snapshots := []*RollbackSnapshot{}
	for _, snapshot := range db.rollbackSnapshots {
		if snapshot.UserID == userID {
			snapshots = append(snapshots, snapshot)
		}
	}

	// Sort by CreatedAt (most recent first)
	for i := 0; i < len(snapshots)-1; i++ {
		for j := i + 1; j < len(snapshots); j++ {
			if snapshots[i].CreatedAt.Before(snapshots[j].CreatedAt) {
				snapshots[i], snapshots[j] = snapshots[j], snapshots[i]
			}
		}
	}

	// Apply limit
	if limit > 0 && len(snapshots) > limit {
		snapshots = snapshots[:limit]
	}

	return snapshots, nil
}

// UpdateRollbackSnapshot updates an existing snapshot
func (db *DB) UpdateRollbackSnapshot(snapshot *RollbackSnapshot) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	if _, exists := db.rollbackSnapshots[snapshot.ID]; !exists {
		return fmt.Errorf("snapshot not found")
	}

	db.rollbackSnapshots[snapshot.ID] = snapshot

	if err := db.saveData(); err != nil {
		return err
	}

	log.Printf("DB: Updated rollback snapshot ID %d\n", snapshot.ID)
	return nil
}

// DeleteSnapshot deletes a snapshot by ID
func (db *DB) DeleteSnapshot(snapshotID int) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	if _, exists := db.rollbackSnapshots[snapshotID]; !exists {
		return fmt.Errorf("snapshot not found")
	}

	delete(db.rollbackSnapshots, snapshotID)

	if err := db.saveData(); err != nil {
		return err
	}

	log.Printf("DB: Deleted rollback snapshot ID %d\n", snapshotID)
	return nil
}

// CleanupExpiredSnapshots removes snapshots that have expired
func (db *DB) CleanupExpiredSnapshots() error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	now := time.Now()
	deletedCount := 0

	for id, snapshot := range db.rollbackSnapshots {
		if now.After(snapshot.ExpiresAt) {
			delete(db.rollbackSnapshots, id)
			deletedCount++
		}
	}

	if deletedCount > 0 {
		log.Printf("DB: Cleaned up %d expired snapshots\n", deletedCount)
		return db.saveData()
	}

	return nil
}

// Audit Log Operations

// CreateAuditLogEntry creates a new audit log entry
func (db *DB) CreateAuditLogEntry(entry *AuditLogEntry) (*AuditLogEntry, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	entry.ID = db.nextAuditLogID
	entry.Timestamp = time.Now()

	db.auditLogs[entry.ID] = entry
	db.nextAuditLogID++

	if err := db.saveData(); err != nil {
		return nil, err
	}

	log.Printf("DB: Created audit log entry ID %d for ticket %s (action: %s)\n",
		entry.ID, entry.TicketID, entry.ActionType)
	return entry, nil
}

// GetAuditLogsByOperationID retrieves all audit logs for a specific operation
func (db *DB) GetAuditLogsByOperationID(operationID int) ([]*AuditLogEntry, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	logs := []*AuditLogEntry{}
	for _, log := range db.auditLogs {
		if log.OperationID == operationID {
			logs = append(logs, log)
		}
	}

	// Sort by timestamp
	for i := 0; i < len(logs)-1; i++ {
		for j := i + 1; j < len(logs); j++ {
			if logs[i].Timestamp.After(logs[j].Timestamp) {
				logs[i], logs[j] = logs[j], logs[i]
			}
		}
	}

	return logs, nil
}

// GetAuditLogsByTicketID retrieves all audit logs for a specific ticket
func (db *DB) GetAuditLogsByTicketID(ticketID string) ([]*AuditLogEntry, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	logs := []*AuditLogEntry{}
	for _, log := range db.auditLogs {
		if log.TicketID == ticketID {
			logs = append(logs, log)
		}
	}

	// Sort by timestamp (most recent first)
	for i := 0; i < len(logs)-1; i++ {
		for j := i + 1; j < len(logs); j++ {
			if logs[i].Timestamp.Before(logs[j].Timestamp) {
				logs[i], logs[j] = logs[j], logs[i]
			}
		}
	}

	return logs, nil
}

// GetAuditLogsWithFilter retrieves audit logs with advanced filtering
func (db *DB) GetAuditLogsWithFilter(filter AuditLogFilter) ([]*AuditLogEntry, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	logs := []*AuditLogEntry{}

	for _, log := range db.auditLogs {
		// Apply filters
		if filter.UserEmail != "" && log.UserEmail != filter.UserEmail {
			continue
		}
		if filter.TicketID != "" && log.TicketID != filter.TicketID {
			continue
		}
		if filter.Platform != "" && log.Platform != filter.Platform {
			continue
		}
		if filter.ActionType != "" && log.ActionType != filter.ActionType {
			continue
		}
		if !filter.StartDate.IsZero() && log.Timestamp.Before(filter.StartDate) {
			continue
		}
		if !filter.EndDate.IsZero() && log.Timestamp.After(filter.EndDate) {
			continue
		}

		logs = append(logs, log)
	}

	// Sort by timestamp (most recent first)
	for i := 0; i < len(logs)-1; i++ {
		for j := i + 1; j < len(logs); j++ {
			if logs[i].Timestamp.Before(logs[j].Timestamp) {
				logs[i], logs[j] = logs[j], logs[i]
			}
		}
	}

	// Apply limit
	if filter.Limit > 0 && len(logs) > filter.Limit {
		logs = logs[:filter.Limit]
	}

	return logs, nil
}

// GetRecentAuditLogs retrieves the most recent audit logs (all users)
func (db *DB) GetRecentAuditLogs(limit int) ([]*AuditLogEntry, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	logs := []*AuditLogEntry{}
	for _, log := range db.auditLogs {
		logs = append(logs, log)
	}

	// Sort by timestamp (most recent first)
	for i := 0; i < len(logs)-1; i++ {
		for j := i + 1; j < len(logs); j++ {
			if logs[i].Timestamp.Before(logs[j].Timestamp) {
				logs[i], logs[j] = logs[j], logs[i]
			}
		}
	}

	// Apply limit
	if limit > 0 && len(logs) > limit {
		logs = logs[:limit]
	}

	return logs, nil
}
