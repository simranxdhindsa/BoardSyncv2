package legacy

import (
	"fmt"
	"sync"
	"time"

	configpkg "asana-youtrack-sync/config"
	"asana-youtrack-sync/database"
)

// AutoSyncManager manages automatic synchronization
type AutoSyncManager struct {
	db            *database.DB
	configService *configpkg.Service
	syncService   *SyncService
	running       map[int]bool      // userID -> running status
	stopChannels  map[int]chan bool // userID -> stop channel
	intervals     map[int]int       // userID -> interval in seconds
	mutex         sync.RWMutex
	lastSync      map[int]time.Time // userID -> last sync time
	syncCount     map[int]int       // userID -> total sync count
}

// AutoCreateManager manages automatic ticket creation
type AutoCreateManager struct {
	db            *database.DB
	configService *configpkg.Service
	syncService   *SyncService
	running       map[int]bool      // userID -> running status
	stopChannels  map[int]chan bool // userID -> stop channel
	intervals     map[int]int       // userID -> interval in seconds
	mutex         sync.RWMutex
	lastCreate    map[int]time.Time // userID -> last create time
	createCount   map[int]int       // userID -> total create count
}

// Global managers
var (
	autoSyncManager   *AutoSyncManager
	autoCreateManager *AutoCreateManager
	managerOnce       sync.Once
)

// InitializeAutoManagers initializes the auto sync and create managers
func InitializeAutoManagers(db *database.DB, configService *configpkg.Service) {
	managerOnce.Do(func() {
		autoSyncManager = &AutoSyncManager{
			db:            db,
			configService: configService,
			syncService:   NewSyncService(db, configService),
			running:       make(map[int]bool),
			stopChannels:  make(map[int]chan bool),
			intervals:     make(map[int]int),
			lastSync:      make(map[int]time.Time),
			syncCount:     make(map[int]int),
		}

		autoCreateManager = &AutoCreateManager{
			db:            db,
			configService: configService,
			syncService:   NewSyncService(db, configService),
			running:       make(map[int]bool),
			stopChannels:  make(map[int]chan bool),
			intervals:     make(map[int]int),
			lastCreate:    make(map[int]time.Time),
			createCount:   make(map[int]int),
		}
	})
}

// AUTO SYNC METHODS
// =================

// StartAutoSync starts automatic synchronization for a user
func (asm *AutoSyncManager) StartAutoSync(userID int, intervalSeconds int) error {
	asm.mutex.Lock()
	defer asm.mutex.Unlock()

	// Stop existing auto-sync if running
	if asm.running[userID] {
		asm.stopAutoSyncUnsafe(userID)
	}

	// Set default interval if not provided
	if intervalSeconds <= 0 {
		intervalSeconds = 15
	}

	// Create stop channel
	stopChan := make(chan bool)
	asm.stopChannels[userID] = stopChan
	asm.intervals[userID] = intervalSeconds
	asm.running[userID] = true

	fmt.Printf("AUTO-SYNC: Starting for user %d with %d second interval\n", userID, intervalSeconds)

	// Start the auto-sync goroutine
	go asm.autoSyncLoop(userID, intervalSeconds, stopChan)

	return nil
}

// StopAutoSync stops automatic synchronization for a user
func (asm *AutoSyncManager) StopAutoSync(userID int) error {
	asm.mutex.Lock()
	defer asm.mutex.Unlock()

	return asm.stopAutoSyncUnsafe(userID)
}

// stopAutoSyncUnsafe stops auto-sync without acquiring lock (internal use)
func (asm *AutoSyncManager) stopAutoSyncUnsafe(userID int) error {
	if !asm.running[userID] {
		return fmt.Errorf("auto-sync not running for user %d", userID)
	}

	// Send stop signal
	if stopChan, exists := asm.stopChannels[userID]; exists {
		close(stopChan)
		delete(asm.stopChannels, userID)
	}

	asm.running[userID] = false
	delete(asm.intervals, userID)

	fmt.Printf("AUTO-SYNC: Stopped for user %d\n", userID)
	return nil
}

// autoSyncLoop runs the automatic synchronization loop
func (asm *AutoSyncManager) autoSyncLoop(userID int, intervalSeconds int, stopChan chan bool) {
	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	defer ticker.Stop()

	fmt.Printf("AUTO-SYNC: Loop started for user %d\n", userID)

	for {
		select {
		case <-stopChan:
			fmt.Printf("AUTO-SYNC: Loop stopped for user %d\n", userID)
			return

		case <-ticker.C:
			fmt.Printf("AUTO-SYNC: Executing sync for user %d\n", userID)

			// Perform sync operation
			err := asm.performAutoSync(userID)

			asm.mutex.Lock()
			asm.lastSync[userID] = time.Now()
			if err == nil {
				asm.syncCount[userID]++
			}
			asm.mutex.Unlock()

			if err != nil {
				fmt.Printf("AUTO-SYNC: Error for user %d: %v\n", userID, err)
			} else {
				fmt.Printf("AUTO-SYNC: Success for user %d\n", userID)
			}
		}
	}
}

// performAutoSync performs the actual sync operation
func (asm *AutoSyncManager) performAutoSync(userID int) error {
	// Perform auto-sync for mismatched tickets
	err := asm.syncService.AutoSync(userID)
	if err != nil {
		return fmt.Errorf("auto-sync failed: %w", err)
	}

	return nil
}

// GetAutoSyncStatusDetailed returns detailed auto-sync status
func (asm *AutoSyncManager) GetAutoSyncStatusDetailed(userID int) map[string]interface{} {
	asm.mutex.RLock()
	defer asm.mutex.RUnlock()

	baseStatus := asm.GetAutoSyncStatus(userID)

	// Get current mismatched tickets to show what would be synced
	result, err := asm.syncService.GetMismatchedTickets(userID)

	pendingCount := 0
	if err == nil {
		if count, ok := result["count"].(int); ok {
			pendingCount = count
		}
	}

	return map[string]interface{}{
		"running":        baseStatus.Running,
		"interval":       baseStatus.Interval,
		"last_sync":      baseStatus.LastSync,
		"next_sync":      baseStatus.NextSync,
		"sync_count":     baseStatus.SyncCount,
		"last_sync_info": baseStatus.LastSyncInfo,
		"pending_count":  pendingCount,
	}
}

// GetAutoSyncStatus returns the current status of auto-sync for a user
func (asm *AutoSyncManager) GetAutoSyncStatus(userID int) AutoSyncStatus {
	asm.mutex.RLock()
	defer asm.mutex.RUnlock()

	status := AutoSyncStatus{
		Running:      asm.running[userID],
		Interval:     asm.intervals[userID],
		SyncCount:    asm.syncCount[userID],
		LastSyncInfo: "No sync performed yet",
	}

	if lastSync, exists := asm.lastSync[userID]; exists {
		status.LastSync = lastSync
		if status.Running {
			nextSync := lastSync.Add(time.Duration(status.Interval) * time.Second)
			status.NextSync = nextSync
		}
		status.LastSyncInfo = fmt.Sprintf("Last sync: %s", lastSync.Format("2006-01-02 15:04:05"))
	}

	return status
}

// AUTO CREATE METHODS
// ===================

// StartAutoCreate starts automatic ticket creation for a user
func (acm *AutoCreateManager) StartAutoCreate(userID int, intervalSeconds int) error {
	acm.mutex.Lock()
	defer acm.mutex.Unlock()

	// Stop existing auto-create if running
	if acm.running[userID] {
		acm.stopAutoCreateUnsafe(userID)
	}

	// Set default interval if not provided
	if intervalSeconds <= 0 {
		intervalSeconds = 15
	}

	// Create stop channel
	stopChan := make(chan bool)
	acm.stopChannels[userID] = stopChan
	acm.intervals[userID] = intervalSeconds
	acm.running[userID] = true

	fmt.Printf("AUTO-CREATE: Starting for user %d with %d second interval\n", userID, intervalSeconds)

	// Start the auto-create goroutine
	go acm.autoCreateLoop(userID, intervalSeconds, stopChan)

	return nil
}

// StopAutoCreate stops automatic ticket creation for a user
func (acm *AutoCreateManager) StopAutoCreate(userID int) error {
	acm.mutex.Lock()
	defer acm.mutex.Unlock()

	return acm.stopAutoCreateUnsafe(userID)
}

// stopAutoCreateUnsafe stops auto-create without acquiring lock (internal use)
func (acm *AutoCreateManager) stopAutoCreateUnsafe(userID int) error {
	if !acm.running[userID] {
		return fmt.Errorf("auto-create not running for user %d", userID)
	}

	// Send stop signal
	if stopChan, exists := acm.stopChannels[userID]; exists {
		close(stopChan)
		delete(acm.stopChannels, userID)
	}

	acm.running[userID] = false
	delete(acm.intervals, userID)

	fmt.Printf("AUTO-CREATE: Stopped for user %d\n", userID)
	return nil
}

// autoCreateLoop runs the automatic ticket creation loop
func (acm *AutoCreateManager) autoCreateLoop(userID int, intervalSeconds int, stopChan chan bool) {
	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	defer ticker.Stop()

	fmt.Printf("AUTO-CREATE: Loop started for user %d\n", userID)

	for {
		select {
		case <-stopChan:
			fmt.Printf("AUTO-CREATE: Loop stopped for user %d\n", userID)
			return

		case <-ticker.C:
			fmt.Printf("AUTO-CREATE: Executing create for user %d\n", userID)

			// Perform create operation
			err := acm.performAutoCreate(userID)

			acm.mutex.Lock()
			acm.lastCreate[userID] = time.Now()
			if err == nil {
				acm.createCount[userID]++
			}
			acm.mutex.Unlock()

			if err != nil {
				fmt.Printf("AUTO-CREATE: Error for user %d: %v\n", userID, err)
			} else {
				fmt.Printf("AUTO-CREATE: Success for user %d\n", userID)
			}
		}
	}
}

// performAutoCreate performs the actual ticket creation operation
func (acm *AutoCreateManager) performAutoCreate(userID int) error {
	// Create missing tickets
	result, err := acm.syncService.CreateMissingTickets(userID)
	if err != nil {
		return fmt.Errorf("create operation failed: %w", err)
	}

	// Check if any tickets were created
	// The result is already a map[string]interface{}, so we can access it directly
	if created, exists := result["created"]; exists {
		if createdCount, ok := created.(int); ok && createdCount > 0 {
			fmt.Printf("AUTO-CREATE: Created %d tickets for user %d\n", createdCount, userID)
		}
	}

	return nil
}

// GetAutoCreateStatus returns the current status of auto-create for a user
func (acm *AutoCreateManager) GetAutoCreateStatus(userID int) AutoCreateStatus {
	acm.mutex.RLock()
	defer acm.mutex.RUnlock()

	status := AutoCreateStatus{
		Running:        acm.running[userID],
		Interval:       acm.intervals[userID],
		CreateCount:    acm.createCount[userID],
		LastCreateInfo: "No create performed yet",
	}

	if lastCreate, exists := acm.lastCreate[userID]; exists {
		status.LastCreate = lastCreate
		if status.Running {
			nextCreate := lastCreate.Add(time.Duration(status.Interval) * time.Second)
			status.NextCreate = nextCreate
		}
		status.LastCreateInfo = fmt.Sprintf("Last create: %s", lastCreate.Format("2006-01-02 15:04:05"))
	}

	return status
}

// GLOBAL FUNCTIONS FOR HANDLERS
// =============================

// GetAutoSyncManager returns the global auto-sync manager
func GetAutoSyncManager() *AutoSyncManager {
	return autoSyncManager
}

// GetAutoCreateManager returns the global auto-create manager
func GetAutoCreateManager() *AutoCreateManager {
	return autoCreateManager
}
