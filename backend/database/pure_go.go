package database

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// PureGoDB is a simple file-based database for development/testing
type PureGoDB struct {
	dataDir         string
	mutex           sync.RWMutex
	users           map[int]*User
	settings        map[int]*UserSettings
	operations      map[int]*SyncOperation
	nextUserID      int
	nextSettingsID  int
	nextOperationID int
}

// NewPureGoDB creates a new pure Go database instance
func NewPureGoDB(dataDir string) (*PureGoDB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	db := &PureGoDB{
		dataDir:         dataDir,
		users:           make(map[int]*User),
		settings:        make(map[int]*UserSettings),
		operations:      make(map[int]*SyncOperation),
		nextUserID:      1,
		nextSettingsID:  1,
		nextOperationID: 1,
	}

	// Load existing data
	if err := db.loadData(); err != nil {
		return nil, err
	}

	return db, nil
}

// User operations
func (db *PureGoDB) CreateUser(username, email, passwordHash string) (*User, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	// Check if user exists
	for _, user := range db.users {
		if user.Username == username || user.Email == email {
			return nil, fmt.Errorf("user already exists")
		}
	}

	user := &User{
		ID:           db.nextUserID,
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	db.users[user.ID] = user
	db.nextUserID++

	// Create default settings
	settings := &UserSettings{
		ID:                  db.nextSettingsID,
		UserID:              user.ID,
		CustomFieldMappings: CustomFieldMappings{},
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	db.settings[settings.ID] = settings
	db.nextSettingsID++

	if err := db.saveData(); err != nil {
		return nil, err
	}

	return user, nil
}

func (db *PureGoDB) GetUserByUsername(username string) (*User, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	for _, user := range db.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (db *PureGoDB) GetUserByEmail(email string) (*User, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	for _, user := range db.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (db *PureGoDB) GetUserByID(id int) (*User, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	if user, exists := db.users[id]; exists {
		return user, nil
	}
	return nil, fmt.Errorf("user not found")
}

func (db *PureGoDB) UpdateUserPassword(userID int, passwordHash string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	if user, exists := db.users[userID]; exists {
		user.PasswordHash = passwordHash
		user.UpdatedAt = time.Now()
		return db.saveData()
	}
	return fmt.Errorf("user not found")
}

// Settings operations
func (db *PureGoDB) GetUserSettings(userID int) (*UserSettings, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	for _, settings := range db.settings {
		if settings.UserID == userID {
			return settings, nil
		}
	}
	return nil, fmt.Errorf("settings not found")
}

func (db *PureGoDB) UpdateUserSettings(userID int, asanaPAT, youtrackBaseURL, youtrackToken, asanaProjectID, youtrackProjectID string, mappings CustomFieldMappings) (*UserSettings, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	var settings *UserSettings
	for _, s := range db.settings {
		if s.UserID == userID {
			settings = s
			break
		}
	}

	if settings == nil {
		return nil, fmt.Errorf("settings not found")
	}

	settings.AsanaPAT = asanaPAT
	settings.YouTrackBaseURL = youtrackBaseURL
	settings.YouTrackToken = youtrackToken
	settings.AsanaProjectID = asanaProjectID
	settings.YouTrackProjectID = youtrackProjectID
	settings.CustomFieldMappings = mappings
	settings.UpdatedAt = time.Now()

	if err := db.saveData(); err != nil {
		return nil, err
	}

	return settings, nil
}

// Operation operations
func (db *PureGoDB) CreateOperation(userID int, operationType string, operationData map[string]interface{}) (*SyncOperation, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	operation := &SyncOperation{
		ID:            db.nextOperationID,
		UserID:        userID,
		OperationType: operationType,
		OperationData: operationData,
		Status:        "pending",
		CreatedAt:     time.Now(),
	}

	db.operations[operation.ID] = operation
	db.nextOperationID++

	if err := db.saveData(); err != nil {
		return nil, err
	}

	return operation, nil
}

func (db *PureGoDB) GetOperation(operationID int) (*SyncOperation, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	if operation, exists := db.operations[operationID]; exists {
		return operation, nil
	}
	return nil, fmt.Errorf("operation not found")
}

func (db *PureGoDB) UpdateOperationStatus(operationID int, status string, errorMessage *string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	if operation, exists := db.operations[operationID]; exists {
		operation.Status = status
		operation.ErrorMessage = errorMessage
		if status == "completed" || status == "failed" || status == "rolled_back" {
			now := time.Now()
			operation.CompletedAt = &now
		}
		return db.saveData()
	}
	return fmt.Errorf("operation not found")
}

func (db *PureGoDB) GetUserOperations(userID int, limit int) ([]*SyncOperation, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	var operations []*SyncOperation
	count := 0

	// Get operations in reverse order (newest first)
	for i := db.nextOperationID - 1; i > 0 && count < limit; i-- {
		if operation, exists := db.operations[i]; exists && operation.UserID == userID {
			operations = append(operations, operation)
			count++
		}
	}

	return operations, nil
}

// Data persistence
func (db *PureGoDB) saveData() error {
	data := struct {
		Users           map[int]*User          `json:"users"`
		Settings        map[int]*UserSettings  `json:"settings"`
		Operations      map[int]*SyncOperation `json:"operations"`
		NextUserID      int                    `json:"next_user_id"`
		NextSettingsID  int                    `json:"next_settings_id"`
		NextOperationID int                    `json:"next_operation_id"`
	}{
		Users:           db.users,
		Settings:        db.settings,
		Operations:      db.operations,
		NextUserID:      db.nextUserID,
		NextSettingsID:  db.nextSettingsID,
		NextOperationID: db.nextOperationID,
	}

	file, err := os.Create(db.dataDir + "/data.json")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func (db *PureGoDB) loadData() error {
	filePath := db.dataDir + "/data.json"

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // No data to load, start fresh
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var data struct {
		Users           map[int]*User          `json:"users"`
		Settings        map[int]*UserSettings  `json:"settings"`
		Operations      map[int]*SyncOperation `json:"operations"`
		NextUserID      int                    `json:"next_user_id"`
		NextSettingsID  int                    `json:"next_settings_id"`
		NextOperationID int                    `json:"next_operation_id"`
	}

	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return err
	}

	if data.Users != nil {
		db.users = data.Users
	}
	if data.Settings != nil {
		db.settings = data.Settings
	}
	if data.Operations != nil {
		db.operations = data.Operations
	}

	db.nextUserID = data.NextUserID
	db.nextSettingsID = data.NextSettingsID
	db.nextOperationID = data.NextOperationID

	return nil
}

// Close the database (for interface compatibility)
func (db *PureGoDB) Close() error {
	return db.saveData()
}
