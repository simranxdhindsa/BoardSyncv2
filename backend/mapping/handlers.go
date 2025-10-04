// Create new file: backend/mapping/handlers.go
package mapping

import (
	"encoding/json"
	"net/http"
	"strconv"

	"asana-youtrack-sync/auth"
	"asana-youtrack-sync/utils"

	"github.com/gorilla/mux"
)

// Handler handles ticket mapping HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new mapping handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// RegisterRoutes registers mapping routes
func (h *Handler) RegisterRoutes(router *mux.Router, authService *auth.Service) {
	mappings := router.PathPrefix("/api/mappings").Subrouter()

	// Add CORS middleware
	mappings.Use(utils.CORSMiddleware)

	// Apply authentication middleware
	mappings.Use(authService.Middleware)

	// Register routes
	mappings.HandleFunc("", h.CreateMapping).Methods("POST", "OPTIONS")
	mappings.HandleFunc("", h.GetAllMappings).Methods("GET", "OPTIONS")
	mappings.HandleFunc("/{id}", h.DeleteMapping).Methods("DELETE", "OPTIONS")
	mappings.HandleFunc("/asana/{taskId}", h.GetByAsanaID).Methods("GET", "OPTIONS")
	mappings.HandleFunc("/youtrack/{issueId}", h.GetByYouTrackID).Methods("GET", "OPTIONS")
}

// CreateMapping handles POST /api/mappings
func (h *Handler) CreateMapping(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	var req CreateMappingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendBadRequest(w, "Invalid request body")
		return
	}


	// Validate request
	if req.AsanaURL == "" || req.YouTrackURL == "" {
		utils.SendBadRequest(w, "Both asana_url and youtrack_url are required")
		return
	}

	// Create mapping
	mapping, err := h.service.CreateMapping(user.UserID, req)
	if err != nil {
		utils.SendBadRequest(w, err.Error())
		return
	}

	utils.SendCreated(w, mapping, "Ticket mapping created successfully")
}

// GetAllMappings handles GET /api/mappings
func (h *Handler) GetAllMappings(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	mappings, err := h.service.GetAllMappings(user.UserID)
	if err != nil {
		utils.SendInternalError(w, "Failed to get mappings")
		return
	}

	utils.SendSuccess(w, mappings, "Mappings retrieved successfully")
}

// DeleteMapping handles DELETE /api/mappings/{id}
func (h *Handler) DeleteMapping(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	mappingID, err := strconv.Atoi(vars["id"])
	if err != nil {
		utils.SendBadRequest(w, "Invalid mapping ID")
		return
	}

	if err := h.service.DeleteMapping(user.UserID, mappingID); err != nil {
		utils.SendBadRequest(w, err.Error())
		return
	}

	utils.SendSuccess(w, nil, "Mapping deleted successfully")
}

// GetByAsanaID handles GET /api/mappings/asana/{taskId}
func (h *Handler) GetByAsanaID(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	taskID := vars["taskId"]

	mapping, err := h.service.GetMappingByAsanaID(user.UserID, taskID)
	if err != nil {
		utils.SendNotFound(w, "Mapping not found")
		return
	}

	utils.SendSuccess(w, mapping, "Mapping found")
}

// GetByYouTrackID handles GET /api/mappings/youtrack/{issueId}
func (h *Handler) GetByYouTrackID(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	user, ok := auth.GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	issueID := vars["issueId"]

	mapping, err := h.service.GetMappingByYouTrackID(user.UserID, issueID)
	if err != nil {
		utils.SendNotFound(w, "Mapping not found")
		return
	}

	utils.SendSuccess(w, mapping, "Mapping found")
}
