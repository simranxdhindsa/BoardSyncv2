package utils

import (
	"encoding/json"
	"fmt"
	"time"
)

// LogLevel represents the severity level of a log
type LogLevel string

const (
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
	LogLevelDebug LogLevel = "DEBUG"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Level     LogLevel               `json:"level"`
	Action    string                 `json:"action"`
	Timestamp string                 `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// LogInfo logs an info-level message
func LogInfo(action string, data map[string]interface{}) {
	log(LogLevelInfo, action, data)
}

// LogWarn logs a warning-level message
func LogWarn(action string, data map[string]interface{}) {
	log(LogLevelWarn, action, data)
}

// LogError logs an error-level message
func LogError(action string, data map[string]interface{}) {
	log(LogLevelError, action, data)
}

// LogDebug logs a debug-level message
func LogDebug(action string, data map[string]interface{}) {
	log(LogLevelDebug, action, data)
}

// log is the internal logging function
func log(level LogLevel, action string, data map[string]interface{}) {
	entry := LogEntry{
		Level:     level,
		Action:    action,
		Timestamp: time.Now().Format(time.RFC3339),
		Data:      data,
	}

	// Convert to JSON for structured logging
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		// Fallback to simple logging if JSON marshaling fails
		fmt.Printf("[%s] %s - %s: %v\n", level, entry.Timestamp, action, data)
		return
	}

	// Print structured JSON log
	fmt.Println(string(jsonBytes))
}

// LogRequest logs an HTTP request
func LogRequest(method, path, userID string) {
	LogInfo("HTTP_REQUEST", map[string]interface{}{
		"method":  method,
		"path":    path,
		"user_id": userID,
	})
}

// LogResponse logs an HTTP response
func LogResponse(statusCode int, duration time.Duration) {
	LogInfo("HTTP_RESPONSE", map[string]interface{}{
		"status_code": statusCode,
		"duration_ms": duration.Milliseconds(),
	})
}