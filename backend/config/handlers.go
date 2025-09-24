package config

import (
	"encoding/json"
	"net/http"

	"asana-youtrack-sync/auth"
	"asana-youtrack-sync/utils"

	"github.com/gorilla/mux"
)

type Handler struct {
	service *Service
}

// NewHandler creates a new settings handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// RegisterRoutes registers settings routes
func (h *Handler) RegisterRoutes(router *mux.Router, authService *auth.Service) {
	settings := router.PathPrefix("/api/settings").Subrouter()

	// Add CORS middleware to settings routes
	settings.Use(utils.CORSMiddleware)

	// Apply authentication middleware to all settings routes
	settings.Use(authService.Middleware)

	// Register routes with OPTIONS support
	settings.HandleFunc("", h.GetSettings).Methods("GET", "OPTIONS")
	settings.HandleFunc("", h.UpdateSettings).Methods("PUT", "OPTIONS")
	settings.HandleFunc("/asana/projects", h.GetAsanaProjects).Methods("GET", "OPTIONS")
	settings.HandleFunc("/youtrack/projects", h.GetYouTrackProjects).Methods("GET", "OPTIONS")
	settings.HandleFunc("/test-connections", h.TestConnections).Methods("POST", "OPTIONS")
}

// Handle OPTIONS requests for all settings endpoints
func (h *Handler) handleOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Max-Age", "86400")
	w.WriteHeader(http.StatusOK)
}

// GetSettings retrieves user settings
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		h.handleOptions(w, r)
		return
	}

	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	settings, err := h.service.GetSettings(user.UserID)
	if err != nil {
		utils.SendInternalError(w, "Failed to get settings")
		return
	}

	utils.SendSuccess(w, settings, "Settings retrieved successfully")
}

// UpdateSettings updates user settings
func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		h.handleOptions(w, r)
		return
	}

	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	var req UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendBadRequest(w, "Invalid request body")
		return
	}

	// Basic validation
	if req.AsanaPAT != "" && len(req.AsanaPAT) < 10 {
		utils.SendBadRequest(w, "Invalid Asana PAT")
		return
	}

	if req.YouTrackBaseURL != "" && req.YouTrackToken == "" {
		utils.SendBadRequest(w, "YouTrack token is required when URL is provided")
		return
	}

	if req.YouTrackToken != "" && req.YouTrackBaseURL == "" {
		utils.SendBadRequest(w, "YouTrack URL is required when token is provided")
		return
	}

	settings, err := h.service.UpdateSettings(user.UserID, req)
	if err != nil {
		utils.SendInternalError(w, "Failed to update settings")
		return
	}

	utils.SendSuccess(w, settings, "Settings updated successfully")
}

// GetAsanaProjects retrieves available Asana projects
func (h *Handler) GetAsanaProjects(w http.ResponseWriter, r *http.Request) {
	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		h.handleOptions(w, r)
		return
	}

	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	projects, err := h.service.GetAsanaProjects(user.UserID)
	if err != nil {
		if err.Error() == "Asana PAT not configured" {
			utils.SendBadRequest(w, "Asana PAT not configured. Please update your settings first.")
			return
		}
		utils.SendInternalError(w, "Failed to fetch Asana projects: "+err.Error())
		return
	}

	utils.SendSuccess(w, projects, "Asana projects retrieved successfully")
}

// GetYouTrackProjects retrieves available YouTrack projects
func (h *Handler) GetYouTrackProjects(w http.ResponseWriter, r *http.Request) {
	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		h.handleOptions(w, r)
		return
	}

	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	projects, err := h.service.GetYouTrackProjects(user.UserID)
	if err != nil {
		if err.Error() == "YouTrack credentials not configured" {
			utils.SendBadRequest(w, "YouTrack credentials not configured. Please update your settings first.")
			return
		}
		utils.SendInternalError(w, "Failed to fetch YouTrack projects: "+err.Error())
		return
	}

	utils.SendSuccess(w, projects, "YouTrack projects retrieved successfully")
}

// TestConnections tests API connections
func (h *Handler) TestConnections(w http.ResponseWriter, r *http.Request) {
	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		h.handleOptions(w, r)
		return
	}

	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	results, err := h.service.TestConnections(user.UserID)
	if err != nil {
		utils.SendInternalError(w, "Failed to test connections")
		return
	}

	// Determine overall status
	allConnected := true
	for _, connected := range results {
		if !connected {
			allConnected = false
			break
		}
	}

	message := "Connection test completed"
	if allConnected {
		message = "All connections successful"
	} else {
		message = "Some connections failed"
	}

	utils.SendSuccess(w, map[string]interface{}{
		"results":       results,
		"all_connected": allConnected,
	}, message)
}
