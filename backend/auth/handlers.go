package auth

import (
	"encoding/json"
	"net/http"

	"asana-youtrack-sync/utils"

	"github.com/gorilla/mux"
)

type Handler struct {
	service *Service
}

// NewHandler creates a new authentication handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// RegisterRoutes registers authentication routes
func (h *Handler) RegisterRoutes(router *mux.Router) {
	auth := router.PathPrefix("/api/auth").Subrouter()

	auth.HandleFunc("/register", h.Register).Methods("POST")
	auth.HandleFunc("/login", h.Login).Methods("POST")

	// Protected routes - create a subrouter with middleware
	protected := auth.PathPrefix("").Subrouter()
	protected.Use(h.service.Middleware)

	protected.HandleFunc("/refresh", h.RefreshToken).Methods("POST")
	protected.HandleFunc("/me", h.GetProfile).Methods("GET")
	protected.HandleFunc("/change-password", h.ChangePassword).Methods("POST")
	protected.HandleFunc("/logout", h.Logout).Methods("POST")
}

// Register handles user registration
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendBadRequest(w, "Invalid request body")
		return
	}

	// Basic validation
	if req.Username == "" || req.Email == "" || req.Password == "" {
		utils.SendBadRequest(w, "Username, email, and password are required")
		return
	}

	if len(req.Password) < 6 {
		utils.SendBadRequest(w, "Password must be at least 6 characters long")
		return
	}

	user, err := h.service.Register(req)
	if err != nil {
		switch err {
		case ErrUserExists:
			utils.SendConflict(w, "User already exists")
		default:
			utils.SendInternalError(w, "Internal server error")
		}
		return
	}

	utils.SendCreated(w, user, "User registered successfully")
}

// Login handles user authentication
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendBadRequest(w, "Invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		utils.SendBadRequest(w, "Username and password are required")
		return
	}

	response, err := h.service.Login(req)
	if err != nil {
		switch err {
		case ErrInvalidCredentials:
			utils.SendUnauthorized(w, "Invalid credentials")
		default:
			utils.SendInternalError(w, "Internal server error")
		}
		return
	}

	utils.SendSuccess(w, response, "Login successful")
}

// RefreshToken handles token refresh
func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	response, err := h.service.RefreshToken(user.UserID)
	if err != nil {
		utils.SendInternalError(w, "Internal server error")
		return
	}

	utils.SendSuccess(w, response, "Token refreshed successfully")
}

// GetProfile returns the current user's profile
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	userInfo, err := h.service.GetUser(user.UserID)
	if err != nil {
		utils.SendInternalError(w, "Internal server error")
		return
	}

	utils.SendSuccess(w, userInfo, "Profile retrieved successfully")
}

// ChangePassword handles password changes
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendBadRequest(w, "Invalid request body")
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		utils.SendBadRequest(w, "Current password and new password are required")
		return
	}

	if len(req.NewPassword) < 6 {
		utils.SendBadRequest(w, "New password must be at least 6 characters long")
		return
	}

	err := h.service.ChangePassword(user.UserID, req)
	if err != nil {
		switch err {
		case ErrInvalidCredentials:
			utils.SendBadRequest(w, "Current password is incorrect")
		case ErrUserNotFound:
			utils.SendNotFound(w, "User not found")
		default:
			utils.SendInternalError(w, "Internal server error")
		}
		return
	}

	utils.SendSuccess(w, nil, "Password changed successfully")
}

// Logout handles user logout (client-side token invalidation)
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	// Since we're using stateless JWT tokens, logout is handled client-side
	// by removing the token from storage. We just return a success response.
	utils.SendSuccess(w, nil, "Logged out successfully")
}

// Health check endpoint
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	utils.SendSuccess(w, map[string]string{
		"status":  "healthy",
		"service": "authentication",
	}, "Service is healthy")
}
