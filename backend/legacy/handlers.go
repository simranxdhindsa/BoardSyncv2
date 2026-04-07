package legacy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"asana-youtrack-sync/auth"
	configpkg "asana-youtrack-sync/config"
	"asana-youtrack-sync/database"
	"asana-youtrack-sync/utils"
)

// Handler manages all legacy API endpoints
type Handler struct {
	db              *database.DB
	configService   *configpkg.Service
	analysisService *AnalysisService
	syncService     *SyncService
	deleteService   *DeleteService
	ignoreService   *IgnoreService
	snapshotService interface {
		CreatePreSyncSnapshot(userID, operationID int, syncType string) (*database.RollbackSnapshot, error)
		RecordTicketCreation(operationID int, platform, ticketID string, mappingID int) error
		RecordTicketUpdate(operationID int, platform, ticketID, oldStatus, newStatus string, originalData map[string]interface{}) error
	}
}

// NewHandler creates a new legacy handler with all services
func NewHandler(db *database.DB, configService *configpkg.Service, snapshotService interface {
	CreatePreSyncSnapshot(userID, operationID int, syncType string) (*database.RollbackSnapshot, error)
	RecordTicketCreation(operationID int, platform, ticketID string, mappingID int) error
	RecordTicketUpdate(operationID int, platform, ticketID, oldStatus, newStatus string, originalData map[string]interface{}) error
}) *Handler {
	return &Handler{
		db:              db,
		configService:   configService,
		analysisService: NewAnalysisService(db, configService),
		syncService:     NewSyncService(db, configService),
		deleteService:   NewDeleteService(configService),
		ignoreService:   NewIgnoreService(db, configService),
		snapshotService: snapshotService,
	}
}

// HealthCheck provides service health information
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "enhanced-asana-youtrack-sync",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "4.0",
		"features": []string{
			"User-specific database settings",
			"Tag/Subsystem synchronization",
			"Individual ticket creation",
			"Enhanced status parsing",
			"Tag mismatch detection",
			"Bulk ticket deletion",
			"Authentication-based operations",
			"Modular service architecture",
			"Database-backed ignored tickets per project",
			"Column-aware create and sync operations",
		},
		"columns": map[string]interface{}{
			"syncable":     SyncableColumns,
			"display_only": DisplayOnlyColumns,
		},
	}

	utils.SendSuccess(w, response, "Service is healthy")
}

// StatusCheck provides service status information for authenticated users
func (h *Handler) StatusCheck(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	// Get user settings to show configuration status
	settings, err := h.configService.GetSettings(user.UserID)
	if err != nil {
		utils.SendInternalError(w, "Failed to get user settings")
		return
	}

	// Check if APIs are properly configured
	asanaConfigured := settings.AsanaPAT != "" && settings.AsanaProjectID != ""
	youtrackConfigured := settings.YouTrackBaseURL != "" &&
		settings.YouTrackToken != "" &&
		settings.YouTrackProjectID != ""

	response := map[string]interface{}{
		"service":             "enhanced-asana-youtrack-sync",
		"user_id":             user.UserID,
		"username":            user.Username,
		"asana_configured":    asanaConfigured,
		"youtrack_configured": youtrackConfigured,
		"asana_project":       settings.AsanaProjectID,
		"youtrack_project":    settings.YouTrackProjectID,
		"columns": map[string]interface{}{
			"syncable":     SyncableColumns,
			"display_only": DisplayOnlyColumns,
		},
		"ignored_tickets": h.ignoreService.CountIgnored(user.UserID),
		"endpoints": []string{
			"GET /analyze - Analyze ticket differences",
			"POST /create?column=COLUMN - Create missing tickets (column-aware)",
			"POST /create-single - Create individual ticket",
			"GET/POST /sync?column=COLUMN - Sync mismatched tickets (column-aware)",
			"GET/POST /ignore - Manage ignored tickets",
			"GET /tickets - Get tickets by type",
			"POST /delete-tickets - Delete tickets (bulk)",
		},
	}

	utils.SendSuccess(w, response, "Service status retrieved")
}

// resolveColumns maps a URL ?column= parameter to the slice of Asana section names to analyze.
// Handles both hardcoded legacy aliases and dynamic user-defined column names (e.g. "mobile_done").
func resolveColumns(columnFilter string) (columns []string, mappedName string) {
	if columnFilter == "" || columnFilter == "all_syncable" {
		return SyncableColumns, "all_syncable"
	}
	// Hardcoded aliases for commonly used column slugs
	aliases := map[string]string{
		"backlog":         "backlog",
		"in_progress":     "in progress",
		"dev":             "dev",
		"stage":           "stage",
		"prod":            "prod",
		"blocked":         "blocked",
		"ready_for_stage": "ready for stage",
		"findings":        "findings",
	}
	if mapped, ok := aliases[columnFilter]; ok {
		return []string{mapped}, mapped
	}
	// Dynamic column from DB: convert underscores to spaces.
	// FilterTasksByColumns handles arbitrary names via strings.Contains.
	dynamic := strings.ReplaceAll(columnFilter, "_", " ")
	fmt.Printf("ANALYZE: Dynamic column '%s' → '%s'\n", columnFilter, dynamic)
	return []string{dynamic}, dynamic
}

// AnalyzeTickets performs comprehensive ticket analysis
func (h *Handler) AnalyzeTickets(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	// Get column filter from query parameters
	columnFilter := r.URL.Query().Get("column")
	fmt.Printf("ANALYZE: User %d analyzing with column filter: '%s'\n", user.UserID, columnFilter)
	columnsToAnalyze, mappedColumnName := resolveColumns(columnFilter)

	// Perform analysis
	analysis, err := h.analysisService.PerformAnalysis(user.UserID, columnsToAnalyze)
	if err != nil {
		fmt.Printf("ANALYZE: Analysis failed for user %d: %v\n", user.UserID, err)
		utils.SendInternalError(w, fmt.Sprintf("Analysis failed: %v", err))
		return
	}

	// Get summary statistics
	summary, err := h.analysisService.GetAnalysisSummary(user.UserID, columnsToAnalyze)
	if err != nil {
		utils.SendInternalError(w, fmt.Sprintf("Failed to get summary: %v", err))
		return
	}

	fmt.Printf("ANALYZE: Complete for user %d - %d matched, %d mismatched, %d missing\n",
		user.UserID, len(analysis.Matched), len(analysis.Mismatched), len(analysis.MissingYouTrack))

	response := map[string]interface{}{
		"analysis":         analysis,
		"column_filter":    columnFilter,
		"mapped_column":    mappedColumnName,
		"analyzed_columns": columnsToAnalyze,
		"summary":          summary,
	}

	utils.SendSuccess(w, response, "Analysis completed successfully")
}

// AnalyzeWithProgress streams analysis progress via Server-Sent Events, then sends
// the full analysis result as the final event.
func (h *Handler) AnalyzeWithProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.WriteHeader(http.StatusOK)
		return
	}

	user, ok := auth.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var mu sync.Mutex
	sendEvent := func(v interface{}) {
		mu.Lock()
		defer mu.Unlock()
		data, _ := json.Marshal(v)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	// Heartbeat: send a SSE comment every 5s so proxies/browsers don't kill the connection
	// during long silent stretches (title matching, large fetches, etc.)
	stopHeartbeat := make(chan struct{})
	defer close(stopHeartbeat)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stopHeartbeat:
				return
			case <-ticker.C:
				mu.Lock()
				fmt.Fprintf(w, ": keepalive\n\n")
				flusher.Flush()
				mu.Unlock()
			}
		}
	}()

	columnFilter := r.URL.Query().Get("column")
	columnsToAnalyze, mappedColumnName := resolveColumns(columnFilter)

	progressFn := func(stage string, processed, total int) {
		sendEvent(map[string]interface{}{
			"stage":     stage,
			"processed": processed,
			"total":     total,
		})
	}

	analysis, err := h.analysisService.PerformAnalysis(user.UserID, columnsToAnalyze, progressFn)
	if err != nil {
		sendEvent(map[string]interface{}{"error": err.Error()})
		return
	}

	summary, _ := h.analysisService.GetAnalysisSummary(user.UserID, columnsToAnalyze)

	sendEvent(map[string]interface{}{
		"done":             true,
		"analysis":         analysis,
		"summary":          summary,
		"column_filter":    columnFilter,
		"mapped_column":    mappedColumnName,
		"analyzed_columns": columnsToAnalyze,
	})
}

// GetTicketsByType returns tickets of a specific type
func (h *Handler) GetTicketsByType(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	ticketType := r.URL.Query().Get("type")
	column := r.URL.Query().Get("column")

	if ticketType == "" {
		utils.SendBadRequest(w, "Missing 'type' parameter")
		return
	}

	// Get tickets by type
	tickets, err := h.analysisService.GetTicketsByType(user.UserID, ticketType, column)
	if err != nil {
		utils.SendInternalError(w, fmt.Sprintf("Failed to get tickets: %v", err))
		return
	}

	// Calculate count based on ticket type
	var count int
	switch v := tickets.(type) {
	case []MatchedTicket:
		count = len(v)
	case []MismatchedTicket:
		count = len(v)
	case []AsanaTask:
		count = len(v)
	case []YouTrackIssue:
		count = len(v)
	case []string:
		count = len(v)
	default:
		count = 0
	}

	response := map[string]interface{}{
		"type":    ticketType,
		"column":  column,
		"tickets": tickets,
		"count":   count,
	}

	utils.SendSuccess(w, response, fmt.Sprintf("%s tickets retrieved successfully", ticketType))
}

// CreateMissingTickets creates missing tickets in YouTrack
func (h *Handler) CreateMissingTickets(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	if r.Method != "POST" && r.Method != "GET" {
		utils.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED",
			"Method not allowed. Use POST or GET.", "")
		return
	}

	// Get column filter from query parameters
	columnFilter := r.URL.Query().Get("column")

	// Resolve column name using shared helper (handles dynamic columns like "to_do" → "to do")
	var mappedColumn string
	if columnFilter != "" && columnFilter != "all_syncable" {
		cols, _ := resolveColumns(columnFilter)
		if len(cols) > 0 {
			mappedColumn = cols[0]
		}
		fmt.Printf("CREATE: Column filter '%s' resolved to '%s'\n", columnFilter, mappedColumn)
	}

	// Create sync operation record BEFORE performing the operation
	columnDisplay := mappedColumn
	if columnDisplay == "" {
		columnDisplay = "all columns"
	}

	operationType := "Ticket Creation"
	operationData := map[string]interface{}{
		"action": "create_missing_tickets",
		"column": columnDisplay,
	}

	operation, err := h.db.CreateOperation(user.UserID, operationType, operationData)
	if err != nil {
		utils.SendInternalError(w, fmt.Sprintf("Failed to create operation record: %v", err))
		return
	}

	// Create snapshot BEFORE creating tickets
	if h.snapshotService != nil {
		_, err = h.snapshotService.CreatePreSyncSnapshot(user.UserID, operation.ID, "create_tickets")
		if err != nil {
			fmt.Printf("WARNING: Failed to create snapshot: %v\n", err)
		}
	}

	result, err := h.syncService.CreateMissingTickets(user.UserID, mappedColumn)
	if err != nil {
		// Update operation status to failed
		if operation != nil {
			errMsg := err.Error()
			h.db.UpdateOperationStatus(operation.ID, "failed", &errMsg)
		}
		utils.SendInternalError(w, fmt.Sprintf("Failed to create tickets: %v", err))
		return
	}

	// Extract meaningful data from result
	createdCount := 0
	skippedCount := 0
	failedCount := 0

	if created, ok := result["created"].(int); ok {
		createdCount = created
	}
	if skipped, ok := result["skipped"].(int); ok {
		skippedCount = skipped
	}
	if failed, ok := result["failed"].(int); ok {
		failedCount = failed
	}

	// Update operation with detailed information
	operation.OperationData = map[string]interface{}{
		"action":  "create_missing_tickets",
		"column":  columnDisplay,
		"created": createdCount,
		"skipped": skippedCount,
		"failed":  failedCount,
		"total":   createdCount + skippedCount + failedCount,
	}

	// Mark as completed
	h.db.UpdateOperationStatus(operation.ID, "completed", nil)

	utils.SendSuccess(w, result, "Ticket creation completed")
}

// CreateSingleTicket creates a single ticket in YouTrack
func (h *Handler) CreateSingleTicket(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	if r.Method != "POST" {
		utils.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED",
			"Method not allowed. Use POST.", "")
		return
	}

	var req CreateSingleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendBadRequest(w, "Invalid request body")
		return
	}

	if req.TaskID == "" {
		utils.SendBadRequest(w, "task_id is required")
		return
	}

	result, err := h.syncService.CreateSingleTicket(user.UserID, req.TaskID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			utils.SendNotFound(w, err.Error())
		} else {
			utils.SendInternalError(w, err.Error())
		}
		return
	}

	// If the service returned status:"failed", surface it as HTTP 422 so the
	// frontend !response.ok branch fires and the card stays visible.
	if status, _ := result["status"].(string); status == "failed" {
		errMsg, _ := result["error"].(string)
		if errMsg == "" {
			errMsg = "YouTrack ticket creation failed"
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   errMsg,
			"data":    result,
		})
		return
	}

	utils.SendSuccess(w, result, "Single ticket operation completed")
}

// SyncMismatchedTickets synchronizes mismatched tickets
func (h *Handler) SyncMismatchedTickets(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	// Get column filter from query parameters
	columnFilter := r.URL.Query().Get("column")

	var mappedColumn string
	if columnFilter != "" && columnFilter != "all_syncable" {
		cols, _ := resolveColumns(columnFilter)
		if len(cols) > 0 {
			mappedColumn = cols[0]
		}
		fmt.Printf("SYNC: Column filter '%s' resolved to '%s'\n", columnFilter, mappedColumn)
	}

	if r.Method == "GET" {
		// Return available mismatched tickets for preview
		result, err := h.syncService.GetMismatchedTickets(user.UserID, mappedColumn)
		if err != nil {
			utils.SendInternalError(w, fmt.Sprintf("Failed to get mismatched tickets: %v", err))
			return
		}
		utils.SendSuccess(w, result, "Mismatched tickets retrieved")
		return
	}

	if r.Method != "POST" {
		utils.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED",
			"Method not allowed. Use GET to see available tickets, POST to sync.", "")
		return
	}

	var requests []SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		utils.SendBadRequest(w, "Invalid JSON format")
		return
	}

	// Validate sync requests
	if err := h.syncService.ValidateSyncRequests(requests); err != nil {
		utils.SendBadRequest(w, err.Error())
		return
	}

	// Create sync operation record BEFORE performing the operation
	columnDisplay := mappedColumn
	if columnDisplay == "" {
		columnDisplay = "all columns"
	}

	operationType := "Ticket Sync"
	operationData := map[string]interface{}{
		"action": "sync_mismatched_tickets",
		"column": columnDisplay,
		"total":  len(requests),
	}

	operation, err := h.db.CreateOperation(user.UserID, operationType, operationData)
	if err != nil {
		utils.SendInternalError(w, fmt.Sprintf("Failed to create operation record: %v", err))
		return
	}

	// Create snapshot BEFORE syncing tickets
	if h.snapshotService != nil {
		_, err = h.snapshotService.CreatePreSyncSnapshot(user.UserID, operation.ID, "sync_tickets")
		if err != nil {
			fmt.Printf("WARNING: Failed to create snapshot: %v\n", err)
		}
	}

	result, err := h.syncService.SyncMismatchedTickets(user.UserID, requests, mappedColumn)
	if err != nil {
		// Update operation status to failed
		if operation != nil {
			errMsg := err.Error()
			h.db.UpdateOperationStatus(operation.ID, "failed", &errMsg)
		}
		utils.SendInternalError(w, fmt.Sprintf("Sync failed: %v", err))
		return
	}

	// Extract meaningful data from result
	syncedCount := 0
	failedCount := 0

	if synced, ok := result["synced"].(int); ok {
		syncedCount = synced
	}
	if failed, ok := result["failed"].(int); ok {
		failedCount = failed
	}

	// Update operation with detailed information
	operation.OperationData = map[string]interface{}{
		"action": "sync_mismatched_tickets",
		"column": columnDisplay,
		"synced": syncedCount,
		"failed": failedCount,
		"total":  len(requests),
	}

	// Mark as completed
	h.db.UpdateOperationStatus(operation.ID, "completed", nil)

	utils.SendSuccess(w, result, "Sync operation completed")
}

// ManageIgnoredTickets manages ignored tickets (both temporary and permanent)
func (h *Handler) ManageIgnoredTickets(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	switch r.Method {
	case "GET":
		// Return current ignore status
		status := h.ignoreService.GetIgnoreStatus(user.UserID)
		utils.SendSuccess(w, status, "Ignored tickets status retrieved")

	case "POST":
		// Process ignore request
		var req IgnoreRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			utils.SendBadRequest(w, "Invalid JSON format")
			return
		}

		if req.TicketID == "" || req.Action == "" || req.Type == "" {
			utils.SendBadRequest(w, "ticket_id, action, and type are required")
			return
		}

		err := h.ignoreService.ProcessIgnoreRequest(user.UserID, req.TicketID, req.Action, req.Type)
		if err != nil {
			utils.SendBadRequest(w, err.Error())
			return
		}

		response := map[string]interface{}{
			"action":    req.Action,
			"type":      req.Type,
			"ticket_id": req.TicketID,
		}

		utils.SendSuccess(w, response, "Ignore operation completed successfully")

	default:
		utils.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED",
			"Method not allowed. Use GET or POST.", "")
	}
}

// DeleteTickets handles bulk ticket deletion
func (h *Handler) DeleteTickets(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	if r.Method != "POST" {
		utils.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED",
			"Method not allowed. Use POST.", "")
		return
	}

	var req DeleteTicketsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendBadRequest(w, "Invalid JSON format")
		return
	}

	// Validate delete request
	if err := h.deleteService.ValidateDeleteRequest(req); err != nil {
		utils.SendBadRequest(w, err.Error())
		return
	}

	fmt.Printf("DELETE: Starting bulk delete of %d tickets from %s for user %d\n",
		len(req.TicketIDs), req.Source, user.UserID)

	// Perform bulk deletion
	response := h.deleteService.PerformBulkDelete(user.UserID, req.TicketIDs, req.Source)

	// Set appropriate HTTP status based on result
	httpStatus := http.StatusOK
	if response.Status == "failed" {
		httpStatus = http.StatusInternalServerError
	} else if response.Status == "partial" {
		httpStatus = http.StatusPartialContent
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   response.Status != "failed",
		"data":      response,
		"message":   response.Summary,
		"timestamp": time.Now(),
	})

	fmt.Printf("DELETE: Completed for user %d: %s\n", user.UserID, response.Summary)
}

// GetSyncStats provides comprehensive synchronization statistics
func (h *Handler) GetSyncStats(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	stats, err := h.syncService.GetSyncStats(user.UserID)
	if err != nil {
		utils.SendInternalError(w, fmt.Sprintf("Failed to get sync stats: %v", err))
		return
	}

	utils.SendSuccess(w, stats, "Sync statistics retrieved successfully")
}

// GetSyncableTickets returns tickets that can be synced
func (h *Handler) GetSyncableTickets(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	result, err := h.syncService.GetSyncableTickets(user.UserID)
	if err != nil {
		utils.SendInternalError(w, fmt.Sprintf("Failed to get syncable tickets: %v", err))
		return
	}

	utils.SendSuccess(w, result, "Syncable tickets retrieved successfully")
}

// SyncByColumn syncs tickets from a specific column
func (h *Handler) SyncByColumn(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	if r.Method != "POST" {
		utils.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED",
			"Method not allowed. Use POST.", "")
		return
	}

	column := r.URL.Query().Get("column")
	if column == "" {
		utils.SendBadRequest(w, "column parameter is required")
		return
	}

	result, err := h.syncService.SyncTicketsByColumn(user.UserID, column)
	if err != nil {
		utils.SendInternalError(w, fmt.Sprintf("Column sync failed: %v", err))
		return
	}

	utils.SendSuccess(w, result, fmt.Sprintf("Column '%s' sync completed", column))
}

// CreateByColumn creates missing tickets from a specific column
func (h *Handler) CreateByColumn(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	if r.Method != "POST" {
		utils.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED",
			"Method not allowed. Use POST.", "")
		return
	}

	column := r.URL.Query().Get("column")
	if column == "" {
		utils.SendBadRequest(w, "column parameter is required")
		return
	}

	result, err := h.syncService.CreateTicketsByColumn(user.UserID, column)
	if err != nil {
		utils.SendInternalError(w, fmt.Sprintf("Column create failed: %v", err))
		return
	}

	utils.SendSuccess(w, result, fmt.Sprintf("Column '%s' create completed", column))
}

// GetDeletionPreview provides a preview of what would be deleted
func (h *Handler) GetDeletionPreview(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	// Parse query parameters
	ticketIDs := r.URL.Query()["ticket_ids"]
	source := r.URL.Query().Get("source")

	if len(ticketIDs) == 0 {
		utils.SendBadRequest(w, "ticket_ids parameter is required")
		return
	}

	if source == "" {
		utils.SendBadRequest(w, "source parameter is required")
		return
	}

	preview, err := h.deleteService.GetDeletionPreview(user.UserID, ticketIDs, source)
	if err != nil {
		utils.SendInternalError(w, fmt.Sprintf("Failed to get deletion preview: %v", err))
		return
	}

	utils.SendSuccess(w, preview, "Deletion preview generated successfully")
}

// GetSyncPreview provides a preview of what would be synced
func (h *Handler) GetSyncPreview(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	// Parse query parameters
	ticketIDs := r.URL.Query()["ticket_ids"]

	if len(ticketIDs) == 0 {
		utils.SendBadRequest(w, "ticket_ids parameter is required")
		return
	}

	preview, err := h.syncService.GetSyncPreview(user.UserID, ticketIDs)
	if err != nil {
		utils.SendInternalError(w, fmt.Sprintf("Failed to get sync preview: %v", err))
		return
	}

	utils.SendSuccess(w, preview, "Sync preview generated successfully")
}

// backend/legacy/handlers_enhanced.go - ADD THESE TO EXISTING HANDLERS

// AnalyzeTicketsEnhanced performs comprehensive ticket analysis with filtering and sorting
func (h *Handler) AnalyzeTicketsEnhanced(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	// Get column filter from query parameters
	columnFilter := r.URL.Query().Get("column")

	// Parse filter and sort options from request body (if POST) or query params (if GET)
	var filter TicketFilter
	var sortOpts TicketSortOptions

	if r.Method == "POST" {
		var req struct {
			Column string            `json:"column"`
			Filter TicketFilter      `json:"filter"`
			Sort   TicketSortOptions `json:"sort"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			columnFilter = req.Column
			filter = req.Filter
			sortOpts = req.Sort
		}
	} else {
		// Parse from query parameters
		parseFilterFromQuery(r, &filter)
		parseSortFromQuery(r, &sortOpts)
	}

	fmt.Printf("ANALYZE: User %d analyzing with column: '%s', filter: %+v, sort: %+v\n", user.UserID, columnFilter, filter, sortOpts)

	// Determine columns to analyze
	var columnsToAnalyze []string
	var mappedColumnName string

	columnsToAnalyze, mappedColumnName = resolveColumns(columnFilter)

	// Perform analysis with filtering and sorting
	analysis, err := h.analysisService.PerformAnalysisWithFiltering(user.UserID, columnsToAnalyze, filter, sortOpts)
	if err != nil {
		fmt.Printf("ANALYZE: Analysis failed for user %d: %v\n", user.UserID, err)
		utils.SendInternalError(w, fmt.Sprintf("Analysis failed: %v", err))
		return
	}

	// Get summary statistics
	summary, err := h.analysisService.GetAnalysisSummary(user.UserID, columnsToAnalyze)
	if err != nil {
		utils.SendInternalError(w, fmt.Sprintf("Failed to get summary: %v", err))
		return
	}

	// Get available filter options
	filterOptions, err := h.analysisService.GetFilterOptions(user.UserID, columnsToAnalyze)
	if err != nil {
		fmt.Printf("ANALYZE: Failed to get filter options: %v\n", err)
	}

	fmt.Printf("ANALYZE: Complete for user %d - %d matched, %d mismatched, %d missing\n",
		user.UserID, len(analysis.Matched), len(analysis.Mismatched), len(analysis.MissingYouTrack))

	response := map[string]interface{}{
		"analysis":         analysis,
		"column_filter":    columnFilter,
		"mapped_column":    mappedColumnName,
		"analyzed_columns": columnsToAnalyze,
		"summary":          summary,
		"filter_options":   filterOptions,
		"applied_filter":   filter,
		"applied_sort":     sortOpts,
	}

	utils.SendSuccess(w, response, "Analysis completed successfully")
}

// GetChangedMappings removed - title/description change detection no longer needed

// GetFilterOptions returns available filter options
func (h *Handler) GetFilterOptions(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	columnFilter := r.URL.Query().Get("column")
	var columnsToAnalyze []string

	columnsToAnalyze, _ = resolveColumns(columnFilter)

	filterOptions, err := h.analysisService.GetFilterOptions(user.UserID, columnsToAnalyze)
	if err != nil {
		utils.SendInternalError(w, fmt.Sprintf("Failed to get filter options: %v", err))
		return
	}

	utils.SendSuccess(w, filterOptions, "Filter options retrieved successfully")
}

// Helper functions to parse filter and sort from query parameters

func parseFilterFromQuery(r *http.Request, filter *TicketFilter) {
	// Parse assignees (comma-separated)
	if assigneesStr := r.URL.Query().Get("assignees"); assigneesStr != "" {
		filter.Assignees = strings.Split(assigneesStr, ",")
	}

	// Parse priorities (comma-separated)
	if prioritiesStr := r.URL.Query().Get("priorities"); prioritiesStr != "" {
		filter.Priority = strings.Split(prioritiesStr, ",")
	}

	// Parse start date
	if startDateStr := r.URL.Query().Get("start_date"); startDateStr != "" {
		if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
			filter.StartDate = t
		}
	}

	// Parse end date
	if endDateStr := r.URL.Query().Get("end_date"); endDateStr != "" {
		if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
			filter.EndDate = t
		}
	}
}

func parseSortFromQuery(r *http.Request, sortOpts *TicketSortOptions) {
	sortOpts.SortBy = r.URL.Query().Get("sort_by")
	sortOpts.SortOrder = r.URL.Query().Get("sort_order")

	if sortOpts.SortOrder == "" {
		sortOpts.SortOrder = "asc"
	}
}

// backend/legacy/handlers.go - ADD THIS METHOD TO THE Handler STRUCT

// GetSyncService returns the sync service (helper method for main.go)
func (h *Handler) GetSyncService() *SyncService {
	return h.syncService
}

//DEBUG
// Add these handlers to backend/legacy/handlers.go

// // VerifyColumnsAndMapping verifies column detection and mapping
// func (h *Handler) VerifyColumnsAndMapping(w http.ResponseWriter, r *http.Request) {
// 	user, ok := auth.GetUserFromContext(r)
// 	if !ok {
// 		utils.SendUnauthorized(w, "Authentication required")
// 		return
// 	}

// 	fmt.Printf("VERIFY: User %d requested column verification\n", user.UserID)

// 	result, err := h.analysisService.VerifyColumnsAndMapping(user.UserID)
// 	if err != nil {
// 		utils.SendInternalError(w, fmt.Sprintf("Failed to verify columns: %v", err))
// 		return
// 	}

// 	utils.SendSuccess(w, result, "Column verification completed successfully")
// }

// // GetColumnMappingReport returns a human-readable mapping report
// func (h *Handler) GetColumnMappingReport(w http.ResponseWriter, r *http.Request) {
// 	user, ok := auth.GetUserFromContext(r)
// 	if !ok {
// 		utils.SendUnauthorized(w, "Authentication required")
// 		return
// 	}

// 	report, err := h.analysisService.GetColumnMappingReport(user.UserID)
// 	if err != nil {
// 		utils.SendInternalError(w, fmt.Sprintf("Failed to generate report: %v", err))
// 		return
// 	}

// 	utils.SendSuccess(w, report, "Column mapping report generated successfully")
// }

// GetYouTrackStatesRaw returns raw YouTrack state information for debugging
func (h *Handler) GetYouTrackStatesRaw(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	youtrackService := NewYouTrackService(h.configService)
	issues, err := youtrackService.GetIssues(user.UserID)
	if err != nil {
		utils.SendInternalError(w, fmt.Sprintf("Failed to get YouTrack issues: %v", err))
		return
	}

	// Extract all State field information
	stateInfo := []map[string]interface{}{}

	for _, issue := range issues {
		for _, field := range issue.CustomFields {
			if field.Name == "State" {
				info := map[string]interface{}{
					"issue_id":          issue.ID,
					"issue_summary":     issue.Summary,
					"raw_field_value":   field.Value,
					"extracted_status":  youtrackService.GetStatus(issue),
					"normalized_status": youtrackService.GetStatusNormalized(issue),
				}
				stateInfo = append(stateInfo, info)
			}
		}
	}

	result := map[string]interface{}{
		"total_issues": len(issues),
		"state_info":   stateInfo,
		"note":         "This shows the raw State field structure from YouTrack API",
	}

	utils.SendSuccess(w, result, "Raw YouTrack state information retrieved")
}

// MapTicket saves a mapping between an existing Asana task and a YouTrack issue
func (h *Handler) MapTicket(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	var req struct {
		AsanaTaskID     string `json:"asana_task_id"`
		YouTrackIssueID string `json:"youtrack_issue_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendBadRequest(w, "Invalid request body")
		return
	}
	if req.AsanaTaskID == "" || req.YouTrackIssueID == "" {
		utils.SendBadRequest(w, "asana_task_id and youtrack_issue_id are required")
		return
	}

	settings, err := h.configService.GetSettings(user.UserID)
	if err != nil {
		utils.SendInternalError(w, "Failed to get settings")
		return
	}

	_, err = h.db.CreateTicketMapping(user.UserID, settings.AsanaProjectID,
		req.AsanaTaskID, settings.YouTrackProjectID, req.YouTrackIssueID)
	if err != nil {
		utils.SendInternalError(w, fmt.Sprintf("Failed to create mapping: %v", err))
		return
	}

	fmt.Printf("MAP-TICKET: Mapped Asana %s <-> YT %s for user %d\n", req.AsanaTaskID, req.YouTrackIssueID, user.UserID)
	utils.SendSuccess(w, map[string]interface{}{"success": true, "message": "Mapping saved"}, "Mapping created successfully")
}

// AddToBoard adds a list of YouTrack issues to the configured agile board
func (h *Handler) AddToBoard(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	var req struct {
		IssueIDs []string `json:"issue_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IssueIDs) == 0 {
		utils.SendBadRequest(w, "issue_ids required")
		return
	}

	settings, err := h.configService.GetSettings(user.UserID)
	if err != nil || settings.YouTrackBoardID == "" {
		utils.SendBadRequest(w, "Board not configured in settings")
		return
	}

	ytService := NewYouTrackService(h.configService)
	results := map[string]string{}
	for _, issueID := range req.IssueIDs {
		if err := ytService.AssignIssueToBoard(user.UserID, issueID); err != nil {
			results[issueID] = "failed: " + err.Error()
		} else {
			results[issueID] = "ok"
		}
	}

	fmt.Printf("ADD-TO-BOARD: Processed %d issues for user %d\n", len(req.IssueIDs), user.UserID)
	utils.SendSuccess(w, results, fmt.Sprintf("Processed %d issues", len(req.IssueIDs)))
}

// SyncPriorities sets the Priority custom field on YouTrack issues to match the
// priority codes extracted from their Asana task titles (e.g. "P1", "A3").
// Request body: {"items": [{"youtrack_issue_id": "ARD-278", "priority": "P1"}, ...]}
func (h *Handler) SyncPriorities(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	var req struct {
		Items []struct {
			YouTrackIssueID string `json:"youtrack_issue_id"`
			Priority        string `json:"priority"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Items) == 0 {
		utils.SendBadRequest(w, "items required: [{youtrack_issue_id, priority}]")
		return
	}

	settings, err := h.configService.GetSettings(user.UserID)
	if err != nil {
		utils.SendInternalError(w, "Failed to load settings")
		return
	}

	ytService := NewYouTrackService(h.configService)
	synced, failed := 0, 0
	results := make([]map[string]interface{}, 0, len(req.Items))

	for _, item := range req.Items {
		if item.YouTrackIssueID == "" || item.Priority == "" {
			continue
		}
		err := ytService.SyncPriority(settings, item.YouTrackIssueID, item.Priority)
		entry := map[string]interface{}{
			"youtrack_issue_id": item.YouTrackIssueID,
			"priority":          item.Priority,
		}
		if err != nil {
			entry["status"] = "failed"
			entry["error"] = err.Error()
			failed++
			fmt.Printf("PRIORITY: Failed to set %s → %s: %v\n", item.YouTrackIssueID, item.Priority, err)
		} else {
			entry["status"] = "ok"
			synced++
		}
		results = append(results, entry)
	}

	utils.SendSuccess(w, map[string]interface{}{
		"synced":  synced,
		"failed":  failed,
		"results": results,
	}, fmt.Sprintf("Priority sync: %d synced, %d failed", synced, failed))
}

// Column verification endpoints
