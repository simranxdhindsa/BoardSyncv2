package sync

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"asana-youtrack-sync/auth"
	"asana-youtrack-sync/database"
	"asana-youtrack-sync/legacy"
)

// HandleRollback handles rollback requests
func HandleRollback(
	rollbackRestoreService *RollbackRestoreService,
	youtrackService *legacy.YouTrackService,
	asanaService *legacy.AsanaService,
	wsManager *WebSocketManager,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.GetUserFromContext(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userID := user.UserID

		vars := mux.Vars(r)
		operationIDStr := vars["id"]
		operationID, err := strconv.Atoi(operationIDStr)
		if err != nil {
			http.Error(w, "Invalid operation ID", http.StatusBadRequest)
			return
		}

		// Check if can rollback
		canRollback, reason := rollbackRestoreService.CanRollback(operationID)
		if !canRollback {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   reason,
			})
			return
		}

		// Perform rollback
		result, err := rollbackRestoreService.PerformRollback(
			operationID,
			userID,
			user.Email,
			youtrackService,
			asanaService,
		)

		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		// Notify via WebSocket
		wsManager.SendToUser(userID, "rollback_complete", map[string]interface{}{
			"operation_id":      operationID,
			"tickets_deleted":   result.TicketsDeleted,
			"tickets_restored":  result.TicketsRestored,
			"mappings_reverted": result.MappingsReverted,
			"errors":            result.Errors,
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": result.Success || result.PartialSuccess,
			"result":  result,
		})
	}
}

// HandleGetAuditLogs handles requests for audit logs with filtering
func HandleGetAuditLogs(auditService *AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.GetUserFromContext(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Parse query parameters for filtering
		filter := database.AuditLogFilter{
			UserEmail:  r.URL.Query().Get("user_email"),
			TicketID:   r.URL.Query().Get("ticket_id"),
			Platform:   r.URL.Query().Get("platform"),
			ActionType: r.URL.Query().Get("action_type"),
			Limit:      50, // Default limit
		}

		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if limit, err := strconv.Atoi(limitStr); err == nil {
				filter.Limit = limit
			}
		}

		// Parse date filters
		if startDateStr := r.URL.Query().Get("start_date"); startDateStr != "" {
			if startDate, err := time.Parse("2006-01-02", startDateStr); err == nil {
				filter.StartDate = startDate
			}
		}

		if endDateStr := r.URL.Query().Get("end_date"); endDateStr != "" {
			if endDate, err := time.Parse("2006-01-02", endDateStr); err == nil {
				filter.EndDate = endDate
			}
		}

		// If no user_email filter specified, only show logs for current user
		if filter.UserEmail == "" {
			filter.UserEmail = user.Email
		}

		logs, err := auditService.GetFilteredAuditLogs(filter)
		if err != nil {
			http.Error(w, "Failed to retrieve audit logs: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"logs":    logs,
			"count":   len(logs),
		})
	}
}

// HandleGetTicketHistory handles requests for a specific ticket's history
func HandleGetTicketHistory(auditService *AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := auth.GetUserFromContext(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		ticketID := vars["ticket_id"]

		if ticketID == "" {
			http.Error(w, "Ticket ID is required", http.StatusBadRequest)
			return
		}

		logs, err := auditService.GetTicketHistory(ticketID)
		if err != nil {
			http.Error(w, "Failed to retrieve ticket history: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":   true,
			"ticket_id": ticketID,
			"history":   logs,
			"count":     len(logs),
		})
	}
}

// HandleExportAuditLogs handles CSV export of audit logs
func HandleExportAuditLogs(auditService *AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.GetUserFromContext(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Parse query parameters for filtering
		filter := database.AuditLogFilter{
			UserEmail:  user.Email, // Only export current user's logs
			Platform:   r.URL.Query().Get("platform"),
			ActionType: r.URL.Query().Get("action_type"),
			Limit:      1000, // Max export limit
		}

		// Parse date filters
		if startDateStr := r.URL.Query().Get("start_date"); startDateStr != "" {
			if startDate, err := time.Parse("2006-01-02", startDateStr); err == nil {
				filter.StartDate = startDate
			}
		}

		if endDateStr := r.URL.Query().Get("end_date"); endDateStr != "" {
			if endDate, err := time.Parse("2006-01-02", endDateStr); err == nil {
				filter.EndDate = endDate
			}
		}

		csvData, err := auditService.ExportAuditLogsToCSV(filter)
		if err != nil {
			http.Error(w, "Failed to export audit logs: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Set headers for CSV download
		filename := fmt.Sprintf("audit_logs_%s.csv", time.Now().Format("2006-01-02_15-04-05"))
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		w.Write([]byte(csvData))
	}
}

// HandleGetSnapshotSummary handles requests for snapshot summaries
func HandleGetSnapshotSummary(snapshotService *SnapshotService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.GetUserFromContext(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		operationIDStr := vars["id"]
		operationID, err := strconv.Atoi(operationIDStr)
		if err != nil {
			http.Error(w, "Invalid operation ID", http.StatusBadRequest)
			return
		}

		summary, err := snapshotService.GetSnapshotSummary(operationID)
		if err != nil {
			http.Error(w, "Failed to get snapshot summary: "+err.Error(), http.StatusNotFound)
			return
		}

		_ = user // User validation passed

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"summary": summary,
		})
	}
}

// HandleGetOperationAuditLogs retrieves all audit logs for a specific operation
func HandleGetOperationAuditLogs(auditService *AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := auth.GetUserFromContext(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		operationIDStr := vars["id"]
		operationID, err := strconv.Atoi(operationIDStr)
		if err != nil {
			http.Error(w, "Invalid operation ID", http.StatusBadRequest)
			return
		}

		logs, err := auditService.GetOperationHistory(operationID)
		if err != nil {
			http.Error(w, "Failed to retrieve operation audit logs: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":      true,
			"operation_id": operationID,
			"logs":         logs,
			"count":        len(logs),
		})
	}
}
