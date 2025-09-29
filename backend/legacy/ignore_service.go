package legacy

import (
	"fmt"
	"asana-youtrack-sync/database"
	configpkg "asana-youtrack-sync/config"
)

// IgnoreService handles ignored tickets functionality with database storage
type IgnoreService struct {
	db            *database.DB
	configService *configpkg.Service
}

// NewIgnoreService creates a new ignore service
func NewIgnoreService(db *database.DB, configService *configpkg.Service) *IgnoreService {
	return &IgnoreService{
		db:            db,
		configService: configService,
	}
}

// IsIgnored checks if a ticket is ignored (temporarily or forever) for a user's current project
func (s *IgnoreService) IsIgnored(userID int, ticketID string) bool {
	settings, err := s.configService.GetSettings(userID)
	if err != nil || settings.AsanaProjectID == "" {
		return false
	}

	isIgnored, _ := s.db.IsTicketIgnored(userID, settings.AsanaProjectID, ticketID)
	return isIgnored
}

// IsTemporarilyIgnored checks if a ticket is temporarily ignored
func (s *IgnoreService) IsTemporarilyIgnored(userID int, ticketID string) bool {
	settings, err := s.configService.GetSettings(userID)
	if err != nil || settings.AsanaProjectID == "" {
		return false
	}

	isIgnored, ignoreType := s.db.IsTicketIgnored(userID, settings.AsanaProjectID, ticketID)
	return isIgnored && ignoreType == "temp"
}

// IsForeverIgnored checks if a ticket is permanently ignored
func (s *IgnoreService) IsForeverIgnored(userID int, ticketID string) bool {
	settings, err := s.configService.GetSettings(userID)
	if err != nil || settings.AsanaProjectID == "" {
		return false
	}

	isIgnored, ignoreType := s.db.IsTicketIgnored(userID, settings.AsanaProjectID, ticketID)
	return isIgnored && ignoreType == "forever"
}

// AddTemporaryIgnore adds a ticket to temporary ignore list
func (s *IgnoreService) AddTemporaryIgnore(userID int, ticketID string) error {
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return fmt.Errorf("failed to get user settings: %w", err)
	}

	if settings.AsanaProjectID == "" {
		return fmt.Errorf("no Asana project configured")
	}

	_, err = s.db.AddIgnoredTicket(userID, settings.AsanaProjectID, ticketID, "temp")
	return err
}

// AddForeverIgnore adds a ticket to permanent ignore list
func (s *IgnoreService) AddForeverIgnore(userID int, ticketID string) error {
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return fmt.Errorf("failed to get user settings: %w", err)
	}

	if settings.AsanaProjectID == "" {
		return fmt.Errorf("no Asana project configured")
	}

	_, err = s.db.AddIgnoredTicket(userID, settings.AsanaProjectID, ticketID, "forever")
	return err
}

// RemoveTemporaryIgnore removes a ticket from temporary ignore list
func (s *IgnoreService) RemoveTemporaryIgnore(userID int, ticketID string) error {
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return fmt.Errorf("failed to get user settings: %w", err)
	}

	if settings.AsanaProjectID == "" {
		return fmt.Errorf("no Asana project configured")
	}

	return s.db.RemoveIgnoredTicket(userID, settings.AsanaProjectID, ticketID, "temp")
}

// RemoveForeverIgnore removes a ticket from permanent ignore list
func (s *IgnoreService) RemoveForeverIgnore(userID int, ticketID string) error {
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return fmt.Errorf("failed to get user settings: %w", err)
	}

	if settings.AsanaProjectID == "" {
		return fmt.Errorf("no Asana project configured")
	}

	return s.db.RemoveIgnoredTicket(userID, settings.AsanaProjectID, ticketID, "forever")
}

// GetTemporarilyIgnored returns all temporarily ignored ticket IDs for user's current project
func (s *IgnoreService) GetTemporarilyIgnored(userID int) []string {
	settings, err := s.configService.GetSettings(userID)
	if err != nil || settings.AsanaProjectID == "" {
		return []string{}
	}

	ignoredTickets, err := s.db.GetIgnoredTickets(userID, settings.AsanaProjectID)
	if err != nil {
		return []string{}
	}

	var tempIgnored []string
	for _, ticket := range ignoredTickets {
		if ticket.IgnoreType == "temp" {
			tempIgnored = append(tempIgnored, ticket.TicketID)
		}
	}

	return tempIgnored
}

// GetForeverIgnored returns all permanently ignored ticket IDs for user's current project
func (s *IgnoreService) GetForeverIgnored(userID int) []string {
	settings, err := s.configService.GetSettings(userID)
	if err != nil || settings.AsanaProjectID == "" {
		return []string{}
	}

	ignoredTickets, err := s.db.GetIgnoredTickets(userID, settings.AsanaProjectID)
	if err != nil {
		return []string{}
	}

	var foreverIgnored []string
	for _, ticket := range ignoredTickets {
		if ticket.IgnoreType == "forever" {
			foreverIgnored = append(foreverIgnored, ticket.TicketID)
		}
	}

	return foreverIgnored
}

// GetIgnoredTickets returns all ignored ticket IDs (temp + forever) for user's current project
func (s *IgnoreService) GetIgnoredTickets(userID int) []string {
	settings, err := s.configService.GetSettings(userID)
	if err != nil || settings.AsanaProjectID == "" {
		return []string{}
	}

	ignoredTickets, err := s.db.GetIgnoredTickets(userID, settings.AsanaProjectID)
	if err != nil {
		return []string{}
	}

	var allIgnored []string
	for _, ticket := range ignoredTickets {
		allIgnored = append(allIgnored, ticket.TicketID)
	}

	return allIgnored
}

// ClearTemporaryIgnores clears all temporary ignores for user's current project
func (s *IgnoreService) ClearTemporaryIgnores(userID int) error {
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return fmt.Errorf("failed to get user settings: %w", err)
	}

	if settings.AsanaProjectID == "" {
		return fmt.Errorf("no Asana project configured")
	}

	return s.db.ClearIgnoredTickets(userID, settings.AsanaProjectID, "temp")
}

// ClearForeverIgnores clears all permanent ignores for user's current project
func (s *IgnoreService) ClearForeverIgnores(userID int) error {
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return fmt.Errorf("failed to get user settings: %w", err)
	}

	if settings.AsanaProjectID == "" {
		return fmt.Errorf("no Asana project configured")
	}

	return s.db.ClearIgnoredTickets(userID, settings.AsanaProjectID, "forever")
}

// ClearAllIgnores clears all ignores (temporary and permanent) for user's current project
func (s *IgnoreService) ClearAllIgnores(userID int) error {
	settings, err := s.configService.GetSettings(userID)
	if err != nil {
		return fmt.Errorf("failed to get user settings: %w", err)
	}

	if settings.AsanaProjectID == "" {
		return fmt.Errorf("no Asana project configured")
	}

	return s.db.ClearIgnoredTickets(userID, settings.AsanaProjectID, "")
}

// GetIgnoreStatus returns the ignore status for user's current project
func (s *IgnoreService) GetIgnoreStatus(userID int) map[string]interface{} {
	tempIgnored := s.GetTemporarilyIgnored(userID)
	foreverIgnored := s.GetForeverIgnored(userID)

	return map[string]interface{}{
		"temp_ignored":    tempIgnored,
		"forever_ignored": foreverIgnored,
		"temp_count":      len(tempIgnored),
		"forever_count":   len(foreverIgnored),
		"total_ignored":   len(tempIgnored) + len(foreverIgnored),
	}
}

// ProcessIgnoreRequest processes an ignore action request
func (s *IgnoreService) ProcessIgnoreRequest(userID int, ticketID, action, ignoreType string) error {
	switch action {
	case "add":
		if ignoreType == "forever" {
			return s.AddForeverIgnore(userID, ticketID)
		} else {
			return s.AddTemporaryIgnore(userID, ticketID)
		}
	case "remove":
		if ignoreType == "forever" {
			return s.RemoveForeverIgnore(userID, ticketID)
		} else {
			return s.RemoveTemporaryIgnore(userID, ticketID)
		}
	default:
		return fmt.Errorf("invalid action: %s", action)
	}
}

// MoveToForever moves a ticket from temporary to permanent ignore
func (s *IgnoreService) MoveToForever(userID int, ticketID string) error {
	if err := s.RemoveTemporaryIgnore(userID, ticketID); err != nil {
		return err
	}
	return s.AddForeverIgnore(userID, ticketID)
}

// MoveToTemporary moves a ticket from permanent to temporary ignore
func (s *IgnoreService) MoveToTemporary(userID int, ticketID string) error {
	if err := s.RemoveForeverIgnore(userID, ticketID); err != nil {
		return err
	}
	return s.AddTemporaryIgnore(userID, ticketID)
}

// HasAnyIgnored checks if there are any ignored tickets for user's current project
func (s *IgnoreService) HasAnyIgnored(userID int) bool {
	ignored := s.GetIgnoredTickets(userID)
	return len(ignored) > 0
}

// CountIgnored returns the total count of ignored tickets for user's current project
func (s *IgnoreService) CountIgnored(userID int) int {
	ignored := s.GetIgnoredTickets(userID)
	return len(ignored)
}