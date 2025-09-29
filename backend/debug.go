// Create this file: backend/fix_password.go
// Run with: go run fix_password.go

package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/crypto/argon2"
)

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UserSettings struct {
	ID                  int                    `json:"id"`
	UserID              int                    `json:"user_id"`
	AsanaPAT            string                 `json:"asana_pat"`
	YouTrackBaseURL     string                 `json:"youtrack_base_url"`
	YouTrackToken       string                 `json:"youtrack_token"`
	AsanaProjectID      string                 `json:"asana_project_id"`
	YouTrackProjectID   string                 `json:"youtrack_project_id"`
	CustomFieldMappings map[string]interface{} `json:"custom_field_mappings"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

type DatabaseData struct {
	Users           map[string]*User         `json:"users"`
	Settings        map[string]*UserSettings `json:"settings"`
	Operations      map[string]interface{}   `json:"operations"`
	IgnoredTickets  map[string]interface{}   `json:"ignored_tickets"`
	NextUserID      int                      `json:"next_user_id"`
	NextSettingsID  int                      `json:"next_settings_id"`
	NextOperationID int                      `json:"next_operation_id"`
	NextIgnoredID   int                      `json:"next_ignored_id"`
}

func hashPassword(password string) string {
	salt := make([]byte, 16)
	rand.Read(salt)

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	// Encode salt and hash to base64
	saltEncoded := base64.StdEncoding.EncodeToString(salt)
	hashEncoded := base64.StdEncoding.EncodeToString(hash)

	return saltEncoded + "$" + hashEncoded
}

func main() {
	dbPath := "./sync_app.db_data/data.json"

	// Read existing database
	data, err := os.ReadFile(dbPath)
	if err != nil {
		log.Fatal("Failed to read database file:", err)
	}

	var dbData DatabaseData
	if err := json.Unmarshal(data, &dbData); err != nil {
		log.Fatal("Failed to parse database:", err)
	}

	// Check if user exists
	if len(dbData.Users) == 0 {
		log.Fatal("No users found in database")
	}

	// Get the first user (assuming it's the one we want to fix)
	var targetUser *User
	var userKey string
	for key, user := range dbData.Users {
		if user.Username == "simranxdhindsa" {
			targetUser = user
			userKey = key
			break
		}
	}

	if targetUser == nil {
		log.Fatal("User 'simranxdhindsa' not found")
	}

	fmt.Printf("Found user: %s (ID: %d)\n", targetUser.Username, targetUser.ID)
	fmt.Printf("Current password hash length: %d\n", len(targetUser.PasswordHash))

	// Ask for new password
	fmt.Print("\nEnter new password: ")
	var newPassword string
	fmt.Scanln(&newPassword)

	if len(newPassword) < 6 {
		log.Fatal("Password must be at least 6 characters")
	}

	// Hash the password
	passwordHash := hashPassword(newPassword)
	fmt.Printf("\nNew password hash length: %d\n", len(passwordHash))

	// Update the user
	targetUser.PasswordHash = passwordHash
	targetUser.UpdatedAt = time.Now()
	dbData.Users[userKey] = targetUser

	// Save back to file
	output, err := json.MarshalIndent(dbData, "", "  ")
	if err != nil {
		log.Fatal("Failed to marshal database:", err)
	}

	if err := os.WriteFile(dbPath, output, 0644); err != nil {
		log.Fatal("Failed to write database:", err)
	}

	fmt.Printf("\n✅ Password updated successfully for user: %s\n", targetUser.Username)
	fmt.Printf("You can now login with username: %s and your new password\n", targetUser.Username)
}