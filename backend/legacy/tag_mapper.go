package legacy

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	configpkg "asana-youtrack-sync/config"
)

// TagMapper handles mapping of Asana tags to YouTrack subsystems
type TagMapper struct {
	mappings      map[string]string
	mutex         sync.RWMutex
	filePath      string
	configService *configpkg.Service
	userID        int
}

// NewTagMapper creates a new tag mapper with default mappings (deprecated - use NewTagMapperForUser)
func NewTagMapper() *TagMapper {
	mapper := &TagMapper{
		mappings: make(map[string]string),
		filePath: "tag_mappings.json",
	}

	// Initialize with default mappings
	mapper.loadDefaultMappings()

	// Try to load custom mappings from file
	mapper.loadFromFile()

	return mapper
}

// NewTagMapperForUser creates a new tag mapper with user-specific mappings from database
func NewTagMapperForUser(userID int, configService *configpkg.Service) *TagMapper {
	mapper := &TagMapper{
		mappings:      make(map[string]string),
		filePath:      "tag_mappings.json",
		configService: configService,
		userID:        userID,
	}

	// Load user-specific mappings from database
	mapper.loadFromDatabase()

	return mapper
}

// NewTagMapperWithCustom creates a tag mapper with custom mappings
func NewTagMapperWithCustom(customMappings map[string]string) *TagMapper {
	mapper := &TagMapper{
		mappings: make(map[string]string),
		filePath: "tag_mappings.json",
	}
	
	// Start with default mappings
	mapper.loadDefaultMappings()
	
	// Override with custom mappings
	mapper.mutex.Lock()
	for k, v := range customMappings {
		if mapper.ValidateMapping(k, v) {
			mapper.mappings[k] = v
		}
	}
	mapper.mutex.Unlock()
	
	return mapper
}

// NewTagMapperWithFile creates a tag mapper with a specific file path
func NewTagMapperWithFile(filePath string) *TagMapper {
	mapper := &TagMapper{
		mappings: make(map[string]string),
		filePath: filePath,
	}
	
	// Initialize with default mappings
	mapper.loadDefaultMappings()
	
	// Try to load from specified file
	mapper.loadFromFile()
	
	return mapper
}

// loadDefaultMappings loads the default tag to subsystem mappings
func (tm *TagMapper) loadDefaultMappings() {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	
	for k, v := range DefaultTagMapping {
		tm.mappings[k] = v
	}
}

// MapTagToSubsystem maps an Asana tag to YouTrack subsystem
func (tm *TagMapper) MapTagToSubsystem(asanaTag string) string {
	if asanaTag == "" {
		return ""
	}
	
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	// First try exact match
	if subsystem, exists := tm.mappings[asanaTag]; exists {
		return subsystem
	}
	
	// Try case-insensitive match
	asanaTagLower := strings.ToLower(asanaTag)
	for tag, subsystem := range tm.mappings {
		if strings.ToLower(tag) == asanaTagLower {
			return subsystem
		}
	}
	
	// If no mapping found, use lowercase tag as subsystem
	return strings.ToLower(asanaTag)
}

// MapMultipleTags maps multiple tags and returns the first valid mapping
func (tm *TagMapper) MapMultipleTags(asanaTags []string) string {
	for _, tag := range asanaTags {
		if subsystem := tm.MapTagToSubsystem(tag); subsystem != "" {
			return subsystem
		}
	}
	return ""
}

// GetMappings returns all current mappings (thread-safe copy)
func (tm *TagMapper) GetMappings() map[string]string {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	// Return a copy to prevent external modification
	result := make(map[string]string)
	for k, v := range tm.mappings {
		result[k] = v
	}
	return result
}

// AddMapping adds a new tag mapping
func (tm *TagMapper) AddMapping(asanaTag, youTrackSubsystem string) error {
	if !tm.ValidateMapping(asanaTag, youTrackSubsystem) {
		return fmt.Errorf("invalid mapping: tag='%s', subsystem='%s'", asanaTag, youTrackSubsystem)
	}
	
	tm.mutex.Lock()
	tm.mappings[asanaTag] = youTrackSubsystem
	tm.mutex.Unlock()
	
	return tm.saveToFile()
}

// RemoveMapping removes a tag mapping
func (tm *TagMapper) RemoveMapping(asanaTag string) error {
	tm.mutex.Lock()
	delete(tm.mappings, asanaTag)
	tm.mutex.Unlock()
	
	return tm.saveToFile()
}

// UpdateMapping updates an existing mapping or adds a new one
func (tm *TagMapper) UpdateMapping(asanaTag, youTrackSubsystem string) error {
	if !tm.ValidateMapping(asanaTag, youTrackSubsystem) {
		return fmt.Errorf("invalid mapping: tag='%s', subsystem='%s'", asanaTag, youTrackSubsystem)
	}
	
	tm.mutex.Lock()
	tm.mappings[asanaTag] = youTrackSubsystem
	tm.mutex.Unlock()
	
	return tm.saveToFile()
}

// HasMapping checks if a mapping exists for the given tag
func (tm *TagMapper) HasMapping(asanaTag string) bool {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	_, exists := tm.mappings[asanaTag]
	return exists
}

// GetSubsystemForTag returns the subsystem for a tag, empty string if not found
func (tm *TagMapper) GetSubsystemForTag(asanaTag string) string {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	if subsystem, exists := tm.mappings[asanaTag]; exists {
		return subsystem
	}
	return ""
}

// GetTagsForSubsystem returns all tags that map to a specific subsystem
func (tm *TagMapper) GetTagsForSubsystem(subsystem string) []string {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	var tags []string
	subsystemLower := strings.ToLower(subsystem)
	
	for tag, mappedSubsystem := range tm.mappings {
		if strings.ToLower(mappedSubsystem) == subsystemLower {
			tags = append(tags, tag)
		}
	}
	
	return tags
}

// GetAllSubsystems returns all unique subsystems in the mappings
func (tm *TagMapper) GetAllSubsystems() []string {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	subsystemSet := make(map[string]bool)
	for _, subsystem := range tm.mappings {
		subsystemSet[subsystem] = true
	}
	
	var subsystems []string
	for subsystem := range subsystemSet {
		subsystems = append(subsystems, subsystem)
	}
	
	return subsystems
}

// GetAllTags returns all tags that have mappings
func (tm *TagMapper) GetAllTags() []string {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	var tags []string
	for tag := range tm.mappings {
		tags = append(tags, tag)
	}
	
	return tags
}

// ValidateMapping checks if a tag-subsystem mapping is valid
func (tm *TagMapper) ValidateMapping(asanaTag, youTrackSubsystem string) bool {
	// Basic validation - both should be non-empty and not just whitespace
	asanaTag = strings.TrimSpace(asanaTag)
	youTrackSubsystem = strings.TrimSpace(youTrackSubsystem)
	
	if asanaTag == "" || youTrackSubsystem == "" {
		return false
	}
	
	// Additional validation rules can be added here
	// For example, checking for special characters, length limits, etc.
	
	return true
}

// LoadFromMap loads mappings from a map (useful for loading from config)
func (tm *TagMapper) LoadFromMap(mappings map[string]string) error {
	validMappings := make(map[string]string)
	
	for k, v := range mappings {
		if tm.ValidateMapping(k, v) {
			validMappings[k] = v
		}
	}
	
	tm.mutex.Lock()
	tm.mappings = validMappings
	tm.mutex.Unlock()
	
	return tm.saveToFile()
}

// Export returns the current mappings for saving to config
func (tm *TagMapper) Export() map[string]string {
	return tm.GetMappings()
}

// Reset resets to default mappings
func (tm *TagMapper) Reset() error {
	tm.mutex.Lock()
	tm.mappings = make(map[string]string)
	for k, v := range DefaultTagMapping {
		tm.mappings[k] = v
	}
	tm.mutex.Unlock()
	
	return tm.saveToFile()
}

// Count returns the number of mappings
func (tm *TagMapper) Count() int {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	return len(tm.mappings)
}

// IsEmpty checks if there are no mappings
func (tm *TagMapper) IsEmpty() bool {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	return len(tm.mappings) == 0
}

// SetFilePath sets a custom file path for persistence
func (tm *TagMapper) SetFilePath(filePath string) {
	tm.mutex.Lock()
	tm.filePath = filePath
	tm.mutex.Unlock()
}

// GetFilePath returns the current file path
func (tm *TagMapper) GetFilePath() string {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	return tm.filePath
}

// SaveToFile saves the current mappings to file
func (tm *TagMapper) SaveToFile() error {
	return tm.saveToFile()
}

// LoadFromFile loads mappings from file
func (tm *TagMapper) LoadFromFile() error {
	return tm.loadFromFile()
}

// saveToFile saves mappings to the configured file path
func (tm *TagMapper) saveToFile() error {
	tm.mutex.RLock()
	mappings := make(map[string]string)
	for k, v := range tm.mappings {
		mappings[k] = v
	}
	filePath := tm.filePath
	tm.mutex.RUnlock()
	
	data, err := json.MarshalIndent(mappings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal mappings: %w", err)
	}
	
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write mappings file: %w", err)
	}
	
	return nil
}

// loadFromFile loads mappings from the configured file path
func (tm *TagMapper) loadFromFile() error {
	tm.mutex.RLock()
	filePath := tm.filePath
	tm.mutex.RUnlock()

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// File doesn't exist, which is fine - we'll use defaults
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read mappings file: %w", err)
	}

	var fileMappings map[string]string
	if err := json.Unmarshal(data, &fileMappings); err != nil {
		return fmt.Errorf("failed to unmarshal mappings: %w", err)
	}

	// Validate and load mappings
	tm.mutex.Lock()
	for k, v := range fileMappings {
		if tm.ValidateMapping(k, v) {
			tm.mappings[k] = v
		}
	}
	tm.mutex.Unlock()

	return nil
}

// loadFromDatabase loads tag mappings from user settings in database
func (tm *TagMapper) loadFromDatabase() error {
	if tm.configService == nil || tm.userID == 0 {
		// Fallback to default mappings if no config service or user ID
		tm.loadDefaultMappings()
		return nil
	}

	// Get user settings from database
	settings, err := tm.configService.GetSettings(tm.userID)
	if err != nil {
		fmt.Printf("TAG_MAPPER: Failed to load settings for user %d, using defaults: %v\n", tm.userID, err)
		tm.loadDefaultMappings()
		return nil
	}

	// Load tag mappings from CustomFieldMappings
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// Clear existing mappings
	tm.mappings = make(map[string]string)

	// Load from database if available
	if settings.CustomFieldMappings.TagMapping != nil && len(settings.CustomFieldMappings.TagMapping) > 0 {
		for asanaTag, youtrackSubsystem := range settings.CustomFieldMappings.TagMapping {
			if tm.ValidateMapping(asanaTag, youtrackSubsystem) {
				tm.mappings[asanaTag] = youtrackSubsystem
			}
		}
		fmt.Printf("TAG_MAPPER: Loaded %d tag mappings from database for user %d\n", len(tm.mappings), tm.userID)
	} else {
		// No custom mappings in database, use defaults
		for k, v := range DefaultTagMapping {
			tm.mappings[k] = v
		}
		fmt.Printf("TAG_MAPPER: No custom tag mappings found for user %d, using %d default mappings\n", tm.userID, len(tm.mappings))
	}

	return nil
}

// GetMappingStats returns statistics about the current mappings
func (tm *TagMapper) GetMappingStats() map[string]interface{} {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	subsystemCounts := make(map[string]int)
	for _, subsystem := range tm.mappings {
		subsystemCounts[subsystem]++
	}
	
	return map[string]interface{}{
		"total_mappings":    len(tm.mappings),
		"unique_subsystems": len(subsystemCounts),
		"subsystem_counts":  subsystemCounts,
		"file_path":         tm.filePath,
	}
}

// FindSimilarMappings finds mappings similar to the given tag
func (tm *TagMapper) FindSimilarMappings(tag string) []string {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	tagLower := strings.ToLower(tag)
	var similar []string
	
	for existingTag := range tm.mappings {
		existingLower := strings.ToLower(existingTag)
		
		// Check for partial matches
		if strings.Contains(existingLower, tagLower) || strings.Contains(tagLower, existingLower) {
			similar = append(similar, existingTag)
		}
	}
	
	return similar
}

// BulkUpdateMappings updates multiple mappings at once
func (tm *TagMapper) BulkUpdateMappings(mappings map[string]string) error {
	validMappings := make(map[string]string)
	
	// Validate all mappings first
	for tag, subsystem := range mappings {
		if tm.ValidateMapping(tag, subsystem) {
			validMappings[tag] = subsystem
		} else {
			return fmt.Errorf("invalid mapping: tag='%s', subsystem='%s'", tag, subsystem)
		}
	}
	
	// Update all mappings
	tm.mutex.Lock()
	for tag, subsystem := range validMappings {
		tm.mappings[tag] = subsystem
	}
	tm.mutex.Unlock()
	
	return tm.saveToFile()
}

// GetDefaultMappings returns the default tag mappings
func (tm *TagMapper) GetDefaultMappings() map[string]string {
	result := make(map[string]string)
	for k, v := range DefaultTagMapping {
		result[k] = v
	}
	return result
}

// IsDefaultMapping checks if a mapping is part of the default set
func (tm *TagMapper) IsDefaultMapping(tag string) bool {
	_, exists := DefaultTagMapping[tag]
	return exists
}

// GetCustomMappings returns only the custom (non-default) mappings
func (tm *TagMapper) GetCustomMappings() map[string]string {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	custom := make(map[string]string)
	for tag, subsystem := range tm.mappings {
		if defaultSubsystem, isDefault := DefaultTagMapping[tag]; !isDefault || defaultSubsystem != subsystem {
			custom[tag] = subsystem
		}
	}
	
	return custom
}

// SearchMappings searches for mappings containing the search term
func (tm *TagMapper) SearchMappings(searchTerm string) map[string]string {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	results := make(map[string]string)
	searchLower := strings.ToLower(searchTerm)
	
	for tag, subsystem := range tm.mappings {
		tagLower := strings.ToLower(tag)
		subsystemLower := strings.ToLower(subsystem)
		
		if strings.Contains(tagLower, searchLower) || strings.Contains(subsystemLower, searchLower) {
			results[tag] = subsystem
		}
	}
	
	return results
}

// GetMappingsBySubsystem returns all mappings grouped by subsystem
func (tm *TagMapper) GetMappingsBySubsystem() map[string][]string {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	grouped := make(map[string][]string)
	
	for tag, subsystem := range tm.mappings {
		grouped[subsystem] = append(grouped[subsystem], tag)
	}
	
	return grouped
}

// ClearCustomMappings removes all custom mappings, keeping only defaults
func (tm *TagMapper) ClearCustomMappings() error {
	tm.mutex.Lock()
	
	// Keep only default mappings
	newMappings := make(map[string]string)
	for tag, subsystem := range tm.mappings {
		if defaultSubsystem, isDefault := DefaultTagMapping[tag]; isDefault && defaultSubsystem == subsystem {
			newMappings[tag] = subsystem
		}
	}
	
	tm.mappings = newMappings
	tm.mutex.Unlock()
	
	return tm.saveToFile()
}

// ImportMappings imports mappings from a JSON file
func (tm *TagMapper) ImportMappings(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}
	
	var importMappings map[string]string
	if err := json.Unmarshal(data, &importMappings); err != nil {
		return fmt.Errorf("failed to parse import file: %w", err)
	}
	
	return tm.LoadFromMap(importMappings)
}

// ExportMappings exports current mappings to a JSON file
func (tm *TagMapper) ExportMappings(filePath string) error {
	mappings := tm.GetMappings()
	
	data, err := json.MarshalIndent(mappings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal mappings: %w", err)
	}
	
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}
	
	return nil
}

// BackupMappings creates a backup of current mappings
func (tm *TagMapper) BackupMappings() error {
	backupPath := tm.filePath + ".backup"
	return tm.ExportMappings(backupPath)
}

// RestoreFromBackup restores mappings from backup file
func (tm *TagMapper) RestoreFromBackup() error {
	backupPath := tm.filePath + ".backup"
	return tm.ImportMappings(backupPath)
}