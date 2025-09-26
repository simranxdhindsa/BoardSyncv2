package legacy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"asana-youtrack-sync/auth"
	configpkg "asana-youtrack-sync/config"
	"asana-youtrack-sync/utils"
)

// Handler manages all legacy API endpoints
type Handler struct {
	configService   *configpkg.Service
	analysisService *AnalysisService
	syncService     *SyncService
	deleteService   *DeleteService
	ignoreService   *IgnoreService
}

// NewHandler creates a new legacy handler with all services
func NewHandler(configService *configpkg.Service) *Handler {
	return &Handler{
		configService:   configService,
		analysisService: NewAnalysisService(configService),
		syncService:     NewSyncService(configService),
		deleteService:   NewDeleteService(configService),
		ignoreService:   NewIgnoreService(),
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
		"ignored_tickets": h.ignoreService.CountIgnored(),
		"endpoints": []string{
			"GET /analyze - Analyze ticket differences",
			"POST /create - Create missing tickets (bulk)",
			"POST /create-single - Create individual ticket",
			"GET/POST /sync - Sync mismatched tickets",
			"GET/POST /ignore - Manage ignored tickets",
			"GET /tickets - Get tickets by type",
			"POST /delete-tickets - Delete tickets (bulk)",
		},
	}

	utils.SendSuccess(w, response, "Service status retrieved")
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

	// Determine columns to analyze
	var columnsToAnalyze []string
	var mappedColumnName string

	if columnFilter == "" || columnFilter == "all_syncable" {
		columnsToAnalyze = SyncableColumns
		mappedColumnName = "all_syncable"
	} else {
		// Frontend to backend column name mapping
		columnMap := map[string]string{
			"backlog":         "backlog",
			"in_progress":     "in progress",
			"dev":             "dev",
			"stage":           "stage",
			"blocked":         "blocked",
			"ready_for_stage": "ready for stage",
			"findings":        "findings",
		}

		if mappedColumn, exists := columnMap[columnFilter]; exists {
			columnsToAnalyze = []string{mappedColumn}
			mappedColumnName = mappedColumn
		} else {
			fmt.Printf("ANALYZE: Unknown column '%s', using all syncable\n", columnFilter)
			columnsToAnalyze = SyncableColumns
			mappedColumnName = "all_syncable"
		}
	}

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

	fmt.Printf("TICKETS: Returning %d tickets of type '%s' for user %d\n",
		count, ticketType, user.UserID)

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

	result, err := h.syncService.CreateMissingTickets(user.UserID)
	if err != nil {
		utils.SendInternalError(w, fmt.Sprintf("Failed to create tickets: %v", err))
		return
	}

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

	utils.SendSuccess(w, result, "Single ticket operation completed")
}

// SyncMismatchedTickets synchronizes mismatched tickets
func (h *Handler) SyncMismatchedTickets(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	if r.Method == "GET" {
		// Return available mismatched tickets for preview
		result, err := h.syncService.GetMismatchedTickets(user.UserID)
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

	result, err := h.syncService.SyncMismatchedTickets(user.UserID, requests)
	if err != nil {
		utils.SendInternalError(w, fmt.Sprintf("Sync failed: %v", err))
		return
	}

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
		status := h.ignoreService.GetIgnoreStatus()
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

		err := h.ignoreService.ProcessIgnoreRequest(req.TicketID, req.Action, req.Type)
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

	// Send response with proper status code
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
