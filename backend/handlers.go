package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"service":   "enhanced-asana-youtrack-sync",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "3.2", // Updated version for delete functionality
		"features": []string{
			"Tag/Subsystem synchronization",
			"Individual ticket creation",
			"Enhanced status parsing",
			"Tag mismatch detection",
			"Auto-sync functionality",
			"Auto-create functionality",
			"Ticket detail views",
			"Interactive console (fixed)",
			"Bulk ticket deletion", // NEW
		},
		"columns": map[string]interface{}{
			"syncable":     syncableColumns,
			"display_only": displayOnlyColumns,
		},
	})
}

func statusCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service":          "enhanced-asana-youtrack-sync",
		"last_sync":        lastSyncTime.Format(time.RFC3339),
		"poll_interval":    config.PollIntervalMS,
		"asana_project":    config.AsanaProjectID,
		"youtrack_project": config.YouTrackProjectID,
		"columns": map[string]interface{}{
			"syncable":     syncableColumns,
			"display_only": displayOnlyColumns,
		},
		"temp_ignored":    len(ignoredTicketsTemp),
		"forever_ignored": len(ignoredTicketsForever),
		"tag_mappings":    len(defaultTagMapping),
		"auto_sync": map[string]interface{}{
			"running":   autoSyncRunning,
			"interval":  autoSyncInterval,
			"count":     autoSyncCount,
			"last_info": autoSyncLastInfo,
		},
		"auto_create": map[string]interface{}{
			"running":   autoCreateRunning,
			"interval":  autoCreateInterval,
			"count":     autoCreateCount,
			"last_info": autoCreateLastInfo,
		},
		"endpoints": []string{
			"GET /health - Health check",
			"GET /status - Service status",
			"GET /analyze - Analyze ticket differences",
			"POST /create - Create missing tickets (bulk)",
			"POST /create-single - Create individual ticket",
			"GET/POST /sync - Sync mismatched tickets",
			"GET/POST /ignore - Manage ignored tickets",
			"GET/POST /auto-sync - Control auto-sync functionality",
			"GET/POST /auto-create - Control auto-create functionality",
			"GET /tickets - Get tickets by type",
			"POST /delete-tickets - Delete tickets (bulk)", // NEW
		},
	})
}

func analyzeTicketsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed. Use GET.", http.StatusMethodNotAllowed)
		return
	}

	// Get column filter from query parameters
	columnFilter := r.URL.Query().Get("column")
	fmt.Printf("ANALYZE DEBUG: Received column filter: '%s'\n", columnFilter)

	// FIXED: Proper column mapping and filtering
	var columnsToAnalyze []string
	var mappedColumnName string

	if columnFilter == "" || columnFilter == "all_syncable" {
		columnsToAnalyze = syncableColumns
		mappedColumnName = "all_syncable"
	} else {
		// CRITICAL: Frontend to backend column name mapping
		columnMap := map[string]string{
			"backlog":         "backlog",
			"in_progress":     "in progress",
			"dev":             "dev",
			"stage":           "stage",
			"blocked":         "blocked",
			"ready_for_stage": "ready for stage",
			"findings":        "findings",
		}

		if mappedColumn, exists := columnMap[columnFilter]; exists {
			columnsToAnalyze = []string{mappedColumn}
			mappedColumnName = mappedColumn
			fmt.Printf("ANALYZE DEBUG: Column '%s' mapped to '%s'\n", columnFilter, mappedColumn)
		} else {
			fmt.Printf("ANALYZE DEBUG: Unknown column '%s', using all syncable columns\n", columnFilter)
			columnsToAnalyze = syncableColumns
			mappedColumnName = "all_syncable"
		}
	}

	fmt.Printf("ANALYZE DEBUG: Final columns to analyze: %v\n", columnsToAnalyze)

	// Perform analysis with the specific columns
	analysis, err := performTicketAnalysis(columnsToAnalyze)
	if err != nil {
		fmt.Printf("ANALYZE DEBUG: Analysis failed: %v\n", err)
		http.Error(w, fmt.Sprintf("Analysis failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Count different types of mismatches
	tagMismatchCount := 0
	statusMismatchCount := 0
	for _, ticket := range analysis.Mismatched {
		if ticket.TagMismatch {
			tagMismatchCount++
		}
		if ticket.AsanaStatus != ticket.YouTrackStatus {
			statusMismatchCount++
		}
	}

	fmt.Printf("ANALYZE DEBUG: Analysis complete - Matched: %d, Mismatched: %d, Missing: %d\n",
		len(analysis.Matched), len(analysis.Mismatched), len(analysis.MissingYouTrack))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":           "success",
		"timestamp":        time.Now().Format(time.RFC3339),
		"analysis":         analysis,
		"column_filter":    columnFilter,
		"mapped_column":    mappedColumnName,
		"analyzed_columns": columnsToAnalyze,
		"summary": map[string]int{
			"matched":           len(analysis.Matched),
			"mismatched":        len(analysis.Mismatched),
			"missing_youtrack":  len(analysis.MissingYouTrack),
			"findings_tickets":  len(analysis.FindingsTickets),
			"findings_alerts":   len(analysis.FindingsAlerts),
			"ready_for_stage":   len(analysis.ReadyForStage),
			"blocked_tickets":   len(analysis.BlockedTickets),
			"orphaned_youtrack": len(analysis.OrphanedYouTrack),
			"ignored":           len(analysis.Ignored),
			"tag_mismatches":    tagMismatchCount,
			"status_mismatches": statusMismatchCount,
		},
	})
}

// Get tickets by type handler
// FIXED: Update the getTicketsByTypeHandler to accept column parameter
func getTicketsByTypeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed. Use GET.", http.StatusMethodNotAllowed)
		return
	}

	ticketType := r.URL.Query().Get("type")
	column := r.URL.Query().Get("column")

	if ticketType == "" {
		http.Error(w, "Missing 'type' parameter", http.StatusBadRequest)
		return
	}

	// Handle ignored tickets separately (they don't have column context)
	if ticketType == "ignored" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"type":    "ignored",
			"tickets": getMapKeys(ignoredTicketsForever),
			"count":   len(ignoredTicketsForever),
		})
		return
	}

	// CRITICAL FIX: Determine which columns to analyze based on the column parameter
	var columnsToAnalyze []string
	if column == "" || column == "all_syncable" {
		columnsToAnalyze = syncableColumns
	} else {
		// FIXED: Proper frontend to backend column name mapping
		columnMap := map[string]string{
			"backlog":         "backlog",
			"in_progress":     "in progress",
			"dev":             "dev",
			"stage":           "stage",
			"blocked":         "blocked",
			"ready_for_stage": "ready for stage",
			"findings":        "findings",
		}

		if mappedColumn, exists := columnMap[column]; exists {
			columnsToAnalyze = []string{mappedColumn}
			fmt.Printf("HANDLER DEBUG: Analyzing column '%s' mapped to '%s'\n", column, mappedColumn)
		} else {
			fmt.Printf("HANDLER DEBUG: Unknown column '%s', falling back to all syncable columns\n", column)
			columnsToAnalyze = syncableColumns
		}
	}

	fmt.Printf("HANDLER DEBUG: Final columns to analyze: %v\n", columnsToAnalyze)

	// Use the column-specific analysis
	analysis, err := performTicketAnalysis(columnsToAnalyze)
	if err != nil {
		fmt.Printf("HANDLER DEBUG: Analysis failed: %v\n", err)
		http.Error(w, fmt.Sprintf("Analysis failed: %v", err), http.StatusInternalServerError)
		return
	}

	var tickets interface{}
	var count int

	switch ticketType {
	case "matched":
		tickets = analysis.Matched
		count = len(analysis.Matched)
	case "mismatched":
		tickets = analysis.Mismatched
		count = len(analysis.Mismatched)
	case "missing":
		tickets = analysis.MissingYouTrack
		count = len(analysis.MissingYouTrack)
	case "findings":
		tickets = analysis.FindingsTickets
		count = len(analysis.FindingsTickets)
	case "ready_for_stage":
		tickets = analysis.ReadyForStage
		count = len(analysis.ReadyForStage)
	case "blocked":
		tickets = analysis.BlockedTickets
		count = len(analysis.BlockedTickets)
	case "orphaned":
		tickets = analysis.OrphanedYouTrack
		count = len(analysis.OrphanedYouTrack)
	default:
		http.Error(w, "Invalid ticket type", http.StatusBadRequest)
		return
	}

	fmt.Printf("HANDLER DEBUG: Returning %d tickets of type '%s' for column '%s'\n", count, ticketType, column)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":           "success",
		"type":             ticketType,
		"column":           column,
		"analyzed_columns": columnsToAnalyze,
		"tickets":          tickets,
		"count":            count,
	})
}

// NEW: Delete tickets handler
func deleteTicketsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed. Use POST.", http.StatusMethodNotAllowed)
		return
	}

	var req DeleteTicketsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":    "Invalid JSON format",
			"expected": "Object like: {\"ticket_ids\":[\"123\",\"456\"],\"source\":\"asana|youtrack|both\"}",
			"example":  `{"ticket_ids":["1234567890","0987654321"],"source":"both"}`,
		})
		return
	}

	// Validate request
	if len(req.TicketIDs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "ticket_ids is required and must not be empty",
			"example": `{"ticket_ids":["1234567890"],"source":"asana"}`,
		})
		return
	}

	if req.Source == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":         "source is required",
			"valid_sources": []string{"asana", "youtrack", "both"},
			"example":       `{"ticket_ids":["1234567890"],"source":"asana"}`,
		})
		return
	}

	// Validate source value
	validSources := map[string]bool{"asana": true, "youtrack": true, "both": true}
	if !validSources[req.Source] {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":         "Invalid source value",
			"valid_sources": []string{"asana", "youtrack", "both"},
			"received":      req.Source,
		})
		return
	}

	// Perform bulk delete
	fmt.Printf("Starting bulk delete of %d tickets from %s\n", len(req.TicketIDs), req.Source)

	response := performBulkDelete(req.TicketIDs, req.Source)

	// Set appropriate HTTP status based on result
	httpStatus := http.StatusOK
	if response.Status == "failed" {
		httpStatus = http.StatusInternalServerError
	} else if response.Status == "partial" {
		httpStatus = http.StatusPartialContent
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(response)

	fmt.Printf("Bulk delete completed: %s\n", response.Summary)
}

func createMissingTicketsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" && r.Method != "GET" {
		http.Error(w, "Method not allowed. Use POST or GET.", http.StatusMethodNotAllowed)
		return
	}

	analysis, err := performTicketAnalysis(syncableColumns)
	if err != nil {
		http.Error(w, fmt.Sprintf("Analysis failed: %v", err), http.StatusInternalServerError)
		return
	}

	if len(analysis.MissingYouTrack) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"message": "No missing tickets to create",
			"created": 0,
		})
		return
	}

	results := []map[string]interface{}{}
	created := 0
	skipped := 0

	for _, task := range analysis.MissingYouTrack {
		asanaTags := getAsanaTags(task)

		result := map[string]interface{}{
			"task_id":    task.GID,
			"task_name":  task.Name,
			"asana_tags": asanaTags,
		}

		if isDuplicateTicket(task.Name) {
			result["status"] = "skipped"
			result["reason"] = "Duplicate ticket already exists"
			skipped++
		} else if isIgnored(task.GID) {
			result["status"] = "skipped"
			result["reason"] = "Ticket is ignored"
			skipped++
		} else {
			err := createYouTrackIssue(task)
			if err != nil {
				result["status"] = "failed"
				result["error"] = err.Error()
			} else {
				result["status"] = "created"
				if len(asanaTags) > 0 {
					primaryTag := asanaTags[0]
					mappedSubsystem := mapTagToSubsystem(primaryTag)
					result["mapped_subsystem"] = mappedSubsystem
				}
				created++
			}
		}
		results = append(results, result)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "completed",
		"created": created,
		"skipped": skipped,
		"total":   len(analysis.MissingYouTrack),
		"results": results,
	})
}

func createSingleTicketHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed. Use POST.", http.StatusMethodNotAllowed)
		return
	}

	var req CreateSingleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":    "Invalid JSON format",
			"expected": "Object like: {\"task_id\":\"1234567890\"}",
			"example":  `{"task_id":"1234567890"}`,
		})
		return
	}

	if req.TaskID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "task_id is required",
			"example": `{"task_id":"1234567890"}`,
		})
		return
	}

	allTasks, err := getAsanaTasks()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get Asana tasks: %v", err), http.StatusInternalServerError)
		return
	}

	var targetTask *AsanaTask
	for _, task := range allTasks {
		if task.GID == req.TaskID {
			targetTask = &task
			break
		}
	}

	if targetTask == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Task not found",
			"task_id": req.TaskID,
		})
		return
	}

	asanaTags := getAsanaTags(*targetTask)

	if isDuplicateTicket(targetTask.Name) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "skipped",
			"reason":     "Duplicate ticket already exists",
			"task_id":    req.TaskID,
			"task_name":  targetTask.Name,
			"asana_tags": asanaTags,
		})
		return
	}

	if isIgnored(req.TaskID) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "skipped",
			"reason":     "Ticket is ignored",
			"task_id":    req.TaskID,
			"task_name":  targetTask.Name,
			"asana_tags": asanaTags,
		})
		return
	}

	err = createYouTrackIssue(*targetTask)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "failed",
			"error":      err.Error(),
			"task_id":    req.TaskID,
			"task_name":  targetTask.Name,
			"asana_tags": asanaTags,
		})
		return
	}

	response := map[string]interface{}{
		"status":     "created",
		"task_id":    req.TaskID,
		"task_name":  targetTask.Name,
		"asana_tags": asanaTags,
	}

	if len(asanaTags) > 0 {
		primaryTag := asanaTags[0]
		mappedSubsystem := mapTagToSubsystem(primaryTag)
		response["mapped_subsystem"] = mappedSubsystem
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func syncMismatchedTicketsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method == "GET" {
		analysis, err := performTicketAnalysis(syncableColumns)
		if err != nil {
			http.Error(w, fmt.Sprintf("Analysis failed: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "success",
			"message":    "Mismatched tickets available for sync",
			"count":      len(analysis.Mismatched),
			"mismatched": analysis.Mismatched,
			"usage": map[string]string{
				"sync_all":       "POST with [{\"ticket_id\":\"ID\",\"action\":\"sync\"}] for each ticket",
				"ignore_temp":    "POST with [{\"ticket_id\":\"ID\",\"action\":\"ignore_temp\"}]",
				"ignore_forever": "POST with [{\"ticket_id\":\"ID\",\"action\":\"ignore_forever\"}]",
			},
			"note": "Sync now includes both status and tag/subsystem synchronization",
		})
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed. Use GET to see available tickets, POST to sync.", http.StatusMethodNotAllowed)
		return
	}

	var requests []SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":         "Invalid JSON format",
			"expected":      "Array of objects like: [{\"ticket_id\":\"123\",\"action\":\"sync\"}]",
			"valid_actions": []string{"sync", "ignore_temp", "ignore_forever"},
			"example":       `[{"ticket_id":"1234567890","action":"sync"}]`,
		})
		return
	}

	analysis, err := performTicketAnalysis(syncableColumns)
	if err != nil {
		http.Error(w, fmt.Sprintf("Analysis failed: %v", err), http.StatusInternalServerError)
		return
	}

	mismatchMap := make(map[string]MismatchedTicket)
	for _, ticket := range analysis.Mismatched {
		mismatchMap[ticket.AsanaTask.GID] = ticket
	}

	results := []map[string]interface{}{}
	synced := 0

	for _, req := range requests {
		result := map[string]interface{}{
			"ticket_id": req.TicketID,
			"action":    req.Action,
		}

		ticket, exists := mismatchMap[req.TicketID]
		if !exists {
			result["status"] = "failed"
			result["error"] = "Ticket not found in mismatched list"
			results = append(results, result)
			continue
		}

		switch req.Action {
		case "sync":
			if isIgnored(req.TicketID) {
				result["status"] = "skipped"
				result["reason"] = "Ticket is ignored"
			} else {
				err := updateYouTrackIssue(ticket.YouTrackIssue.ID, ticket.AsanaTask)
				if err != nil {
					result["status"] = "failed"
					result["error"] = err.Error()
				} else {
					result["status"] = "synced"
					result["status_change"] = map[string]string{
						"from": ticket.YouTrackStatus,
						"to":   ticket.AsanaStatus,
					}

					asanaTags := getAsanaTags(ticket.AsanaTask)
					if len(asanaTags) > 0 {
						primaryTag := asanaTags[0]
						mappedSubsystem := mapTagToSubsystem(primaryTag)
						result["tag_sync"] = map[string]interface{}{
							"asana_tags":         asanaTags,
							"mapped_subsystem":   mappedSubsystem,
							"previous_subsystem": ticket.YouTrackSubsystem,
						}
					}
					synced++
				}
			}

		case "ignore_temp":
			ignoredTicketsTemp[req.TicketID] = true
			result["status"] = "ignored_temporarily"

		case "ignore_forever":
			ignoredTicketsForever[req.TicketID] = true
			saveIgnoredTickets()
			result["status"] = "ignored_permanently"

		default:
			result["status"] = "failed"
			result["error"] = "Invalid action"
		}

		results = append(results, result)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "completed",
		"synced":  synced,
		"total":   len(requests),
		"results": results,
		"note":    "Sync operations now include both status and tag/subsystem updates",
	})
}

func manageIgnoredTicketsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch r.Method {
	case "GET":
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"temp_ignored":    getMapKeys(ignoredTicketsTemp),
			"forever_ignored": getMapKeys(ignoredTicketsForever),
			"tag_mappings":    defaultTagMapping,
		})

	case "POST":
		var req IgnoreRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		switch req.Action {
		case "add":
			if req.Type == "forever" {
				ignoredTicketsForever[req.TicketID] = true
				saveIgnoredTickets()
			} else {
				ignoredTicketsTemp[req.TicketID] = true
			}

		case "remove":
			if req.Type == "forever" {
				delete(ignoredTicketsForever, req.TicketID)
				saveIgnoredTickets()
			} else {
				delete(ignoredTicketsTemp, req.TicketID)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "success",
			"action": req.Action,
			"type":   req.Type,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Auto-sync control handler
func autoSyncHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch r.Method {
	case "GET":
		status := AutoSyncStatus{
			Running:      autoSyncRunning,
			Interval:     autoSyncInterval,
			LastSync:     lastSyncTime,
			SyncCount:    autoSyncCount,
			LastSyncInfo: autoSyncLastInfo,
		}

		if autoSyncRunning {
			status.NextSync = time.Now().Add(time.Duration(autoSyncInterval) * time.Second)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "success",
			"auto_sync": status,
			"capabilities": []string{
				"start - Start auto-sync with specified interval",
				"stop - Stop auto-sync",
			},
		})

	case "POST":
		var req AutoSyncRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":    "Invalid JSON format",
				"expected": "Object like: {\"action\":\"start\",\"interval\":15}",
				"example":  `{"action":"start","interval":15}`,
			})
			return
		}

		switch req.Action {
		case "start":
			if autoSyncRunning {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":           "already_running",
					"message":          "Auto-sync is already running",
					"current_interval": autoSyncInterval,
				})
				return
			}

			if req.Interval > 0 {
				autoSyncInterval = req.Interval
			} else {
				autoSyncInterval = 15
			}

			startAutoSync()

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":   "started",
				"message":  "Auto-sync started successfully",
				"interval": autoSyncInterval,
			})

		case "stop":
			if !autoSyncRunning {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":  "not_running",
					"message": "Auto-sync is not currently running",
				})
				return
			}

			stopAutoSync()

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":     "stopped",
				"message":    "Auto-sync stopped successfully",
				"sync_count": autoSyncCount,
			})

		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":         "Invalid action",
				"valid_actions": []string{"start", "stop"},
				"example":       `{"action":"start","interval":15}`,
			})
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Auto-create control handler
func autoCreateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch r.Method {
	case "GET":
		status := AutoCreateStatus{
			Running:        autoCreateRunning,
			Interval:       autoCreateInterval,
			CreateCount:    autoCreateCount,
			LastCreateInfo: autoCreateLastInfo,
		}

		if autoCreateRunning {
			status.NextCreate = time.Now().Add(time.Duration(autoCreateInterval) * time.Second)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":      "success",
			"auto_create": status,
			"capabilities": []string{
				"start - Start auto-create with specified interval",
				"stop - Stop auto-create",
			},
		})

	case "POST":
		var req AutoCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":    "Invalid JSON format",
				"expected": "Object like: {\"action\":\"start\",\"interval\":15}",
				"example":  `{"action":"start","interval":15}`,
			})
			return
		}

		switch req.Action {
		case "start":
			if autoCreateRunning {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":           "already_running",
					"message":          "Auto-create is already running",
					"current_interval": autoCreateInterval,
				})
				return
			}

			if req.Interval > 0 {
				autoCreateInterval = req.Interval
			} else {
				autoCreateInterval = 15
			}

			startAutoCreate()

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":   "started",
				"message":  "Auto-create started successfully",
				"interval": autoCreateInterval,
			})

		case "stop":
			if !autoCreateRunning {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":  "not_running",
					"message": "Auto-create is not currently running",
				})
				return
			}

			stopAutoCreate()

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":       "stopped",
				"message":      "Auto-create stopped successfully",
				"create_count": autoCreateCount,
			})

		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":         "Invalid action",
				"valid_actions": []string{"start", "stop"},
				"example":       `{"action":"start","interval":15}`,
			})
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Auto-sync functions
func startAutoSync() {
	if autoSyncRunning {
		return
	}

	autoSyncRunning = true
	autoSyncDone = make(chan bool)
	autoSyncTicker = time.NewTicker(time.Duration(autoSyncInterval) * time.Second)

	fmt.Printf("Auto-sync started with %d second interval\n", autoSyncInterval)

	go func() {
		for {
			select {
			case <-autoSyncTicker.C:
				performAutoSync()
			case <-autoSyncDone:
				autoSyncTicker.Stop()
				fmt.Println("Auto-sync stopped")
				return
			}
		}
	}()
}

func stopAutoSync() {
	if !autoSyncRunning {
		return
	}

	autoSyncRunning = false
	if autoSyncDone != nil {
		close(autoSyncDone)
	}
	if autoSyncTicker != nil {
		autoSyncTicker.Stop()
	}

	fmt.Println("Auto-sync stopped")
}

func performAutoSync() {
	fmt.Printf("Performing auto-sync #%d...\n", autoSyncCount+1)

	analysis, err := performTicketAnalysis(syncableColumns)
	if err != nil {
		autoSyncLastInfo = fmt.Sprintf("Analysis failed: %v", err)
		fmt.Printf("Auto-sync analysis failed: %v\n", err)
		return
	}

	synced := 0
	errors := 0

	for _, ticket := range analysis.Mismatched {
		if isIgnored(ticket.AsanaTask.GID) {
			continue
		}

		err := updateYouTrackIssue(ticket.YouTrackIssue.ID, ticket.AsanaTask)
		if err != nil {
			fmt.Printf("Auto-sync error updating ticket %s: %v\n", ticket.AsanaTask.GID, err)
			errors++
		} else {
			synced++
		}
	}

	autoSyncCount++
	lastSyncTime = time.Now()
	autoSyncLastInfo = fmt.Sprintf("Synced: %d, Errors: %d", synced, errors)

	fmt.Printf("Auto-sync #%d completed: %s\n", autoSyncCount, autoSyncLastInfo)
}

// Auto-create functions
func startAutoCreate() {
	if autoCreateRunning {
		return
	}

	autoCreateRunning = true
	autoCreateDone = make(chan bool)
	autoCreateTicker = time.NewTicker(time.Duration(autoCreateInterval) * time.Second)

	fmt.Printf("Auto-create started with %d second interval\n", autoCreateInterval)

	go func() {
		for {
			select {
			case <-autoCreateTicker.C:
				performAutoCreate()
			case <-autoCreateDone:
				autoCreateTicker.Stop()
				fmt.Println("Auto-create stopped")
				return
			}
		}
	}()
}

func stopAutoCreate() {
	if !autoCreateRunning {
		return
	}

	autoCreateRunning = false
	if autoCreateDone != nil {
		close(autoCreateDone)
	}
	if autoCreateTicker != nil {
		autoCreateTicker.Stop()
	}

	fmt.Println("Auto-create stopped")
}

func performAutoCreate() {
	fmt.Printf("Performing auto-create #%d...\n", autoCreateCount+1)

	analysis, err := performTicketAnalysis(syncableColumns)
	if err != nil {
		autoCreateLastInfo = fmt.Sprintf("Analysis failed: %v", err)
		fmt.Printf("Auto-create analysis failed: %v\n", err)
		return
	}

	created := 0
	errors := 0

	for _, task := range analysis.MissingYouTrack {
		if isIgnored(task.GID) || isDuplicateTicket(task.Name) {
			continue
		}

		err := createYouTrackIssue(task)
		if err != nil {
			fmt.Printf("Auto-create error creating ticket %s: %v\n", task.GID, err)
			errors++
		} else {
			created++
		}
	}

	autoCreateCount++
	autoCreateLastInfo = fmt.Sprintf("Created: %d, Errors: %d", created, errors)

	fmt.Printf("Auto-create #%d completed: %s\n", autoCreateCount, autoCreateLastInfo)
}
