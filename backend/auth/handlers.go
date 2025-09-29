package auth

import (
	"encoding/json"
	"net/http"
	"time"

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

	// Add CORS middleware to auth routes
	auth.Use(utils.CORSMiddleware)

	// Register routes with OPTIONS support
	auth.HandleFunc("/register", h.Register).Methods("POST", "OPTIONS")
	auth.HandleFunc("/login", h.Login).Methods("POST", "OPTIONS")

	// Protected routes - create a subrouter with middleware
	protected := auth.PathPrefix("").Subrouter()
	protected.Use(h.service.Middleware)

	protected.HandleFunc("/refresh", h.RefreshToken).Methods("POST", "OPTIONS")
	protected.HandleFunc("/me", h.GetProfile).Methods("GET", "OPTIONS")
	protected.HandleFunc("/change-password", h.ChangePassword).Methods("POST", "OPTIONS")
	protected.HandleFunc("/logout", h.Logout).Methods("POST", "OPTIONS")
	
	// Account deletion endpoints
	protected.HandleFunc("/account/summary", h.GetAccountDataSummary).Methods("GET", "OPTIONS")
	protected.HandleFunc("/account/delete", h.DeleteAccount).Methods("POST", "OPTIONS")
}

// Handle OPTIONS requests for all auth endpoints
func (h *Handler) handleOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Max-Age", "86400")
	w.WriteHeader(http.StatusOK)
}

// Register handles user registration
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		h.handleOptions(w, r)
		return
	}

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
	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		h.handleOptions(w, r)
		return
	}

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
	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		h.handleOptions(w, r)
		return
	}

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
	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		h.handleOptions(w, r)
		return
	}

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
	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		h.handleOptions(w, r)
		return
	}

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
	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		h.handleOptions(w, r)
		return
	}

	// Since we're using stateless JWT tokens, logout is handled client-side
	// by removing the token from storage. We just return a success response.
	utils.SendSuccess(w, nil, "Logged out successfully")
}

// GetAccountDataSummary returns a summary of user data before deletion
func (h *Handler) GetAccountDataSummary(w http.ResponseWriter, r *http.Request) {
	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		h.handleOptions(w, r)
		return
	}

	user, ok := GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	// Log the request for audit trail
	utils.LogInfo("GET_ACCOUNT_SUMMARY", map[string]interface{}{
		"user_id":  user.UserID,
		"username": user.Username,
		"action":   "account_data_summary_requested",
	})

	summary, err := h.service.GetUserDataSummary(user.UserID)
	if err != nil {
		utils.LogError("GET_ACCOUNT_SUMMARY_ERROR", map[string]interface{}{
			"user_id": user.UserID,
			"error":   err.Error(),
		})
		utils.SendInternalError(w, "Failed to get account summary")
		return
	}

	utils.LogInfo("GET_ACCOUNT_SUMMARY_SUCCESS", map[string]interface{}{
		"user_id": user.UserID,
	})

	utils.SendSuccess(w, summary, "Account data summary retrieved successfully")
}

// DeleteAccount handles account deletion
func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		h.handleOptions(w, r)
		return
	}

	user, ok := GetUserFromContext(r)
	if !ok {
		utils.SendUnauthorized(w, "Authentication required")
		return
	}

	// Log the deletion attempt for audit trail
	utils.LogInfo("DELETE_ACCOUNT_ATTEMPT", map[string]interface{}{
		"user_id":   user.UserID,
		"username":  user.Username,
		"email":     user.Email,
		"action":    "account_deletion_requested",
		"timestamp": time.Now().Format(time.RFC3339),
	})

	var req DeleteAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.LogError("DELETE_ACCOUNT_INVALID_REQUEST", map[string]interface{}{
			"user_id": user.UserID,
			"error":   "invalid request body",
		})
		utils.SendBadRequest(w, "Invalid request body")
		return
	}

	// Validate request
	if req.Password == "" {
		utils.LogError("DELETE_ACCOUNT_NO_PASSWORD", map[string]interface{}{
			"user_id": user.UserID,
		})
		utils.SendBadRequest(w, "Password is required to delete account")
		return
	}

	if req.Confirmation != "DELETE" {
		utils.LogError("DELETE_ACCOUNT_INVALID_CONFIRMATION", map[string]interface{}{
			"user_id":      user.UserID,
			"confirmation": req.Confirmation,
		})
		utils.SendBadRequest(w, "Confirmation must be exactly 'DELETE' (case-sensitive)")
		return
	}

	// Delete the account
	err := h.service.DeleteUserAccount(user.UserID, req.Password)
	if err != nil {
		utils.LogError("DELETE_ACCOUNT_FAILED", map[string]interface{}{
			"user_id": user.UserID,
			"error":   err.Error(),
		})

		switch err {
		case ErrInvalidCredentials:
			utils.SendUnauthorized(w, "Invalid password. Please verify your password and try again.")
		case ErrUserNotFound:
			utils.SendNotFound(w, "User account not found")
		default:
			utils.SendInternalError(w, "Failed to delete account. Please try again or contact support.")
		}
		return
	}

	// Log successful deletion
	utils.LogInfo("DELETE_ACCOUNT_SUCCESS", map[string]interface{}{
		"user_id":   user.UserID,
		"username":  user.Username,
		"email":     user.Email,
		"action":    "account_permanently_deleted",
		"timestamp": time.Now().Format(time.RFC3339),
	})

	// Return success response
	response := map[string]interface{}{
		"deleted":   true,
		"user_id":   user.UserID,
		"username":  user.Username,
		"timestamp": time.Now().Format(time.RFC3339),
		"message":   "Your account and all associated data have been permanently deleted",
	}

	utils.SendSuccess(w, response, "Your account and all associated data have been permanently deleted")
}

// Health check endpoint
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	utils.SendSuccess(w, map[string]string{
		"status":  "healthy",
		"service": "authentication",
	}, "Service is healthy")
}