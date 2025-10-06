// backend/legacy/filtering_sorting.go - NEW FILE
package legacy

import (
	"sort"
	"strings"
)

// FilterTickets applies filters to a list of matched tickets
func FilterMatchedTickets(tickets []MatchedTicket, filter TicketFilter) []MatchedTicket {
	if !hasActiveFilter(filter) {
		return tickets
	}

	filtered := []MatchedTicket{}
	for _, ticket := range tickets {
		if matchesFilter(ticket, filter) {
			filtered = append(filtered, ticket)
		}
	}
	return filtered
}

// FilterMismatchedTickets applies filters to mismatched tickets
func FilterMismatchedTickets(tickets []MismatchedTicket, filter TicketFilter) []MismatchedTicket {
	if !hasActiveFilter(filter) {
		return tickets
	}

	filtered := []MismatchedTicket{}
	for _, ticket := range tickets {
		if matchesMismatchedFilter(ticket, filter) {
			filtered = append(filtered, ticket)
		}
	}
	return filtered
}

// FilterAsanaTasks applies filters to Asana tasks
func FilterAsanaTasks(tasks []AsanaTask, filter TicketFilter, asanaService *AsanaService, userID int) []AsanaTask {
	if !hasActiveFilter(filter) {
		return tasks
	}

	filtered := []AsanaTask{}
	for _, task := range tasks {
		if matchesAsanaTaskFilter(task, filter, asanaService, userID) {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

// SortMatchedTickets sorts matched tickets based on sort options
func SortMatchedTickets(tickets []MatchedTicket, sortOpts TicketSortOptions) []MatchedTicket {
	if sortOpts.SortBy == "" {
		return tickets
	}

	sorted := make([]MatchedTicket, len(tickets))
	copy(sorted, tickets)

	switch sortOpts.SortBy {
	case "created_at":
		sort.Slice(sorted, func(i, j int) bool {
			if sortOpts.SortOrder == "desc" {
				return sorted[i].CreatedAt.After(sorted[j].CreatedAt)
			}
			return sorted[i].CreatedAt.Before(sorted[j].CreatedAt)
		})
	case "assignee":
		sort.Slice(sorted, func(i, j int) bool {
			if sortOpts.SortOrder == "desc" {
				return sorted[i].AssigneeName > sorted[j].AssigneeName
			}
			return sorted[i].AssigneeName < sorted[j].AssigneeName
		})
	case "priority":
		sort.Slice(sorted, func(i, j int) bool {
			if sortOpts.SortOrder == "desc" {
				return getPriorityValue(sorted[i].Priority) > getPriorityValue(sorted[j].Priority)
			}
			return getPriorityValue(sorted[i].Priority) < getPriorityValue(sorted[j].Priority)
		})
	}

	return sorted
}

// SortMismatchedTickets sorts mismatched tickets
func SortMismatchedTickets(tickets []MismatchedTicket, sortOpts TicketSortOptions) []MismatchedTicket {
	if sortOpts.SortBy == "" {
		return tickets
	}

	sorted := make([]MismatchedTicket, len(tickets))
	copy(sorted, tickets)

	switch sortOpts.SortBy {
	case "created_at":
		sort.Slice(sorted, func(i, j int) bool {
			if sortOpts.SortOrder == "desc" {
				return sorted[i].CreatedAt.After(sorted[j].CreatedAt)
			}
			return sorted[i].CreatedAt.Before(sorted[j].CreatedAt)
		})
	case "assignee":
		sort.Slice(sorted, func(i, j int) bool {
			if sortOpts.SortOrder == "desc" {
				return sorted[i].AssigneeName > sorted[j].AssigneeName
			}
			return sorted[i].AssigneeName < sorted[j].AssigneeName
		})
	case "priority":
		sort.Slice(sorted, func(i, j int) bool {
			if sortOpts.SortOrder == "desc" {
				return getPriorityValue(sorted[i].Priority) > getPriorityValue(sorted[j].Priority)
			}
			return getPriorityValue(sorted[i].Priority) < getPriorityValue(sorted[j].Priority)
		})
	}

	return sorted
}

// SortAsanaTasks sorts Asana tasks
func SortAsanaTasks(tasks []AsanaTask, sortOpts TicketSortOptions, asanaService *AsanaService, userID int) []AsanaTask {
	if sortOpts.SortBy == "" {
		return tasks
	}

	sorted := make([]AsanaTask, len(tasks))
	copy(sorted, tasks)

	switch sortOpts.SortBy {
	case "created_at":
		sort.Slice(sorted, func(i, j int) bool {
			ti := asanaService.GetCreatedAt(sorted[i])
			tj := asanaService.GetCreatedAt(sorted[j])
			if sortOpts.SortOrder == "desc" {
				return ti.After(tj)
			}
			return ti.Before(tj)
		})
	case "assignee":
		sort.Slice(sorted, func(i, j int) bool {
			ai := asanaService.GetAssigneeName(sorted[i])
			aj := asanaService.GetAssigneeName(sorted[j])
			if sortOpts.SortOrder == "desc" {
				return ai > aj
			}
			return ai < aj
		})
	case "priority":
		sort.Slice(sorted, func(i, j int) bool {
			pi := asanaService.GetPriority(sorted[i], userID)
			pj := asanaService.GetPriority(sorted[j], userID)
			if sortOpts.SortOrder == "desc" {
				return getPriorityValue(pi) > getPriorityValue(pj)
			}
			return getPriorityValue(pi) < getPriorityValue(pj)
		})
	}

	return sorted
}

// Helper functions

func hasActiveFilter(filter TicketFilter) bool {
	return len(filter.Assignees) > 0 ||
		!filter.StartDate.IsZero() ||
		!filter.EndDate.IsZero() ||
		len(filter.Priority) > 0
}

func matchesFilter(ticket MatchedTicket, filter TicketFilter) bool {
	// Filter by assignees
	if len(filter.Assignees) > 0 {
		found := false
		for _, assignee := range filter.Assignees {
			if strings.EqualFold(ticket.AssigneeName, assignee) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by date range
	if !filter.StartDate.IsZero() && ticket.CreatedAt.Before(filter.StartDate) {
		return false
	}
	if !filter.EndDate.IsZero() && ticket.CreatedAt.After(filter.EndDate) {
		return false
	}

	// Filter by priority
	if len(filter.Priority) > 0 {
		found := false
		for _, priority := range filter.Priority {
			if strings.EqualFold(ticket.Priority, priority) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func matchesMismatchedFilter(ticket MismatchedTicket, filter TicketFilter) bool {
	// Filter by assignees
	if len(filter.Assignees) > 0 {
		found := false
		for _, assignee := range filter.Assignees {
			if strings.EqualFold(ticket.AssigneeName, assignee) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by date range
	if !filter.StartDate.IsZero() && ticket.CreatedAt.Before(filter.StartDate) {
		return false
	}
	if !filter.EndDate.IsZero() && ticket.CreatedAt.After(filter.EndDate) {
		return false
	}

	// Filter by priority
	if len(filter.Priority) > 0 {
		found := false
		for _, priority := range filter.Priority {
			if strings.EqualFold(ticket.Priority, priority) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func matchesAsanaTaskFilter(task AsanaTask, filter TicketFilter, asanaService *AsanaService, userID int) bool {
	// Filter by assignees
	if len(filter.Assignees) > 0 {
		assigneeName := asanaService.GetAssigneeName(task)
		found := false
		for _, assignee := range filter.Assignees {
			if strings.EqualFold(assigneeName, assignee) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by date range
	createdAt := asanaService.GetCreatedAt(task)
	if !filter.StartDate.IsZero() && createdAt.Before(filter.StartDate) {
		return false
	}
	if !filter.EndDate.IsZero() && createdAt.After(filter.EndDate) {
		return false
	}

	// Filter by priority
	if len(filter.Priority) > 0 {
		taskPriority := asanaService.GetPriority(task, userID)
		found := false
		for _, priority := range filter.Priority {
			if strings.EqualFold(taskPriority, priority) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func getPriorityValue(priority string) int {
	// Assign numeric values for sorting priorities
	priorityMap := map[string]int{
		"urgent": 4,
		"high":   3,
		"medium": 2,
		"low":    1,
		"":       0,
	}

	if val, ok := priorityMap[strings.ToLower(priority)]; ok {
		return val
	}
	return 0
}

// GetUniqueAssignees extracts unique assignees from tickets
func GetUniqueAssignees(matchedTickets []MatchedTicket, mismatchedTickets []MismatchedTicket, missingTickets []AsanaTask, asanaService *AsanaService) []string {
	assigneeSet := make(map[string]bool)

	for _, ticket := range matchedTickets {
		if ticket.AssigneeName != "" {
			assigneeSet[ticket.AssigneeName] = true
		}
	}

	for _, ticket := range mismatchedTickets {
		if ticket.AssigneeName != "" {
			assigneeSet[ticket.AssigneeName] = true
		}
	}

	for _, task := range missingTickets {
		assigneeName := asanaService.GetAssigneeName(task)
		if assigneeName != "" {
			assigneeSet[assigneeName] = true
		}
	}

	assignees := []string{}
	for assignee := range assigneeSet {
		assignees = append(assignees, assignee)
	}

	sort.Strings(assignees)
	return assignees
}

// GetUniquePriorities extracts unique priorities from tickets
func GetUniquePriorities(matchedTickets []MatchedTicket, mismatchedTickets []MismatchedTicket, missingTickets []AsanaTask, asanaService *AsanaService, userID int) []string {
	prioritySet := make(map[string]bool)

	for _, ticket := range matchedTickets {
		if ticket.Priority != "" {
			prioritySet[ticket.Priority] = true
		}
	}

	for _, ticket := range mismatchedTickets {
		if ticket.Priority != "" {
			prioritySet[ticket.Priority] = true
		}
	}

	for _, task := range missingTickets {
		priority := asanaService.GetPriority(task, userID)
		if priority != "" {
			prioritySet[priority] = true
		}
	}

	priorities := []string{}
	for priority := range prioritySet {
		priorities = append(priorities, priority)
	}

	// Sort by priority value
	sort.Slice(priorities, func(i, j int) bool {
		return getPriorityValue(priorities[i]) > getPriorityValue(priorities[j])
	})

	return priorities
}
