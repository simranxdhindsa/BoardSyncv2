package sync

import (
	"asana-youtrack-sync/database"
	"fmt"
	"log"
	"time"
)

// SnapshotService handles snapshot creation and rollback operations
type SnapshotService struct {
	db *database.DB
}

// NewSnapshotService creates a new snapshot service
func NewSnapshotService(db *database.DB) *SnapshotService {
	return &SnapshotService{
		db: db,
	}
}

// CreatePreSyncSnapshot creates a snapshot before sync operations
func (ss *SnapshotService) CreatePreSyncSnapshot(userID, operationID int, syncType string) (*database.RollbackSnapshot, error) {
	snapshot := &database.RollbackSnapshot{
		OperationID: operationID,
		UserID:      userID,
		SnapshotData: database.SnapshotData{
			OriginalTickets:  []database.TicketState{},
			CreatedTickets:   []database.CreatedTicket{},
			UpdatedMappings:  []database.MappingChange{},
			IgnoreChanges:    []database.IgnoreChange{},
			ColumnMappings:   nil,
		},
	}

	// Capture current column mappings
	settings, err := ss.db.GetUserSettings(userID)
	if err == nil {
		snapshot.SnapshotData.ColumnMappings = settings.ColumnMappings
	}

	// Create the snapshot (this will enforce the 15-snapshot limit)
	createdSnapshot, err := ss.db.CreateRollbackSnapshot(snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}

	log.Printf("SnapshotService: Created pre-sync snapshot ID %d for operation %d\n",
		createdSnapshot.ID, operationID)

	return createdSnapshot, nil
}

// RecordTicketCreation records a ticket that was created during sync
func (ss *SnapshotService) RecordTicketCreation(operationID int, platform, ticketID string, mappingID int) error {
	snapshot, err := ss.db.GetSnapshotByOperationID(operationID)
	if err != nil {
		return fmt.Errorf("snapshot not found for operation %d: %w", operationID, err)
	}

	createdTicket := database.CreatedTicket{
		Platform:  platform,
		TicketID:  ticketID,
		MappingID: mappingID,
	}

	snapshot.SnapshotData.CreatedTickets = append(snapshot.SnapshotData.CreatedTickets, createdTicket)

	if err := ss.db.UpdateRollbackSnapshot(snapshot); err != nil {
		return fmt.Errorf("failed to update snapshot: %w", err)
	}

	log.Printf("SnapshotService: Recorded ticket creation: %s %s in operation %d\n",
		platform, ticketID, operationID)

	return nil
}

// RecordTicketUpdate records a ticket update before it happens
func (ss *SnapshotService) RecordTicketUpdate(operationID int, platform, ticketID, oldStatus, newStatus string, originalData map[string]interface{}) error {
	snapshot, err := ss.db.GetSnapshotByOperationID(operationID)
	if err != nil {
		return fmt.Errorf("snapshot not found for operation %d: %w", operationID, err)
	}

	// Check if we already have this ticket in the snapshot
	found := false
	for i, ts := range snapshot.SnapshotData.OriginalTickets {
		if ts.Platform == platform && ts.TicketID == ticketID {
			// Update the new status (keep original status as is)
			snapshot.SnapshotData.OriginalTickets[i].NewStatus = newStatus
			found = true
			break
		}
	}

	if !found {
		// Add new ticket state
		ticketState := database.TicketState{
			Platform:       platform,
			TicketID:       ticketID,
			OriginalStatus: oldStatus,
			NewStatus:      newStatus,
			OriginalData:   originalData,
		}
		snapshot.SnapshotData.OriginalTickets = append(snapshot.SnapshotData.OriginalTickets, ticketState)
	}

	if err := ss.db.UpdateRollbackSnapshot(snapshot); err != nil {
		return fmt.Errorf("failed to update snapshot: %w", err)
	}

	log.Printf("SnapshotService: Recorded ticket update: %s %s (%s -> %s) in operation %d\n",
		platform, ticketID, oldStatus, newStatus, operationID)

	return nil
}

// RecordMappingCreation records a mapping creation
func (ss *SnapshotService) RecordMappingCreation(operationID, mappingID int, mapping *database.TicketMapping) error {
	snapshot, err := ss.db.GetSnapshotByOperationID(operationID)
	if err != nil {
		return fmt.Errorf("snapshot not found for operation %d: %w", operationID, err)
	}

	mappingChange := database.MappingChange{
		MappingID:  mappingID,
		Action:     "created",
		OldMapping: nil,
		NewMapping: mapping,
	}

	snapshot.SnapshotData.UpdatedMappings = append(snapshot.SnapshotData.UpdatedMappings, mappingChange)

	if err := ss.db.UpdateRollbackSnapshot(snapshot); err != nil {
		return fmt.Errorf("failed to update snapshot: %w", err)
	}

	log.Printf("SnapshotService: Recorded mapping creation: ID %d in operation %d\n",
		mappingID, operationID)

	return nil
}

// RecordIgnoreChange records a change to ignore status
func (ss *SnapshotService) RecordIgnoreChange(operationID int, ticketID, oldIgnoreType, newIgnoreType string) error {
	snapshot, err := ss.db.GetSnapshotByOperationID(operationID)
	if err != nil {
		return fmt.Errorf("snapshot not found for operation %d: %w", operationID, err)
	}

	ignoreChange := database.IgnoreChange{
		TicketID:      ticketID,
		OldIgnoreType: oldIgnoreType,
		NewIgnoreType: newIgnoreType,
	}

	snapshot.SnapshotData.IgnoreChanges = append(snapshot.SnapshotData.IgnoreChanges, ignoreChange)

	if err := ss.db.UpdateRollbackSnapshot(snapshot); err != nil {
		return fmt.Errorf("failed to update snapshot: %w", err)
	}

	log.Printf("SnapshotService: Recorded ignore change for ticket %s (%s -> %s) in operation %d\n",
		ticketID, oldIgnoreType, newIgnoreType, operationID)

	return nil
}

// GetSnapshotSummary returns a summary of what would be rolled back
func (ss *SnapshotService) GetSnapshotSummary(operationID int) (*SnapshotSummary, error) {
	snapshot, err := ss.db.GetSnapshotByOperationID(operationID)
	if err != nil {
		return nil, fmt.Errorf("snapshot not found: %w", err)
	}

	operation, err := ss.db.GetOperation(operationID)
	if err != nil {
		return nil, fmt.Errorf("operation not found: %w", err)
	}

	summary := &SnapshotSummary{
		OperationID:     operationID,
		TicketsCreated:  len(snapshot.SnapshotData.CreatedTickets),
		TicketsUpdated:  len(snapshot.SnapshotData.OriginalTickets),
		MappingsChanged: len(snapshot.SnapshotData.UpdatedMappings),
		IgnoreChanges:   len(snapshot.SnapshotData.IgnoreChanges),
		TotalChanges:    len(snapshot.SnapshotData.CreatedTickets) + len(snapshot.SnapshotData.OriginalTickets) + len(snapshot.SnapshotData.UpdatedMappings),
	}

	// Check if can rollback
	canRollback := true
	if operation.Status != "completed" {
		canRollback = false
	}
	if time.Now().After(snapshot.ExpiresAt) {
		canRollback = false
	}

	summary.CanRollback = canRollback
	summary.RollbackDeadline = snapshot.ExpiresAt

	return summary, nil
}

// SnapshotSummary provides a summary of what will be rolled back
type SnapshotSummary struct {
	OperationID      int       `json:"operation_id"`
	TotalChanges     int       `json:"total_changes"`
	TicketsCreated   int       `json:"tickets_created"`
	TicketsUpdated   int       `json:"tickets_updated"`
	MappingsChanged  int       `json:"mappings_changed"`
	IgnoreChanges    int       `json:"ignore_changes"`
	CanRollback      bool      `json:"can_rollback"`
	RollbackDeadline time.Time `json:"rollback_deadline,omitempty"`
}

// CleanupExpiredSnapshots removes expired snapshots
func (ss *SnapshotService) CleanupExpiredSnapshots() error {
	return ss.db.CleanupExpiredSnapshots()
}
