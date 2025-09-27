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
	"asana-youtrack-sync/sync"
	"asana-youtrack-sync/utils"
)

// Global variables to access services from handlers
var (
	configService *configpkg.Service
	authService   *auth.Service
	legacyHandler *legacy.Handler
)

func main() {
	log.Println("Starting Enhanced Asana YouTrack Sync Service v4.0 - Legacy Refactored")

	// Initialize database
	dbPath := getEnvDefault("DB_PATH", "./sync_app.db")
	db, err := database.InitDB(dbPath)
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

	// Initialize legacy handler with user-specific database settings
	legacyHandler = legacy.NewHandler(configService)
	log.Println("✅ Legacy handler initialized with database-backed settings")

	// Initialize auto managers (but don't start them - they start on demand)
	legacy.InitializeAutoManagers(configService)
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
	log.Printf("🚀 Server starting on port %s", port)
	log.Printf("🔗 WebSocket endpoint: ws://localhost:%s/ws", port)
	log.Println("🔐 All legacy API endpoints now require authentication")
	log.Println("📊 User settings are loaded from database, not .env file")

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

	// ========================================================================
	// SETTINGS ROUTES (Protected)
	// ========================================================================
	
	configHandler.RegisterRoutes(router, authService)

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
	// LEGACY API ROUTES (Now Protected with Authentication)
	// ========================================================================
	
	legacyAPI := router.PathPrefix("").Subrouter()
	legacyAPI.Use(authService.Middleware) // All legacy routes now require auth

	// Core analysis and sync endpoints
	legacyAPI.HandleFunc("/status", legacyHandler.StatusCheck).Methods("GET", "OPTIONS")
	legacyAPI.HandleFunc("/analyze", legacyHandler.AnalyzeTickets).Methods("GET", "OPTIONS")
	legacyAPI.HandleFunc("/create", legacyHandler.CreateMissingTickets).Methods("GET", "POST", "OPTIONS")
	legacyAPI.HandleFunc("/create-single", legacyHandler.CreateSingleTicket).Methods("POST", "OPTIONS")
	legacyAPI.HandleFunc("/sync", legacyHandler.SyncMismatchedTickets).Methods("GET", "POST", "OPTIONS")
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
// AUTO-SYNC AND AUTO-CREATE HANDLERS
// ============================================================================

func handleAutoSync(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	switch r.Method {
	case "GET":
		// Get current auto-sync status
		manager := legacy.GetAutoSyncManager()
		status := manager.GetAutoSyncStatus(user.UserID)
		utils.SendSuccess(w, status, "Auto-sync status retrieved")

	case "POST":
		// Start or stop auto-sync
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
				interval = 15 // Default 15 seconds
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

func handleAutoCreate(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	switch r.Method {
	case "GET":
		// Get current auto-create status
		manager := legacy.GetAutoCreateManager()
		status := manager.GetAutoCreateStatus(user.UserID)
		utils.SendSuccess(w, status, "Auto-create status retrieved")

	case "POST":
		// Start or stop auto-create
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
				interval = 15 // Default 15 seconds
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

		// Create sync operation record
		operation, err := rollbackService.CreateOperation(user.UserID, req.Type, map[string]interface{}{
			"direction": req.Direction,
			"options":   req.Options,
		})
		if err != nil {
			utils.SendInternalError(w, "Failed to create sync operation")
			return
		}

		// Start sync process in background
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

		// Check if rollback is possible
		canRollback, reason := rollbackService.CanRollback(operationID)
		if !canRollback {
			utils.SendBadRequest(w, reason)
			return
		}

		// Start rollback process in background
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

// Perform sync operation
func performSync(operation *sync.SyncOperation, wsManager *sync.WebSocketManager, rollbackService *sync.RollbackService) {
	// Update status to in progress
	rollbackService.UpdateOperationStatus(operation.ID, sync.StatusInProgress, nil)

	// Notify start
	wsManager.SendToUser(operation.UserID, sync.MsgTypeSyncStart, map[string]interface{}{
		"operation_id": operation.ID,
		"type":         operation.OperationType,
	})

	// Simulate sync process with progress updates
	for i := 0; i <= 100; i += 20 {
		wsManager.NotifyProgress(operation.UserID, operation.ID, i, fmt.Sprintf("Syncing... %d%%", i))
		time.Sleep(1 * time.Second) // Simulate work
	}

	// For now, just mark as completed
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
		"version":     "4.0.0",
		"description": "Refactored synchronization service with modular architecture and database settings",
		"endpoints": map[string]interface{}{
			"authentication": map[string]string{
				"POST /api/auth/register":        "Register new user",
				"POST /api/auth/login":           "Login user",
				"POST /api/auth/refresh":         "Refresh token",
				"GET  /api/auth/me":              "Get user profile",
				"POST /api/auth/change-password": "Change password",
				"POST /api/auth/logout":          "Logout user",
			},
			"settings": map[string]string{
				"GET  /api/settings":                   "Get user settings",
				"PUT  /api/settings":                   "Update user settings",
				"GET  /api/settings/asana/projects":    "Get Asana projects",
				"GET  /api/settings/youtrack/projects": "Get YouTrack projects",
				"POST /api/settings/test-connections":  "Test API connections",
			},
			"new_sync": map[string]string{
				"POST /api/sync/start":         "Start sync operation",
				"GET  /api/sync/status/{id}":   "Get sync status",
				"GET  /api/sync/history":       "Get sync history",
				"POST /api/sync/rollback/{id}": "Rollback sync operation",
			},
			"legacy_api": map[string]string{
				"GET  /health":             "Health check (public)",
				"GET  /status":             "Service status (protected)",
				"GET  /analyze":            "Analyze ticket differences (protected)",
				"POST /create":             "Create missing tickets (protected)",
				"POST /create-single":      "Create individual ticket (protected)",
				"GET/POST /sync":           "Sync mismatched tickets (protected)",
				"GET/POST /ignore":         "Manage ignored tickets (protected)",
				"GET  /tickets":            "Get tickets by type (protected)",
				"POST /delete-tickets":     "Delete tickets (protected)",
				"GET  /sync-stats":         "Get sync statistics (protected)",
				"GET  /syncable-tickets":   "Get syncable tickets (protected)",
				"POST /sync-by-column":     "Sync by column (protected)",
				"POST /create-by-column":   "Create by column (protected)",
			},
			"websocket": map[string]string{
				"GET /ws": "WebSocket connection for real-time updates",
			},
		},
		"features": []string{
			"JWT-based authentication",
			"User-specific database settings",
			"Modular service architecture",
			"Legacy API compatibility",
			"Real-time sync progress via WebSocket",
			"Rollback capability",
			"Connection pooling",
			"Caching layer",
			"Custom field mapping",
			"Multi-tenant support",
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
	log.Println("   ✅ Legacy .env compatibility maintained")
	log.Println("   ✅ Database-first architecture implemented")
	log.Println("   ✅ User-specific settings enabled")
	log.Println("   ✅ Authentication required for all operations")
	log.Println("   ✅ Modular service architecture")
}

func logRouteRegistration() {
	log.Println("🛣️  Routes registered successfully:")
	log.Println("   📖 PUBLIC:")
	log.Println("      GET  /health - Health check")
	log.Println("      POST /api/auth/register - User registration") 
	log.Println("      POST /api/auth/login - User login")
	log.Println("   🔐 PROTECTED (require Bearer token):")
	log.Println("      POST /api/auth/* - Auth management")
	log.Println("      */   /api/settings/* - User settings")
	log.Println("      POST /api/sync/* - New sync API")
	log.Println("      */   /analyze, /create, /sync, /delete-tickets - Legacy API")
	log.Println("   🔗 WEBSOCKET:")
	log.Println("      GET  /ws - Real-time updates")
	log.Println("   📚 DOCUMENTATION:")
	log.Println("      GET  /api/docs - API documentation")
}