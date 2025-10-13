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
	ticketMappings   map[int]*TicketMapping
	nextUserID       int
	nextSettingsID   int
	nextOperationID  int
	nextIgnoredID    int
	nextMappingID    int
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
		ticketMappings:  make(map[int]*TicketMapping),
		nextUserID:      1,
		nextSettingsID:  1,
		nextOperationID: 1,
		nextIgnoredID:   1,
		nextMappingID:   1,
	}

	// Load existing data
	if err := database.loadData(); err != nil {
		log.Printf("Warning: Failed to load existing data: %v\n", err)
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

	log.Printf("DB: Creating user: %s (ID: %d) with password hash length: %d\n", username, user.ID, len(passwordHash))

	// Create default settings
	settings := &UserSettings{
		ID:     db.nextSettingsID,
		UserID: user.ID,
		CustomFieldMappings: CustomFieldMappings{
			TagMapping:      make(map[string]string),
			PriorityMapping: make(map[string]string),
			StatusMapping:   make(map[string]string),
			CustomFields:    make(map[string]string),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	db.settings[settings.ID] = settings
	db.nextSettingsID++

	// Save immediately after creating user
	if err := db.saveData(); err != nil {
		log.Printf("ERROR: Failed to save user data: %v\n", err)
		return nil, err
	}

	log.Printf("DB: User created and saved successfully: %s\n", username)
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
		
		if err := db.saveData(); err != nil {
			log.Printf("ERROR: Failed to save password update: %v\n", err)
			return err
		}
		
		log.Printf("DB: Password updated successfully for user: %s\n", user.Username)
		return nil
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

func (db *DB) UpdateUserSettings(userID int, asanaPAT, youtrackBaseURL, youtrackToken, asanaProjectID, youtrackProjectID, youtrackBoardID string, mappings CustomFieldMappings) (*UserSettings, error) {
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
	settings.YouTrackBoardID = youtrackBoardID
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

	for _, ignored := range db.ignoredTickets {
		if ignored.UserID == userID && ignored.AsanaProjectID == asanaProjectID && ignored.TicketID == ticketID {
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

	return nil
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

// Ticket Mapping operations
func (db *DB) CreateTicketMapping(userID int, asanaProjectID, asanaTaskID, youtrackProjectID, youtrackIssueID string) (*TicketMapping, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	for _, mapping := range db.ticketMappings {
		if mapping.UserID == userID && 
		   mapping.AsanaTaskID == asanaTaskID && 
		   mapping.YouTrackIssueID == youtrackIssueID {
			return mapping, nil
		}
	}

	mapping := &TicketMapping{
		ID:                db.nextMappingID,
		UserID:            userID,
		AsanaProjectID:    asanaProjectID,
		AsanaTaskID:       asanaTaskID,
		YouTrackProjectID: youtrackProjectID,
		YouTrackIssueID:   youtrackIssueID,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	db.ticketMappings[mapping.ID] = mapping
	db.nextMappingID++

	if err := db.saveData(); err != nil {
		return nil, err
	}

	log.Printf("DB: Created ticket mapping: Asana %s <-> YouTrack %s for user %d\n", 
		asanaTaskID, youtrackIssueID, userID)

	return mapping, nil
}

func (db *DB) GetTicketMappingByAsanaID(userID int, asanaTaskID string) (*TicketMapping, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	for _, mapping := range db.ticketMappings {
		if mapping.UserID == userID && mapping.AsanaTaskID == asanaTaskID {
			return mapping, nil
		}
	}

	return nil, fmt.Errorf("mapping not found for Asana task %s", asanaTaskID)
}

func (db *DB) GetTicketMappingByYouTrackID(userID int, youtrackIssueID string) (*TicketMapping, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	for _, mapping := range db.ticketMappings {
		if mapping.UserID == userID && mapping.YouTrackIssueID == youtrackIssueID {
			return mapping, nil
		}
	}

	return nil, fmt.Errorf("mapping not found for YouTrack issue %s", youtrackIssueID)
}

func (db *DB) GetAllTicketMappings(userID int) ([]*TicketMapping, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	var mappings []*TicketMapping
	for _, mapping := range db.ticketMappings {
		if mapping.UserID == userID {
			mappings = append(mappings, mapping)
		}
	}

	return mappings, nil
}

func (db *DB) DeleteTicketMapping(userID, mappingID int) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	mapping, exists := db.ticketMappings[mappingID]
	if !exists {
		return fmt.Errorf("mapping not found")
	}

	if mapping.UserID != userID {
		return fmt.Errorf("access denied: mapping belongs to different user")
	}

	delete(db.ticketMappings, mappingID)

	if err := db.saveData(); err != nil {
		return err
	}

	log.Printf("DB: Deleted ticket mapping ID %d for user %d\n", mappingID, userID)
	return nil
}

func (db *DB) HasTicketMapping(userID int, asanaTaskID, youtrackIssueID string) bool {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	for _, mapping := range db.ticketMappings {
		if mapping.UserID == userID && 
		   mapping.AsanaTaskID == asanaTaskID && 
		   mapping.YouTrackIssueID == youtrackIssueID {
			return true
		}
	}

	return false
}

// DeleteUser deletes a user and all their associated data
func (db *DB) DeleteUser(userID int) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	user, exists := db.users[userID]
	if !exists {
		return fmt.Errorf("user not found")
	}

	username := user.Username
	email := user.Email

	log.Printf("DB: Starting deletion for user: %s (ID: %d, Email: %s)\n", username, userID, email)

	settingsDeleted := 0
	for id, settings := range db.settings {
		if settings.UserID == userID {
			delete(db.settings, id)
			settingsDeleted++
		}
	}
	log.Printf("DB: Deleted %d settings records for user %d\n", settingsDeleted, userID)

	operationsDeleted := 0
	for id, operation := range db.operations {
		if operation.UserID == userID {
			delete(db.operations, id)
			operationsDeleted++
		}
	}
	log.Printf("DB: Deleted %d operation records for user %d\n", operationsDeleted, userID)

	ignoredDeleted := 0
	for id, ignored := range db.ignoredTickets {
		if ignored.UserID == userID {
			delete(db.ignoredTickets, id)
			ignoredDeleted++
		}
	}
	log.Printf("DB: Deleted %d ignored ticket records for user %d\n", ignoredDeleted, userID)

	mappingsDeleted := 0
	for id, mapping := range db.ticketMappings {
		if mapping.UserID == userID {
			delete(db.ticketMappings, id)
			mappingsDeleted++
		}
	}
	log.Printf("DB: Deleted %d ticket mapping records for user %d\n", mappingsDeleted, userID)

	delete(db.users, userID)
	log.Printf("DB: Deleted user account: %s (ID: %d)\n", username, userID)

	totalDeleted := settingsDeleted + operationsDeleted + ignoredDeleted + mappingsDeleted + 1
	log.Printf("DB: Total records deleted: %d (Settings: %d, Operations: %d, Ignored Tickets: %d, Mappings: %d, User: 1)\n", 
		totalDeleted, settingsDeleted, operationsDeleted, ignoredDeleted, mappingsDeleted)

	if err := db.saveData(); err != nil {
		return fmt.Errorf("failed to save after deletion: %w", err)
	}

	log.Printf("DB: All changes saved to disk successfully for user deletion (ID: %d)\n", userID)
	return nil
}

// GetUserDataSummary returns a summary of all user data
func (db *DB) GetUserDataSummary(userID int) (map[string]int, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	if _, exists := db.users[userID]; !exists {
		return nil, fmt.Errorf("user not found")
	}

	summary := map[string]int{
		"settings":        0,
		"operations":      0,
		"ignored_tickets": 0,
		"ticket_mappings": 0,
	}

	for _, settings := range db.settings {
		if settings.UserID == userID {
			summary["settings"]++
		}
	}

	for _, operation := range db.operations {
		if operation.UserID == userID {
			summary["operations"]++
		}
	}

	for _, ignored := range db.ignoredTickets {
		if ignored.UserID == userID {
			summary["ignored_tickets"]++
		}
	}

	for _, mapping := range db.ticketMappings {
		if mapping.UserID == userID {
			summary["ticket_mappings"]++
		}
	}

	log.Printf("DB: Data summary for user %d - Settings: %d, Operations: %d, Ignored Tickets: %d, Mappings: %d\n",
		userID, summary["settings"], summary["operations"], summary["ignored_tickets"], summary["ticket_mappings"])

	return summary, nil
}

// Data persistence
func (db *DB) saveData() error {
	data := struct {
		Users           map[int]*User          `json:"users"`
		Settings        map[int]*UserSettings  `json:"settings"`
		Operations      map[int]*SyncOperation `json:"operations"`
		IgnoredTickets  map[int]*IgnoredTicket `json:"ignored_tickets"`
		TicketMappings  map[int]*TicketMapping `json:"ticket_mappings"`
		NextUserID      int                    `json:"next_user_id"`
		NextSettingsID  int                    `json:"next_settings_id"`
		NextOperationID int                    `json:"next_operation_id"`
		NextIgnoredID   int                    `json:"next_ignored_id"`
		NextMappingID   int                    `json:"next_mapping_id"`
	}{
		Users:           db.users,
		Settings:        db.settings,
		Operations:      db.operations,
		IgnoredTickets:  db.ignoredTickets,
		TicketMappings:  db.ticketMappings,
		NextUserID:      db.nextUserID,
		NextSettingsID:  db.nextSettingsID,
		NextOperationID: db.nextOperationID,
		NextIgnoredID:   db.nextIgnoredID,
		NextMappingID:   db.nextMappingID,
	}

	filePath := db.dataDir + "/data.json"
	tempPath := filePath + ".tmp"
	
	file, err := os.Create(tempPath)
	if err != nil {
		log.Printf("ERROR: Failed to create temp file: %v\n", err)
		return err
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		file.Close()
		os.Remove(tempPath)
		log.Printf("ERROR: Failed to encode data: %v\n", err)
		return err
	}

	if err := file.Close(); err != nil {
		os.Remove(tempPath)
		log.Printf("ERROR: Failed to close temp file: %v\n", err)
		return err
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath)
		log.Printf("ERROR: Failed to rename temp file: %v\n", err)
		return err
	}

	log.Printf("DB: Data saved successfully to %s\n", filePath)
	return nil
}

func (db *DB) loadData() error {
	filePath := db.dataDir + "/data.json"

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Println("DB: No existing data file found, starting fresh")
		return nil
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
		TicketMappings  map[int]*TicketMapping `json:"ticket_mappings"`
		NextUserID      int                    `json:"next_user_id"`
		NextSettingsID  int                    `json:"next_settings_id"`
		NextOperationID int                    `json:"next_operation_id"`
		NextIgnoredID   int                    `json:"next_ignored_id"`
		NextMappingID   int                    `json:"next_mapping_id"`
	}

	if err := json.NewDecoder(file).Decode(&data); err != nil {
		log.Printf("ERROR: Failed to decode data file: %v\n", err)
		return err
	}

	if data.Users != nil {
		db.users = data.Users
		log.Printf("DB: Loaded %d users\n", len(data.Users))
	}
	if data.Settings != nil {
		db.settings = data.Settings
		log.Printf("DB: Loaded %d settings\n", len(data.Settings))
	}
	if data.Operations != nil {
		db.operations = data.Operations
		log.Printf("DB: Loaded %d operations\n", len(data.Operations))
	}
	if data.IgnoredTickets != nil {
		db.ignoredTickets = data.IgnoredTickets
		log.Printf("DB: Loaded %d ignored tickets\n", len(data.IgnoredTickets))
	}
	if data.TicketMappings != nil {
		db.ticketMappings = data.TicketMappings
		log.Printf("DB: Loaded %d ticket mappings\n", len(data.TicketMappings))
	}

	db.nextUserID = data.NextUserID
	db.nextSettingsID = data.NextSettingsID
	db.nextOperationID = data.NextOperationID
	db.nextIgnoredID = data.NextIgnoredID
	db.nextMappingID = data.NextMappingID

	log.Printf("DB: Data loaded successfully from %s\n", filePath)
	return nil
}

// Close the database
func (db *DB) Close() error {
	log.Println("DB: Closing database and saving final state")
	return db.saveData()
}