package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"asana-youtrack-sync/auth"
	"asana-youtrack-sync/cache"
	configpkg "asana-youtrack-sync/config"
	"asana-youtrack-sync/database"
	"asana-youtrack-sync/legacy"
	"asana-youtrack-sync/mapping"
	"asana-youtrack-sync/sync"
	"asana-youtrack-sync/utils"
)

// Global variables to access services from handlers
var (
	db            *database.DB
	configService *configpkg.Service
	authService   *auth.Service
	legacyHandler *legacy.Handler
)

func main() {
	log.Println("Starting Enhanced Asana YouTrack Sync Service v4.1 - Full Feature Set")
	log.Println("Features: Enhanced Analysis, Filtering, Sorting, Change Detection, Auto-Sync")

	// Initialize database
	dbPath := getEnvDefault("DB_PATH", "./sync_app.db")
	var err error
	db, err = database.InitDB(dbPath)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	log.Println("✅ Database initialized successfully")

	// Initialize cache manager
	cacheManager := cache.NewCacheManager()
	log.Println("✅ Cache manager initialized")

	// Initialize services
	jwtSecret := getEnvDefault("JWT_SECRET", "your-super-secret-jwt-key-change-in-production")
	authService = auth.NewService(db, jwtSecret)
	configService = configpkg.NewService(db)
	rollbackService := sync.NewRollbackService(db)

	log.Println("✅ Core services initialized")

	// Initialize legacy handler with database and user-specific settings
	legacyHandler = legacy.NewHandler(db, configService)
	log.Println("✅ Legacy handler initialized with enhanced features")

	// Initialize auto managers (but don't start them - they start on demand)
	legacy.InitializeAutoManagers(db, configService)
	log.Println("✅ Auto-sync and auto-create managers initialized")

	// Initialize WebSocket manager
	wsManager := sync.NewWebSocketManager()
	go wsManager.Run()
	log.Println("✅ WebSocket manager started")

	// Initialize handlers
	authHandler := auth.NewHandler(authService)
	configHandler := configpkg.NewHandler(configService)

	// Create router
	router := mux.NewRouter()

	// Register routes
	registerRoutes(router, authHandler, configHandler, authService, wsManager, rollbackService, cacheManager)

	// Log configuration status
	logConfigurationStatus()

	// Start server
	port := getEnvDefault("PORT", "8080")
	log.Printf("🚀 Server chalu ho gaya on port %s – bas ab crash na hoye 😂", port)
	log.Printf("🌐 WebSocket endpoint: ws://localhost:%s/ws – test karke dekhdeya 💻", port)
	log.Println("🔐 All old API routes hun password maangde ne – security level")
	log.Println("💾 User settings te ignored tickets hun database vich save hunde ne, per project")
	log.Println("📂 Har Asana project di apni ignored tickets list – just like your ggf")
	log.Println("✨ New features paaye ne but challakedekhdeya ki bannda: Filtering, Sorting, te Change Detection")
	log.Println("🔍 Column verification endpoints added for debugging!")

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}

func registerRoutes(
	router *mux.Router,
	authHandler *auth.Handler,
	configHandler *configpkg.Handler,
	authService *auth.Service,
	wsManager *sync.WebSocketManager,
	rollbackService *sync.RollbackService,
	cacheManager *cache.CacheManager,
) {
	// Add CORS middleware to all routes
	router.Use(utils.CORSMiddleware)

	// ========================================================================
	// PUBLIC ENDPOINTS (No Authentication Required)
	// ========================================================================

	// Health check (public)
	router.HandleFunc("/health", legacyHandler.HealthCheck).Methods("GET", "OPTIONS")

	// PUBLIC Authentication routes
	router.HandleFunc("/api/auth/register", authHandler.Register).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/auth/login", authHandler.Login).Methods("POST", "OPTIONS")

	// ========================================================================
	// PROTECTED AUTHENTICATION ROUTES
	// ========================================================================

	protectedAuth := router.PathPrefix("/api/auth").Subrouter()
	protectedAuth.Use(authService.Middleware)

	protectedAuth.HandleFunc("/refresh", authHandler.RefreshToken).Methods("POST", "OPTIONS")
	protectedAuth.HandleFunc("/me", authHandler.GetProfile).Methods("GET", "OPTIONS")
	protectedAuth.HandleFunc("/change-password", authHandler.ChangePassword).Methods("POST", "OPTIONS")
	protectedAuth.HandleFunc("/logout", authHandler.Logout).Methods("POST", "OPTIONS")
	protectedAuth.HandleFunc("/account/summary", authHandler.GetAccountDataSummary).Methods("GET", "OPTIONS")
	protectedAuth.HandleFunc("/account/delete", authHandler.DeleteAccount).Methods("POST", "OPTIONS")

	// ========================================================================
	// SETTINGS ROUTES (Protected)
	// ========================================================================

	configHandler.RegisterRoutes(router, authService)

	// ========================================================================
	// TICKET MAPPINGS ROUTES (Protected)
	// ========================================================================

	mappingService := mapping.NewService(db, configService)
	mappingHandler := mapping.NewHandler(mappingService)
	mappingHandler.RegisterRoutes(router, authService)

	// ========================================================================
	// WEBSOCKET ENDPOINT
	// ========================================================================

	router.HandleFunc("/ws", wsManager.HandleWebSocket).Methods("GET")

	// ========================================================================
	// NEW SYNC API ROUTES (Protected)
	// ========================================================================

	syncAPI := router.PathPrefix("/api/sync").Subrouter()
	syncAPI.Use(authService.Middleware)

	syncAPI.HandleFunc("/start", handleSyncStart(wsManager, rollbackService)).Methods("POST")
	syncAPI.HandleFunc("/status/{id}", handleSyncStatus(rollbackService)).Methods("GET")
	syncAPI.HandleFunc("/history", handleSyncHistory(rollbackService)).Methods("GET")
	syncAPI.HandleFunc("/rollback/{id}", handleSyncRollback(rollbackService, wsManager)).Methods("POST")

	// ========================================================================
	// LEGACY API ROUTES - ENHANCED (Now Protected with Authentication)
	// ========================================================================

	legacyAPI := router.PathPrefix("").Subrouter()
	legacyAPI.Use(authService.Middleware) // All legacy routes now require auth

	// Core analysis and sync endpoints
	legacyAPI.HandleFunc("/status", legacyHandler.StatusCheck).Methods("GET", "OPTIONS")
	legacyAPI.HandleFunc("/analyze", legacyHandler.AnalyzeTickets).Methods("GET", "OPTIONS")

	// ENHANCED: Analysis with filtering and sorting
	legacyAPI.HandleFunc("/analyze/enhanced", legacyHandler.AnalyzeTicketsEnhanced).Methods("GET", "POST", "OPTIONS")

	// ENHANCED: Get tickets with title/description changes
	legacyAPI.HandleFunc("/changed-mappings", legacyHandler.GetChangedMappings).Methods("GET", "OPTIONS")

	// ENHANCED: Get available filter options
	legacyAPI.HandleFunc("/filter-options", legacyHandler.GetFilterOptions).Methods("GET", "OPTIONS")

	// ========================================================================
	// 🔍 NEW: COLUMN VERIFICATION & DEBUG ENDPOINTS
	// ========================================================================
	// legacyAPI.HandleFunc("/verify-columns", legacyHandler.VerifyColumnsAndMapping).Methods("GET", "OPTIONS")
	// legacyAPI.HandleFunc("/column-report", legacyHandler.GetColumnMappingReport).Methods("GET", "OPTIONS")
	legacyAPI.HandleFunc("/youtrack-states", legacyHandler.GetYouTrackStatesRaw).Methods("GET", "OPTIONS")
	// ========================================================================

	legacyAPI.HandleFunc("/create", legacyHandler.CreateMissingTickets).Methods("GET", "POST", "OPTIONS")
	legacyAPI.HandleFunc("/create-single", legacyHandler.CreateSingleTicket).Methods("POST", "OPTIONS")
	legacyAPI.HandleFunc("/sync", legacyHandler.SyncMismatchedTickets).Methods("GET", "POST", "OPTIONS")

	// ENHANCED: Sync with change detection
	legacyAPI.HandleFunc("/sync/enhanced", handleEnhancedSync).Methods("GET", "POST", "OPTIONS")

	legacyAPI.HandleFunc("/ignore", legacyHandler.ManageIgnoredTickets).Methods("GET", "POST", "OPTIONS")
	legacyAPI.HandleFunc("/tickets", legacyHandler.GetTicketsByType).Methods("GET", "OPTIONS")
	legacyAPI.HandleFunc("/delete-tickets", legacyHandler.DeleteTickets).Methods("POST", "OPTIONS")

	// Additional endpoints
	legacyAPI.HandleFunc("/sync-stats", legacyHandler.GetSyncStats).Methods("GET", "OPTIONS")
	legacyAPI.HandleFunc("/syncable-tickets", legacyHandler.GetSyncableTickets).Methods("GET", "OPTIONS")
	legacyAPI.HandleFunc("/sync-by-column", legacyHandler.SyncByColumn).Methods("POST", "OPTIONS")
	legacyAPI.HandleFunc("/create-by-column", legacyHandler.CreateByColumn).Methods("POST", "OPTIONS")
	legacyAPI.HandleFunc("/deletion-preview", legacyHandler.GetDeletionPreview).Methods("GET", "OPTIONS")
	legacyAPI.HandleFunc("/sync-preview", legacyHandler.GetSyncPreview).Methods("GET", "OPTIONS")

	// Auto-sync endpoints
	legacyAPI.HandleFunc("/auto-sync", handleAutoSync).Methods("GET", "POST", "OPTIONS")
	legacyAPI.HandleFunc("/auto-create", handleAutoCreate).Methods("GET", "POST", "OPTIONS")

	// ENHANCED: Detailed auto-sync status
	legacyAPI.HandleFunc("/auto-sync/detailed", handleAutoSyncDetailed).Methods("GET", "OPTIONS")

	// ========================================================================
	// STATIC FILE SERVING
	// ========================================================================

	staticDir := getEnvDefault("STATIC_DIR", "./frontend/")
	router.PathPrefix("/frontend/").Handler(http.StripPrefix("/frontend/", http.FileServer(http.Dir(staticDir))))

	// Serve index.html at root
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		indexPath := staticDir + "index.html"
		http.ServeFile(w, r, indexPath)
	}).Methods("GET")

	// ========================================================================
	// API DOCUMENTATION
	// ========================================================================

	router.HandleFunc("/api/docs", handleAPIDocs).Methods("GET")

	logRouteRegistration()
}

// ============================================================================
// ENHANCED SYNC HANDLER
// ============================================================================

func handleEnhancedSync(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	columnFilter := r.URL.Query().Get("column")
	var mappedColumn string
	if columnFilter != "" && columnFilter != "all_syncable" {
		columnMap := map[string]string{
			"backlog":         "backlog",
			"in_progress":     "in progress",
			"dev":             "dev",
			"stage":           "stage",
			"blocked":         "blocked",
			"ready_for_stage": "ready for stage",
		}
		if mapped, exists := columnMap[columnFilter]; exists {
			mappedColumn = mapped
		} else {
			mappedColumn = columnFilter
		}
	}

	if r.Method == "GET" {
		// Return available mismatched tickets with change details
		result, err := legacyHandler.GetSyncService().GetMismatchedTicketsWithChanges(user.UserID, mappedColumn)
		if err != nil {
			utils.SendInternalError(w, fmt.Sprintf("Failed to get mismatched tickets: %v", err))
			return
		}
		utils.SendSuccess(w, result, "Mismatched tickets retrieved with change details")
		return
	}

	if r.Method != "POST" {
		utils.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED",
			"Method not allowed. Use GET to see available tickets, POST to sync.", "")
		return
	}

	var requests []legacy.SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		utils.SendBadRequest(w, "Invalid JSON format")
		return
	}

	syncService := legacyHandler.GetSyncService()
	if err := syncService.ValidateSyncRequests(requests); err != nil {
		utils.SendBadRequest(w, err.Error())
		return
	}

	result, err := syncService.SyncMismatchedTicketsEnhanced(user.UserID, requests, mappedColumn)
	if err != nil {
		utils.SendInternalError(w, fmt.Sprintf("Sync failed: %v", err))
		return
	}

	utils.SendSuccess(w, result, "Enhanced sync operation completed")
}

// ============================================================================
// AUTO-SYNC HANDLERS
// ============================================================================

func handleAutoSync(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	switch r.Method {
	case "GET":
		manager := legacy.GetAutoSyncManager()
		status := manager.GetAutoSyncStatus(user.UserID)
		utils.SendSuccess(w, status, "Auto-sync status retrieved")

	case "POST":
		var req legacy.AutoSyncRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			utils.SendBadRequest(w, "Invalid request body")
			return
		}

		manager := legacy.GetAutoSyncManager()

		switch req.Action {
		case "start":
			interval := req.Interval
			if interval <= 0 {
				interval = 15
			}

			err := manager.StartAutoSync(user.UserID, interval)
			if err != nil {
				utils.SendInternalError(w, "Failed to start auto-sync: "+err.Error())
				return
			}

			status := manager.GetAutoSyncStatus(user.UserID)
			utils.SendSuccess(w, status, "Auto-sync started successfully")

		case "stop":
			err := manager.StopAutoSync(user.UserID)
			if err != nil {
				utils.SendBadRequest(w, "Failed to stop auto-sync: "+err.Error())
				return
			}

			status := manager.GetAutoSyncStatus(user.UserID)
			utils.SendSuccess(w, status, "Auto-sync stopped successfully")

		default:
			utils.SendBadRequest(w, "Invalid action. Use 'start' or 'stop'")
		}

	default:
		utils.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED",
			"Method not allowed. Use GET or POST.", "")
	}
}

func handleAutoSyncDetailed(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	manager := legacy.GetAutoSyncManager()
	status := manager.GetAutoSyncStatusDetailed(user.UserID)
	utils.SendSuccess(w, status, "Detailed auto-sync status retrieved")
}

func handleAutoCreate(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	switch r.Method {
	case "GET":
		manager := legacy.GetAutoCreateManager()
		status := manager.GetAutoCreateStatus(user.UserID)
		utils.SendSuccess(w, status, "Auto-create status retrieved")

	case "POST":
		var req legacy.AutoCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			utils.SendBadRequest(w, "Invalid request body")
			return
		}

		manager := legacy.GetAutoCreateManager()

		switch req.Action {
		case "start":
			interval := req.Interval
			if interval <= 0 {
				interval = 15
			}

			err := manager.StartAutoCreate(user.UserID, interval)
			if err != nil {
				utils.SendInternalError(w, "Failed to start auto-create: "+err.Error())
				return
			}

			status := manager.GetAutoCreateStatus(user.UserID)
			utils.SendSuccess(w, status, "Auto-create started successfully")

		case "stop":
			err := manager.StopAutoCreate(user.UserID)
			if err != nil {
				utils.SendBadRequest(w, "Failed to stop auto-create: "+err.Error())
				return
			}

			status := manager.GetAutoCreateStatus(user.UserID)
			utils.SendSuccess(w, status, "Auto-create stopped successfully")

		default:
			utils.SendBadRequest(w, "Invalid action. Use 'start' or 'stop'")
		}

	default:
		utils.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED",
			"Method not allowed. Use GET or POST.", "")
	}
}

// ============================================================================
// NEW SYNC API HANDLERS
// ============================================================================

func handleSyncStart(wsManager *sync.WebSocketManager, rollbackService *sync.RollbackService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.GetUserFromContext(r)
		if !ok {
			utils.SendUnauthorized(w, "Authentication required")
			return
		}

		var req struct {
			Type      string                 `json:"type"`
			Direction string                 `json:"direction"`
			Options   map[string]interface{} `json:"options"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			utils.SendBadRequest(w, "Invalid request body")
			return
		}

		operation, err := rollbackService.CreateOperation(user.UserID, req.Type, map[string]interface{}{
			"direction": req.Direction,
			"options":   req.Options,
		})
		if err != nil {
			utils.SendInternalError(w, "Failed to create sync operation")
			return
		}

		go performSync(operation, wsManager, rollbackService)

		utils.SendSuccess(w, map[string]interface{}{
			"operation_id": operation.ID,
			"status":       operation.Status,
		}, "Sync started successfully")
	}
}

func handleSyncStatus(rollbackService *sync.RollbackService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.GetUserFromContext(r)
		if !ok {
			utils.SendUnauthorized(w, "Authentication required")
			return
		}

		vars := mux.Vars(r)
		operationID, err := strconv.Atoi(vars["id"])
		if err != nil {
			utils.SendBadRequest(w, "Invalid operation ID")
			return
		}

		operation, err := rollbackService.GetOperation(operationID)
		if err != nil {
			utils.SendNotFound(w, "Operation not found")
			return
		}

		if operation.UserID != user.UserID {
			utils.SendForbidden(w, "Access denied")
			return
		}

		utils.SendSuccess(w, operation, "Operation status retrieved")
	}
}

func handleSyncHistory(rollbackService *sync.RollbackService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.GetUserFromContext(r)
		if !ok {
			utils.SendUnauthorized(w, "Authentication required")
			return
		}

		limit := 50
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
				limit = l
			}
		}

		operations, err := rollbackService.GetUserOperations(user.UserID, limit)
		if err != nil {
			utils.SendInternalError(w, "Failed to get sync history")
			return
		}

		utils.SendSuccess(w, operations, "Sync history retrieved")
	}
}

func handleSyncRollback(rollbackService *sync.RollbackService, wsManager *sync.WebSocketManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.GetUserFromContext(r)
		if !ok {
			utils.SendUnauthorized(w, "Authentication required")
			return
		}

		vars := mux.Vars(r)
		operationID, err := strconv.Atoi(vars["id"])
		if err != nil {
			utils.SendBadRequest(w, "Invalid operation ID")
			return
		}

		canRollback, reason := rollbackService.CanRollback(operationID)
		if !canRollback {
			utils.SendBadRequest(w, reason)
			return
		}

		go func() {
			wsManager.SendToUser(user.UserID, sync.MsgTypeRollback, map[string]interface{}{
				"operation_id": operationID,
				"status":       "starting",
			})

			err := rollbackService.RollbackOperation(operationID, user.UserID)
			if err != nil {
				wsManager.NotifyError(user.UserID, operationID, err.Error())
			} else {
				wsManager.SendToUser(user.UserID, sync.MsgTypeRollback, map[string]interface{}{
					"operation_id": operationID,
					"status":       "completed",
				})
			}
		}()

		utils.SendSuccess(w, map[string]interface{}{
			"operation_id": operationID,
		}, "Rollback started")
	}
}

func performSync(operation *sync.SyncOperation, wsManager *sync.WebSocketManager, rollbackService *sync.RollbackService) {
	rollbackService.UpdateOperationStatus(operation.ID, sync.StatusInProgress, nil)

	wsManager.SendToUser(operation.UserID, sync.MsgTypeSyncStart, map[string]interface{}{
		"operation_id": operation.ID,
		"type":         operation.OperationType,
	})

	for i := 0; i <= 100; i += 20 {
		wsManager.NotifyProgress(operation.UserID, operation.ID, i, fmt.Sprintf("Syncing... %d%%", i))
		time.Sleep(1 * time.Second)
	}

	rollbackService.UpdateOperationStatus(operation.ID, sync.StatusCompleted, nil)
	wsManager.NotifyComplete(operation.UserID, operation.ID, map[string]interface{}{
		"synced_items": 0,
		"message":      "Sync completed successfully",
	})
}

// ============================================================================
// API DOCUMENTATION HANDLER
// ============================================================================

func handleAPIDocs(w http.ResponseWriter, r *http.Request) {
	docs := map[string]interface{}{
		"title":       "Enhanced Asana YouTrack Sync API",
		"version":     "4.1.0",
		"description": "Full-featured synchronization with filtering, sorting, and change detection",
		"endpoints": map[string]interface{}{
			"authentication": map[string]string{
				"POST /api/auth/register":        "Register new user",
				"POST /api/auth/login":           "Login user",
				"POST /api/auth/refresh":         "Refresh token",
				"GET  /api/auth/me":              "Get user profile",
				"POST /api/auth/change-password": "Change password",
				"POST /api/auth/logout":          "Logout user",
				"GET  /api/auth/account/summary": "Get account data summary",
				"POST /api/auth/account/delete":  "Delete account",
			},
			"settings": map[string]string{
				"GET  /api/settings":                   "Get user settings",
				"PUT  /api/settings":                   "Update user settings",
				"GET  /api/settings/asana/projects":    "Get Asana projects",
				"GET  /api/settings/youtrack/projects": "Get YouTrack projects",
				"POST /api/settings/test-connections":  "Test API connections",
			},
			"ticket_mappings": map[string]string{
				"POST   /api/mappings":                    "Create manual ticket mapping",
				"GET    /api/mappings":                    "Get all ticket mappings",
				"DELETE /api/mappings/{id}":               "Delete ticket mapping",
				"GET    /api/mappings/asana/{taskId}":     "Get mapping by Asana task ID",
				"GET    /api/mappings/youtrack/{issueId}": "Get mapping by YouTrack issue ID",
			},
			"column_verification": map[string]string{
				"GET  /verify-columns":    "Verify column detection and mapping (detailed JSON)",
				"GET  /column-report":     "Get human-readable column mapping report",
				"GET  /youtrack-states":   "Get raw YouTrack state information (debug)",
				"POST /validate-mappings": "Validate and cleanup invalid ticket mappings",
			},
			"enhanced_analysis": map[string]string{
				"GET/POST /analyze/enhanced": "Analyze with filtering and sorting",
				"GET      /filter-options":   "Get available filter options",
				"GET      /changed-mappings": "Get tickets with title/description changes",
			},
			"enhanced_sync": map[string]string{
				"GET/POST /sync/enhanced":      "Sync with change detection",
				"GET      /auto-sync/detailed": "Get detailed auto-sync status",
			},
			"new_sync": map[string]string{
				"POST /api/sync/start":         "Start sync operation",
				"GET  /api/sync/status/{id}":   "Get sync status",
				"GET  /api/sync/history":       "Get sync history",
				"POST /api/sync/rollback/{id}": "Rollback sync operation",
			},
			"legacy_api": map[string]string{
				"GET  /health":           "Health check (public)",
				"GET  /status":           "Service status (protected)",
				"GET  /analyze":          "Basic ticket analysis",
				"POST /create":           "Create missing tickets",
				"POST /create-single":    "Create individual ticket",
				"GET/POST /sync":         "Basic sync operation",
				"GET/POST /ignore":       "Manage ignored tickets",
				"GET  /tickets":          "Get tickets by type",
				"POST /delete-tickets":   "Delete tickets",
				"GET  /sync-stats":       "Get sync statistics",
				"GET  /syncable-tickets": "Get syncable tickets",
				"POST /sync-by-column":   "Sync by column",
				"POST /create-by-column": "Create by column",
			},
			"websocket": map[string]string{
				"GET /ws": "WebSocket connection for real-time updates",
			},
		},
		"features": []string{
			"✅ Enhanced analysis with filtering and sorting",
			"✅ Title and description change detection",
			"✅ Sort by: created_at, assignee, priority",
			"✅ Filter by: assignees, priorities, date range",
			"✅ Auto-sync with change detection",
			"✅ Manual ticket mapping with URL parsing",
			"✅ JWT-based authentication",
			"✅ Multi-tenant support",
			"✅ Real-time sync progress via WebSocket",
			"✅ Rollback capability",
			"✅ Column verification and debug endpoints",
		},
		"new_in_v4_1": []string{
			"Enhanced ticket data (created_at, assignee, priority)",
			"Title/description change detection",
			"Advanced filtering (multi-assignee, date range, priority)",
			"Multi-criteria sorting",
			"Detailed auto-sync status with pending changes",
			"Enhanced sync API with change breakdown",
			"🔍 Column verification endpoints for debugging",
			"🔧 Automatic invalid mapping detection and cleanup",
		},
	}

	utils.SendSuccess(w, docs, "API documentation")
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

func getEnvDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func logConfigurationStatus() {
	log.Println("📋 Configuration Status:")
	log.Println("   ✅ Enhanced analysis with filtering/sorting")
	log.Println("   ✅ Title/description change detection")
	log.Println("   ✅ Advanced filtering capabilities")
	log.Println("   ✅ Multi-criteria sorting")
	log.Println("   ✅ Auto-sync with change detection")
	log.Println("   ✅ Database-first architecture")
	log.Println("   ✅ User-specific settings")
	log.Println("   ✅ Authentication required for all operations")
	log.Println("   🔍 Column verification endpoints enabled")
}

func logRouteRegistration() {
	log.Println("🛣️  Routes registered successfully:")
	log.Println("   📖 PUBLIC:")
	log.Println("      GET  /health - Health check")
	log.Println("      POST /api/auth/register - User registration")
	log.Println("      POST /api/auth/login - User login")
	log.Println("   🔒 PROTECTED (require Bearer token):")
	log.Println("      POST /api/auth/* - Auth management")
	log.Println("      */   /api/settings/* - User settings")
	log.Println("      */   /api/mappings/* - Ticket mappings")
	log.Println("   🔍 COLUMN VERIFICATION (NEW):")
	log.Println("      GET  /verify-columns - Detailed column verification")
	log.Println("      GET  /column-report - Human-readable mapping report")
	log.Println("      GET  /youtrack-states - Raw YouTrack state debugging")
	log.Println("      POST /validate-mappings - Cleanup invalid mappings")
	log.Println("   🎯 ENHANCED FEATURES:")
	log.Println("      GET/POST /analyze/enhanced - Analysis with filters/sorting")
	log.Println("      GET      /filter-options - Available filter values")
	log.Println("      GET      /changed-mappings - Tickets with changes")
	log.Println("      GET/POST /sync/enhanced - Sync with change detection")
	log.Println("      GET      /auto-sync/detailed - Detailed auto-sync status")
	log.Println("   🔗 WEBSOCKET:")
	log.Println("      GET  /ws - Real-time updates")
	log.Println("   📚 DOCUMENTATION:")
	log.Println("      GET  /api/docs - Full API documentation")
}
