package sync

import (
	"asana-youtrack-sync/database"
	"fmt"
	"log"
	"time"
)

// RollbackRestoreService handles the actual restoration of data during rollback
type RollbackRestoreService struct {
	db              *database.DB
	snapshotService *SnapshotService
	auditService    *AuditService
}

// NewRollbackRestoreService creates a new rollback restore service
func NewRollbackRestoreService(db *database.DB, snapshotService *SnapshotService, auditService *AuditService) *RollbackRestoreService {
	return &RollbackRestoreService{
		db:              db,
		snapshotService: snapshotService,
		auditService:    auditService,
	}
}

// RollbackResult represents the result of a rollback operation
type RollbackResult struct {
	Success           bool     `json:"success"`
	TicketsDeleted    int      `json:"tickets_deleted"`
	TicketsRestored   int      `json:"tickets_restored"`
	MappingsReverted  int      `json:"mappings_reverted"`
	IgnoresReverted   int      `json:"ignores_reverted"`
	Errors            []string `json:"errors"`
	PartialSuccess    bool     `json:"partial_success"`
}

// PerformRollback executes the complete rollback operation
func (rrs *RollbackRestoreService) PerformRollback(operationID, userID int, userEmail string, youtrackService YouTrackDeleter, asanaService AsanaDeleter) (*RollbackResult, error) {
	result := &RollbackResult{
		Success: false,
		Errors:  []string{},
	}

	// Get the operation
	operation, err := rrs.db.GetOperation(operationID)
	if err != nil {
		return nil, fmt.Errorf("operation not found: %w", err)
	}

	// Verify ownership
	if operation.UserID != userID {
		return nil, fmt.Errorf("unauthorized: operation belongs to different user")
	}

	// Check if operation can be rolled back
	if operation.Status != "completed" {
		return nil, fmt.Errorf("can only rollback completed operations, current status: %s", operation.Status)
	}

	// Get the snapshot
	snapshot, err := rrs.db.GetSnapshotByOperationID(operationID)
	if err != nil {
		return nil, fmt.Errorf("snapshot not found: %w", err)
	}

	// Check expiration
	if time.Now().After(snapshot.ExpiresAt) {
		return nil, fmt.Errorf("snapshot has expired, cannot rollback operations older than 24 hours")
	}

	log.Printf("RollbackRestore: Starting rollback for operation %d\n", operationID)

	// Create rollback operation record
	rollbackOp, err := rrs.db.CreateOperation(userID, "rollback", map[string]interface{}{
		"target_operation_id": operationID,
		"rollback_started":    time.Now(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create rollback operation: %w", err)
	}

	// Update status to in progress
	rrs.db.UpdateOperationStatus(rollbackOp.ID, "in_progress", nil)

	// Step 1: Delete created tickets
	for _, created := range snapshot.SnapshotData.CreatedTickets {
		var err error
		if created.Platform == "youtrack" {
			err = youtrackService.DeleteIssue(userID, created.TicketID)
		} else if created.Platform == "asana" {
			err = asanaService.DeleteTask(userID, created.TicketID)
		}

		if err != nil {
			errMsg := fmt.Sprintf("Failed to delete %s ticket %s: %v", created.Platform, created.TicketID, err)
			result.Errors = append(result.Errors, errMsg)
			log.Printf("RollbackRestore ERROR: %s\n", errMsg)
		} else {
			result.TicketsDeleted++
			log.Printf("RollbackRestore: Deleted %s ticket %s\n", created.Platform, created.TicketID)

			// Log to audit
			rrs.auditService.LogTicketDeleted(rollbackOp.ID, userEmail, created.TicketID, created.Platform)
		}
	}

	// Step 2: Restore original ticket states
	for _, ticketState := range snapshot.SnapshotData.OriginalTickets {
		var err error
		if ticketState.Platform == "youtrack" {
			err = youtrackService.UpdateIssueStatus(userID, ticketState.TicketID, ticketState.OriginalStatus)
		} else if ticketState.Platform == "asana" {
			err = asanaService.UpdateTaskStatus(userID, ticketState.TicketID, ticketState.OriginalStatus)
		}

		if err != nil {
			errMsg := fmt.Sprintf("Failed to restore %s ticket %s to status '%s': %v",
				ticketState.Platform, ticketState.TicketID, ticketState.OriginalStatus, err)
			result.Errors = append(result.Errors, errMsg)
			log.Printf("RollbackRestore ERROR: %s\n", errMsg)
		} else {
			result.TicketsRestored++
			log.Printf("RollbackRestore: Restored %s ticket %s to status '%s'\n",
				ticketState.Platform, ticketState.TicketID, ticketState.OriginalStatus)

			// Log to audit
			rrs.auditService.LogStatusChange(rollbackOp.ID, userEmail, ticketState.TicketID,
				ticketState.Platform, ticketState.NewStatus, ticketState.OriginalStatus)
		}
	}

	// Step 3: Revert mapping changes
	for _, mappingChange := range snapshot.SnapshotData.UpdatedMappings {
		var err error
		switch mappingChange.Action {
		case "created":
			// Delete the mapping that was created
			err = rrs.db.DeleteTicketMapping(userID, mappingChange.MappingID)
			if err != nil {
				errMsg := fmt.Sprintf("Failed to delete mapping ID %d: %v", mappingChange.MappingID, err)
				result.Errors = append(result.Errors, errMsg)
			} else {
				result.MappingsReverted++
				log.Printf("RollbackRestore: Deleted mapping ID %d\n", mappingChange.MappingID)
			}

		case "updated":
			// Note: Mapping updates are not implemented in current system
			log.Printf("RollbackRestore: Mapping update rollback not implemented\n")

		case "deleted":
			// Note: Mapping deletion rollback not implemented in current system
			log.Printf("RollbackRestore: Mapping deletion rollback not implemented\n")
		}
	}

	// Step 4: Restore ignore state
	for _, ignoreChange := range snapshot.SnapshotData.IgnoreChanges {
		settings, err := rrs.db.GetUserSettings(userID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to get settings for ignore restore: %v", err))
			continue
		}

		// Remove current ignore state
		rrs.db.RemoveIgnoredTicket(userID, settings.AsanaProjectID, ignoreChange.TicketID, "")

		// Restore old ignore state
		if ignoreChange.OldIgnoreType != "none" {
			_, err := rrs.db.AddIgnoredTicket(userID, settings.AsanaProjectID, ignoreChange.TicketID, ignoreChange.OldIgnoreType)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to restore ignore state for %s: %v", ignoreChange.TicketID, err))
			} else {
				result.IgnoresReverted++
				log.Printf("RollbackRestore: Restored ignore state for %s to '%s'\n",
					ignoreChange.TicketID, ignoreChange.OldIgnoreType)
			}
		} else {
			result.IgnoresReverted++
			log.Printf("RollbackRestore: Removed ignore state for %s\n", ignoreChange.TicketID)
		}
	}

	// Step 5: Mark original operation as rolled back
	err = rrs.db.UpdateOperationStatus(operationID, "rolled_back", nil)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to update original operation status: %v", err))
	}

	// Step 6: Mark rollback operation as completed
	if len(result.Errors) > 0 {
		result.PartialSuccess = true
		errorMsg := fmt.Sprintf("Rollback completed with %d errors", len(result.Errors))
		rrs.db.UpdateOperationStatus(rollbackOp.ID, "completed", &errorMsg)
	} else {
		result.Success = true
		rrs.db.UpdateOperationStatus(rollbackOp.ID, "completed", nil)
	}

	// Log the rollback in audit
	rollbackDetails := fmt.Sprintf("Deleted: %d, Restored: %d, Mappings: %d, Ignores: %d, Errors: %d",
		result.TicketsDeleted, result.TicketsRestored, result.MappingsReverted, result.IgnoresReverted, len(result.Errors))
	rrs.auditService.LogRollback(rollbackOp.ID, userEmail, rollbackDetails)

	log.Printf("RollbackRestore: Completed rollback for operation %d - %s\n", operationID, rollbackDetails)

	return result, nil
}

// CanRollback checks if an operation can be rolled back
func (rrs *RollbackRestoreService) CanRollback(operationID int) (bool, string) {
	operation, err := rrs.db.GetOperation(operationID)
	if err != nil {
		return false, "Operation not found"
	}

	if operation.Status != "completed" {
		return false, fmt.Sprintf("Operation status is '%s', must be 'completed'", operation.Status)
	}

	snapshot, err := rrs.db.GetSnapshotByOperationID(operationID)
	if err != nil {
		return false, "Snapshot not found"
	}

	if time.Now().After(snapshot.ExpiresAt) {
		return false, "Snapshot expired (>24 hours old)"
	}

	return true, ""
}

// Interface definitions for external services (for dependency injection)
type YouTrackDeleter interface {
	DeleteIssue(userID int, issueID string) error
	UpdateIssueStatus(userID int, issueID, status string) error
}

type AsanaDeleter interface {
	DeleteTask(userID int, taskID string) error
	UpdateTaskStatus(userID int, taskID, status string) error
}
