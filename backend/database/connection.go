package database

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

type DB struct {
	dataDir          string
	mutex            sync.RWMutex
	users            map[int]*User
	settings         map[int]*UserSettings
	operations       map[int]*SyncOperation
	ignoredTickets   map[int]*IgnoredTicket
	nextUserID       int
	nextSettingsID   int
	nextOperationID  int
	nextIgnoredID    int
}

var database *DB

// Initialize database connection - now using pure Go implementation
func InitDB(dbPath string) (*DB, error) {
	// Use directory for pure Go database
	dataDir := dbPath + "_data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	database = &DB{
		dataDir:         dataDir,
		users:           make(map[int]*User),
		settings:        make(map[int]*UserSettings),
		operations:      make(map[int]*SyncOperation),
		ignoredTickets:  make(map[int]*IgnoredTicket),
		nextUserID:      1,
		nextSettingsID:  1,
		nextOperationID: 1,
		nextIgnoredID:   1,
	}

	// Load existing data
	if err := database.loadData(); err != nil {
		return nil, err
	}

	log.Println("Pure Go database initialized successfully")
	return database, nil
}

// Get returns the database instance
func GetDB() *DB {
	return database
}

// User operations
func (db *DB) CreateUser(username, email, passwordHash string) (*User, error) {
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

func (db *DB) GetUserByUsername(username string) (*User, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	for _, user := range db.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (db *DB) GetUserByEmail(email string) (*User, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	for _, user := range db.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (db *DB) GetUserByID(id int) (*User, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	if user, exists := db.users[id]; exists {
		return user, nil
	}
	return nil, fmt.Errorf("user not found")
}

func (db *DB) UpdateUserPassword(userID int, passwordHash string) error {
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
func (db *DB) GetUserSettings(userID int) (*UserSettings, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	for _, settings := range db.settings {
		if settings.UserID == userID {
			return settings, nil
		}
	}
	return nil, fmt.Errorf("settings not found")
}

func (db *DB) UpdateUserSettings(userID int, asanaPAT, youtrackBaseURL, youtrackToken, asanaProjectID, youtrackProjectID string, mappings CustomFieldMappings) (*UserSettings, error) {
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
func (db *DB) CreateOperation(userID int, operationType string, operationData map[string]interface{}) (*SyncOperation, error) {
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

func (db *DB) GetOperation(operationID int) (*SyncOperation, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	if operation, exists := db.operations[operationID]; exists {
		return operation, nil
	}
	return nil, fmt.Errorf("operation not found")
}

func (db *DB) UpdateOperationStatus(operationID int, status string, errorMessage *string) error {
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

func (db *DB) GetUserOperations(userID int, limit int) ([]*SyncOperation, error) {
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

// Ignored Tickets operations
func (db *DB) AddIgnoredTicket(userID int, asanaProjectID, ticketID, ignoreType string) (*IgnoredTicket, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	// Check if already exists
	for _, ignored := range db.ignoredTickets {
		if ignored.UserID == userID && ignored.AsanaProjectID == asanaProjectID && ignored.TicketID == ticketID {
			// Update ignore type if different
			if ignored.IgnoreType != ignoreType {
				ignored.IgnoreType = ignoreType
				ignored.CreatedAt = time.Now()
				if err := db.saveData(); err != nil {
					return nil, err
				}
			}
			return ignored, nil
		}
	}

	ignoredTicket := &IgnoredTicket{
		ID:             db.nextIgnoredID,
		UserID:         userID,
		AsanaProjectID: asanaProjectID,
		TicketID:       ticketID,
		IgnoreType:     ignoreType,
		CreatedAt:      time.Now(),
	}

	db.ignoredTickets[ignoredTicket.ID] = ignoredTicket
	db.nextIgnoredID++

	if err := db.saveData(); err != nil {
		return nil, err
	}

	return ignoredTicket, nil
}

func (db *DB) RemoveIgnoredTicket(userID int, asanaProjectID, ticketID, ignoreType string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	for id, ignored := range db.ignoredTickets {
		if ignored.UserID == userID && 
		   ignored.AsanaProjectID == asanaProjectID && 
		   ignored.TicketID == ticketID &&
		   (ignoreType == "" || ignored.IgnoreType == ignoreType) {
			delete(db.ignoredTickets, id)
			return db.saveData()
		}
	}

	return nil // No error if not found
}

func (db *DB) GetIgnoredTickets(userID int, asanaProjectID string) ([]*IgnoredTicket, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	var ignored []*IgnoredTicket
	for _, ticket := range db.ignoredTickets {
		if ticket.UserID == userID && ticket.AsanaProjectID == asanaProjectID {
			ignored = append(ignored, ticket)
		}
	}

	return ignored, nil
}

func (db *DB) IsTicketIgnored(userID int, asanaProjectID, ticketID string) (bool, string) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	for _, ignored := range db.ignoredTickets {
		if ignored.UserID == userID && 
		   ignored.AsanaProjectID == asanaProjectID && 
		   ignored.TicketID == ticketID {
			return true, ignored.IgnoreType
		}
	}

	return false, ""
}

func (db *DB) ClearIgnoredTickets(userID int, asanaProjectID, ignoreType string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	toDelete := []int{}
	for id, ignored := range db.ignoredTickets {
		if ignored.UserID == userID && 
		   ignored.AsanaProjectID == asanaProjectID &&
		   (ignoreType == "" || ignored.IgnoreType == ignoreType) {
			toDelete = append(toDelete, id)
		}
	}

	for _, id := range toDelete {
		delete(db.ignoredTickets, id)
	}

	if len(toDelete) > 0 {
		return db.saveData()
	}

	return nil
}

// Data persistence
func (db *DB) saveData() error {
	data := struct {
		Users           map[int]*User          `json:"users"`
		Settings        map[int]*UserSettings  `json:"settings"`
		Operations      map[int]*SyncOperation `json:"operations"`
		IgnoredTickets  map[int]*IgnoredTicket `json:"ignored_tickets"`
		NextUserID      int                    `json:"next_user_id"`
		NextSettingsID  int                    `json:"next_settings_id"`
		NextOperationID int                    `json:"next_operation_id"`
		NextIgnoredID   int                    `json:"next_ignored_id"`
	}{
		Users:           db.users,
		Settings:        db.settings,
		Operations:      db.operations,
		IgnoredTickets:  db.ignoredTickets,
		NextUserID:      db.nextUserID,
		NextSettingsID:  db.nextSettingsID,
		NextOperationID: db.nextOperationID,
		NextIgnoredID:   db.nextIgnoredID,
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

func (db *DB) loadData() error {
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
		IgnoredTickets  map[int]*IgnoredTicket `json:"ignored_tickets"`
		NextUserID      int                    `json:"next_user_id"`
		NextSettingsID  int                    `json:"next_settings_id"`
		NextOperationID int                    `json:"next_operation_id"`
		NextIgnoredID   int                    `json:"next_ignored_id"`
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
	if data.IgnoredTickets != nil {
		db.ignoredTickets = data.IgnoredTickets
	}

	db.nextUserID = data.NextUserID
	db.nextSettingsID = data.NextSettingsID
	db.nextOperationID = data.NextOperationID
	db.nextIgnoredID = data.NextIgnoredID

	return nil
}

// Close the database
func (db *DB) Close() error {
	return db.saveData()
}