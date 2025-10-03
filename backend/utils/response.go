package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// APIResponse represents a standardized API response
type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorInfo  `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// ErrorInfo represents error information
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	APIResponse
	Pagination *PaginationInfo `json:"pagination,omitempty"`
}

// PaginationInfo represents pagination information
type PaginationInfo struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

func ExtractAsanaTaskID(url string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("empty URL provided")
	}

	// Regex pattern to extract task ID: /task/(\d+)
	re := regexp.MustCompile(`/task/(\d+)`)
	matches := re.FindStringSubmatch(url)

	if len(matches) < 2 {
		return "", fmt.Errorf("invalid Asana URL: could not extract task ID")
	}

	taskID := matches[1]
	if taskID == "" {
		return "", fmt.Errorf("extracted task ID is empty")
	}

	return taskID, nil
}

// ExtractYouTrackIssueID extracts the issue ID from a YouTrack URL
// Example: https://loop.youtrack.cloud/issue/ARD-222/Some-Issues-related-to-Audio-AI-Assistant
// Returns: ARD-222
func ExtractYouTrackIssueID(url string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("empty URL provided")
	}

	// Regex pattern to extract issue ID: /issue/([A-Z]+-\d+)
	re := regexp.MustCompile(`/issue/([A-Z]+-\d+)`)
	matches := re.FindStringSubmatch(url)

	if len(matches) < 2 {
		return "", fmt.Errorf("invalid YouTrack URL: could not extract issue ID")
	}

	issueID := matches[1]
	if issueID == "" {
		return "", fmt.Errorf("extracted issue ID is empty")
	}

	return issueID, nil
}

// ValidateAsanaURL validates if the URL is a valid Asana task URL
func ValidateAsanaURL(url string) bool {
	if url == "" {
		return false
	}
	return strings.Contains(url, "app.asana.com") && strings.Contains(url, "/task/")
}

// ValidateYouTrackURL validates if the URL is a valid YouTrack issue URL
func ValidateYouTrackURL(url string) bool {
	if url == "" {
		return false
	}
	return strings.Contains(url, "youtrack.cloud") && strings.Contains(url, "/issue/")
}

// SanitizeTitle replaces "/" with "or" for YouTrack compatibility
func SanitizeTitle(title string) string {
	return strings.ReplaceAll(title, "/", " or ")
}

// SendJSON sends a JSON response with the given status code
func SendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// SendSuccess sends a successful response
func SendSuccess(w http.ResponseWriter, data interface{}, message string) {
	response := APIResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	}
	SendJSON(w, http.StatusOK, response)
}

// SendCreated sends a created response (201)
func SendCreated(w http.ResponseWriter, data interface{}, message string) {
	response := APIResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	}
	SendJSON(w, http.StatusCreated, response)
}

// SendError sends an error response
func SendError(w http.ResponseWriter, statusCode int, code, message, details string) {
	response := APIResponse{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now(),
	}
	SendJSON(w, statusCode, response)
}

// SendBadRequest sends a bad request error (400)
func SendBadRequest(w http.ResponseWriter, message string) {
	SendError(w, http.StatusBadRequest, "BAD_REQUEST", message, "")
}

// SendUnauthorized sends an unauthorized error (401)
func SendUnauthorized(w http.ResponseWriter, message string) {
	SendError(w, http.StatusUnauthorized, "UNAUTHORIZED", message, "")
}

// SendForbidden sends a forbidden error (403)
func SendForbidden(w http.ResponseWriter, message string) {
	SendError(w, http.StatusForbidden, "FORBIDDEN", message, "")
}

// SendNotFound sends a not found error (404)
func SendNotFound(w http.ResponseWriter, message string) {
	SendError(w, http.StatusNotFound, "NOT_FOUND", message, "")
}

// SendConflict sends a conflict error (409)
func SendConflict(w http.ResponseWriter, message string) {
	SendError(w, http.StatusConflict, "CONFLICT", message, "")
}

// SendInternalError sends an internal server error (500)
func SendInternalError(w http.ResponseWriter, message string) {
	SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", message, "")
}

// SendValidationError sends a validation error with details
func SendValidationError(w http.ResponseWriter, details string) {
	SendError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", details)
}

// SendPaginated sends a paginated response
func SendPaginated(w http.ResponseWriter, data interface{}, pagination PaginationInfo, message string) {
	response := PaginatedResponse{
		APIResponse: APIResponse{
			Success:   true,
			Message:   message,
			Data:      data,
			Timestamp: time.Now(),
		},
		Pagination: &pagination,
	}
	SendJSON(w, http.StatusOK, response)
}

// CalculatePagination calculates pagination information
func CalculatePagination(page, limit, total int) PaginationInfo {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	totalPages := (total + limit - 1) / limit
	if totalPages < 1 {
		totalPages = 1
	}

	return PaginationInfo{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// HandlePanic recovers from panics and sends appropriate error response
func HandlePanic(w http.ResponseWriter, r *http.Request) {
	if err := recover(); err != nil {
		SendInternalError(w, "An unexpected error occurred")
	}
}

// CORSMiddleware adds CORS headers
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
