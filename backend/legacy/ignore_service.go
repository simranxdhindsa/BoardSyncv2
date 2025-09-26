package legacy

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// IgnoreService handles ignored tickets functionality
type IgnoreService struct {
	tempIgnored    map[string]bool
	foreverIgnored map[string]bool
	mutex          sync.RWMutex
	dataFile       string
}

// NewIgnoreService creates a new ignore service
func NewIgnoreService() *IgnoreService {
	service := &IgnoreService{
		tempIgnored:    make(map[string]bool),
		foreverIgnored: make(map[string]bool),
		dataFile:       "ignored_tickets.json",
	}
	
	// Load existing ignored tickets
	service.loadIgnoredTickets()
	return service
}

// IsIgnored checks if a ticket is ignored (temporarily or forever)
func (s *IgnoreService) IsIgnored(ticketID string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.tempIgnored[ticketID] || s.foreverIgnored[ticketID]
}

// IsTemporarilyIgnored checks if a ticket is temporarily ignored
func (s *IgnoreService) IsTemporarilyIgnored(ticketID string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.tempIgnored[ticketID]
}

// IsForeverIgnored checks if a ticket is permanently ignored
func (s *IgnoreService) IsForeverIgnored(ticketID string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.foreverIgnored[ticketID]
}

// AddTemporaryIgnore adds a ticket to temporary ignore list
func (s *IgnoreService) AddTemporaryIgnore(ticketID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.tempIgnored[ticketID] = true
}

// AddForeverIgnore adds a ticket to permanent ignore list
func (s *IgnoreService) AddForeverIgnore(ticketID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.foreverIgnored[ticketID] = true
	s.saveIgnoredTickets()
}

// RemoveTemporaryIgnore removes a ticket from temporary ignore list
func (s *IgnoreService) RemoveTemporaryIgnore(ticketID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.tempIgnored, ticketID)
}

// RemoveForeverIgnore removes a ticket from permanent ignore list
func (s *IgnoreService) RemoveForeverIgnore(ticketID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.foreverIgnored, ticketID)
	s.saveIgnoredTickets()
}

// GetTemporarilyIgnored returns all temporarily ignored ticket IDs
func (s *IgnoreService) GetTemporarilyIgnored() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.getMapKeys(s.tempIgnored)
}

// GetForeverIgnored returns all permanently ignored ticket IDs
func (s *IgnoreService) GetForeverIgnored() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.getMapKeys(s.foreverIgnored)
}

// GetIgnoredTickets returns all ignored ticket IDs (temp + forever)
func (s *IgnoreService) GetIgnoredTickets() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	allIgnored := make(map[string]bool)
	
	// Add temporary ignored tickets
	for ticketID := range s.tempIgnored {
		allIgnored[ticketID] = true
	}
	
	// Add forever ignored tickets
	for ticketID := range s.foreverIgnored {
		allIgnored[ticketID] = true
	}
	
	return s.getMapKeys(allIgnored)
}

// ClearTemporaryIgnores clears all temporary ignores
func (s *IgnoreService) ClearTemporaryIgnores() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.tempIgnored = make(map[string]bool)
}

// ClearForeverIgnores clears all permanent ignores
func (s *IgnoreService) ClearForeverIgnores() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.foreverIgnored = make(map[string]bool)
	s.saveIgnoredTickets()
}

// ClearAllIgnores clears all ignores (temporary and permanent)
func (s *IgnoreService) ClearAllIgnores() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.tempIgnored = make(map[string]bool)
	s.foreverIgnored = make(map[string]bool)
	s.saveIgnoredTickets()
}

// GetIgnoreStatus returns the ignore status for multiple tickets
func (s *IgnoreService) GetIgnoreStatus() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	return map[string]interface{}{
		"temp_ignored":    s.getMapKeys(s.tempIgnored),
		"forever_ignored": s.getMapKeys(s.foreverIgnored),
		"temp_count":      len(s.tempIgnored),
		"forever_count":   len(s.foreverIgnored),
		"total_ignored":   len(s.tempIgnored) + len(s.foreverIgnored),
	}
}

// ProcessIgnoreRequest processes an ignore action request
func (s *IgnoreService) ProcessIgnoreRequest(ticketID, action, ignoreType string) error {
	switch action {
	case "add":
		if ignoreType == "forever" {
			s.AddForeverIgnore(ticketID)
		} else {
			s.AddTemporaryIgnore(ticketID)
		}
	case "remove":
		if ignoreType == "forever" {
			s.RemoveForeverIgnore(ticketID)
		} else {
			s.RemoveTemporaryIgnore(ticketID)
		}
	default:
		return fmt.Errorf("invalid action: %s", action)
	}
	return nil
}

// MoveToForever moves a ticket from temporary to permanent ignore
func (s *IgnoreService) MoveToForever(ticketID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	delete(s.tempIgnored, ticketID)
	s.foreverIgnored[ticketID] = true
	s.saveIgnoredTickets()
}

// MoveToTemporary moves a ticket from permanent to temporary ignore
func (s *IgnoreService) MoveToTemporary(ticketID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	delete(s.foreverIgnored, ticketID)
	s.tempIgnored[ticketID] = true
	s.saveIgnoredTickets()
}

// loadIgnoredTickets loads ignored tickets from file
func (s *IgnoreService) loadIgnoredTickets() {
	data, err := os.ReadFile(s.dataFile)
	if err != nil {
		// File doesn't exist or can't be read, start with empty list
		return
	}

	var ignored []string
	if err := json.Unmarshal(data, &ignored); err != nil {
		// Invalid JSON, start with empty list
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	for _, id := range ignored {
		s.foreverIgnored[id] = true
	}
}

// saveIgnoredTickets saves ignored tickets to file
func (s *IgnoreService) saveIgnoredTickets() {
	ignored := s.getMapKeys(s.foreverIgnored)
	data, err := json.MarshalIndent(ignored, "", "  ")
	if err != nil {
		return
	}
	
	os.WriteFile(s.dataFile, data, 0644)
}

// getMapKeys extracts keys from a map
func (s *IgnoreService) getMapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// HasAnyIgnored checks if there are any ignored tickets
func (s *IgnoreService) HasAnyIgnored() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.tempIgnored) > 0 || len(s.foreverIgnored) > 0
}

// CountIgnored returns the total count of ignored tickets
func (s *IgnoreService) CountIgnored() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	// Use a set to avoid double counting tickets that might be in both lists
	allIgnored := make(map[string]bool)
	for ticketID := range s.tempIgnored {
		allIgnored[ticketID] = true
	}
	for ticketID := range s.foreverIgnored {
		allIgnored[ticketID] = true
	}
	
	return len(allIgnored)
}

// SetDataFile sets a custom data file path
func (s *IgnoreService) SetDataFile(filePath string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.dataFile = filePath
}