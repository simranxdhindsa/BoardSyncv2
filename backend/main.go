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
	"asana-youtrack-sync/sync"
	"asana-youtrack-sync/utils"
)

// Global variables to access services from handlers
var (
	configService *configpkg.Service
	authService   *auth.Service
)

func main() {
	// Load configuration first (for legacy endpoints)
	loadConfig()

	// Initialize ignored tickets
	loadIgnoredTickets()

	// Initialize database
	dbPath := getEnvDefault("DB_PATH", "./sync_app.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Initialize cache
	cacheManager := cache.NewCacheManager()

	// Initialize services and make them globally accessible
	jwtSecret := getEnvDefault("JWT_SECRET", "your-super-secret-jwt-key-change-in-production")
	authService = auth.NewService(db, jwtSecret)
	configService = configpkg.NewService(db)
	rollbackService := sync.NewRollbackService(db)

	// Initialize WebSocket manager
	wsManager := sync.NewWebSocketManager()
	go wsManager.Run()

	// Initialize handlers
	authHandler := auth.NewHandler(authService)
	configHandler := configpkg.NewHandler(configService)

	// Create router
	router := mux.NewRouter()

	// Register routes
	registerRoutes(router, authHandler, configHandler, authService, wsManager, rollbackService, cacheManager)

	// Start server
	port := getEnvDefault("PORT", "8080")
	log.Printf("Server starting on port %s", port)
	log.Printf("WebSocket endpoint: ws://localhost:%s/ws", port)

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

	// Health check (public)
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		utils.SendSuccess(w, map[string]string{"status": "healthy"}, "Service is running")
	}).Methods("GET")

	// PUBLIC Authentication routes (NO MIDDLEWARE)
	router.HandleFunc("/api/auth/register", authHandler.Register).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/auth/login", authHandler.Login).Methods("POST", "OPTIONS")

	// PROTECTED Authentication routes (WITH MIDDLEWARE)
	protectedAuth := router.PathPrefix("/api/auth").Subrouter()
	protectedAuth.Use(authService.Middleware)

	protectedAuth.HandleFunc("/refresh", authHandler.RefreshToken).Methods("POST", "OPTIONS")
	protectedAuth.HandleFunc("/me", authHandler.GetProfile).Methods("GET", "OPTIONS")
	protectedAuth.HandleFunc("/change-password", authHandler.ChangePassword).Methods("POST", "OPTIONS")
	protectedAuth.HandleFunc("/logout", authHandler.Logout).Methods("POST", "OPTIONS")

	// Settings routes (protected)
	configHandler.RegisterRoutes(router, authService)

	// WebSocket endpoint (with basic auth)
	router.HandleFunc("/ws", wsManager.HandleWebSocket).Methods("GET")

	// Sync routes (protected)
	syncAPI := router.PathPrefix("/api/sync").Subrouter()
	syncAPI.Use(authService.Middleware)

	syncAPI.HandleFunc("/start", handleSyncStart(wsManager, rollbackService)).Methods("POST")
	syncAPI.HandleFunc("/status/{id}", handleSyncStatus(rollbackService)).Methods("GET")
	syncAPI.HandleFunc("/history", handleSyncHistory(rollbackService)).Methods("GET")
	syncAPI.HandleFunc("/rollback/{id}", handleSyncRollback(rollbackService, wsManager)).Methods("POST")

	// LEGACY API ROUTES (now WITH authentication for user-specific settings)
	legacyAPI := router.PathPrefix("").Subrouter()
	legacyAPI.Use(authService.Middleware) // Add auth to legacy routes

	// Legacy routes (now protected)
	legacyAPI.HandleFunc("/status", statusCheck).Methods("GET")
	legacyAPI.HandleFunc("/analyze", analyzeTicketsHandler).Methods("GET", "OPTIONS")
	legacyAPI.HandleFunc("/create", createMissingTicketsHandler).Methods("GET", "POST", "OPTIONS")
	legacyAPI.HandleFunc("/create-single", createSingleTicketHandler).Methods("POST", "OPTIONS")
	legacyAPI.HandleFunc("/sync", syncMismatchedTicketsHandler).Methods("GET", "POST", "OPTIONS")
	legacyAPI.HandleFunc("/ignore", manageIgnoredTicketsHandler).Methods("GET", "POST", "OPTIONS")
	legacyAPI.HandleFunc("/auto-sync", autoSyncHandler).Methods("GET", "POST", "OPTIONS")
	legacyAPI.HandleFunc("/auto-create", autoCreateHandler).Methods("GET", "POST", "OPTIONS")
	legacyAPI.HandleFunc("/tickets", getTicketsByTypeHandler).Methods("GET", "OPTIONS")
	legacyAPI.HandleFunc("/delete-tickets", deleteTicketsHandler).Methods("POST", "OPTIONS")

	// Static file serving for frontend
	staticDir := getEnvDefault("STATIC_DIR", "./frontend/")
	router.PathPrefix("/frontend/").Handler(http.StripPrefix("/frontend/", http.FileServer(http.Dir(staticDir))))

	// Serve index.html at root
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		indexPath := staticDir + "index.html"
		http.ServeFile(w, r, indexPath)
	}).Methods("GET")

	// API documentation endpoint
	router.HandleFunc("/api/docs", handleAPIDocs).Methods("GET")
}

// Load configuration from environment variables
func loadConfig() {
	config = Config{
		Port:              getEnvDefault("PORT", "8080"),
		SyncServiceAPIKey: getEnvDefault("SYNC_SERVICE_API_KEY", ""),
		AsanaPAT:          getEnvDefault("ASANA_PAT", ""),
		AsanaProjectID:    getEnvDefault("ASANA_PROJECT_ID", ""),
		YouTrackBaseURL:   getEnvDefault("YOUTRACK_BASE_URL", ""),
		YouTrackToken:     getEnvDefault("YOUTRACK_TOKEN", ""),
		YouTrackProjectID: getEnvDefault("YOUTRACK_PROJECT_ID", ""),
		PollIntervalMS:    getEnvDefaultInt("POLL_INTERVAL_MS", 60000),
	}

	log.Printf("Configuration loaded:")
	log.Printf("  Port: %s", config.Port)
	log.Printf("  Note: Legacy .env config loaded but user-specific database settings will be used for API calls")
}

// Sync handlers (from existing code)
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

	// TODO: Implement actual sync logic here
	// This would call your existing sync functions from services.go

	// For now, just mark as completed
	rollbackService.UpdateOperationStatus(operation.ID, sync.StatusCompleted, nil)
	wsManager.NotifyComplete(operation.UserID, operation.ID, map[string]interface{}{
		"synced_items": 0,
		"message":      "Sync completed successfully",
	})
}

// API documentation handler
func handleAPIDocs(w http.ResponseWriter, r *http.Request) {
	docs := map[string]interface{}{
		"title":       "Asana YouTrack Sync API",
		"version":     "2.0.0",
		"description": "Enhanced synchronization service with authentication and user-specific settings",
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
				"GET  /health":          "Health check (public)",
				"GET  /status":          "Service status (protected)",
				"GET  /analyze":         "Analyze ticket differences (protected)",
				"POST /create":          "Create missing tickets (protected)",
				"POST /create-single":   "Create individual ticket (protected)",
				"GET/POST /sync":        "Sync mismatched tickets (protected)",
				"GET/POST /ignore":      "Manage ignored tickets (protected)",
				"GET/POST /auto-sync":   "Control auto-sync functionality (protected)",
				"GET/POST /auto-create": "Control auto-create functionality (protected)",
				"GET  /tickets":         "Get tickets by type (protected)",
				"POST /delete-tickets":  "Delete tickets (protected)",
			},
			"websocket": map[string]string{
				"GET /ws": "WebSocket connection for real-time updates",
			},
		},
		"features": []string{
			"JWT-based authentication (new API)",
			"User-specific settings from database",
			"Legacy API support with authentication",
			"Real-time sync progress via WebSocket",
			"Rollback capability",
			"Connection pooling",
			"Caching layer",
			"Custom field mapping",
			"Multi-tenant support",
		},
		"important_changes": []string{
			"All legacy API endpoints now require authentication",
			"Settings are user-specific and stored in database",
			"No longer uses global .env configuration for API calls",
		},
	}

	utils.SendSuccess(w, docs, "API documentation")
}

// Helper function to get environment variable with default
func getEnvDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Helper function to get environment variable as int with default
func getEnvDefaultInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
