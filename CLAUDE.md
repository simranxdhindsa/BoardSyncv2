# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Rules
- This is a vibecoded app so make sure to search for vulnerabilites
- Always before making any change, search on the web for the newest documentation and only implement if you are 100% sure it will work.
- Try to use same components and same css to maintain consistency in Frontend

## Commands

### Backend
```bash
cd backend
go mod download       # Install dependencies
go run main.go        # Start server on :8080
go build ./...        # Compile check (no output = success)
```

### Frontend
```bash
cd frontend
npm install
npm start             # Dev server on http://localhost:3000
npm run build         # Production build
npm run electron-dev  # Electron desktop dev mode
npm run electron-build-win  # Windows .exe
```

No test suite exists. The backend has no `_test.go` files. Frontend has Jest via `react-scripts test` but no test files have been written.

## Architecture

### Data Flow
The core workflow: user selects Asana columns → `/analyze` fetches & compares Asana tasks vs YouTrack issues → `/sync` pushes status updates to YouTrack.

### Backend (`backend/`)

**`main.go`** — initializes all services, wires dependencies, registers all HTTP routes. This is the composition root.

**`legacy/`** — despite the name, this is the **active primary sync engine**, not deprecated code. It contains:
- `analysis_service.go` — fetches Asana tasks + YouTrack issues, matches them (priority: DB mapping → description extraction → title fuzzy match), categorizes into matched/mismatched/missing/orphaned
- `sync_service.go` — performs the actual YouTrack updates; `SyncMismatchedTickets` uses DB mappings first then falls back to description/title search, and saves found mappings for future use
- `asana_service.go` — Asana API client; `FilterTasksByColumns` handles per-column filtering logic with explicit `case` handlers for each column name
- `youtrack_service.go` — YouTrack API client; `ExtractAsanaID` parses `Asana ID:` from YouTrack issue descriptions; `FindIssueByAsanaID` does a full YT issue scan
- `handlers.go` — HTTP layer for legacy API; every column-filtering handler has a `columnMap` that must be updated when adding new columns
- `auto_managers.go` — background goroutines for auto-sync and auto-create polling

**`database/`** — Pure Go JSON file database (no SQL). Data lives in `sync_app.db_data/data.json`. All state is loaded into memory maps at startup and written back on each mutation. `TicketMappings` table is the critical link between Asana task GIDs and YouTrack issue IDs.

**`sync/`** — Rollback/snapshot/audit features. Snapshots are capped at 15 per ticket and expire after 24h. WebSocket handler for real-time status pushes to frontend.

**`auth/`** — JWT auth with Argon2 password hashing. Token stored in localStorage on the frontend under `auth_token` or `token`.

**`config/`** — Per-user settings including Asana/YouTrack credentials, project IDs, and `ColumnMappings.AsanaToYouTrack` which maps Asana section names to YouTrack status strings.

### Frontend (`frontend/src/`)

**`services/api.js`** — single API client; reads `REACT_APP_API_URL` or defaults to `http://localhost:8080`.

**`App.js`** — top-level state: selected column, analysis results, current view.

**`components/TicketDetailView.js`** (62KB) and **`components/AnalysisResults.js`** (39KB) are the largest, most complex UI files.

**`contexts/AuthContext.js`** — global auth state.

### Adding a New Column

When adding a new Asana column name (e.g. `"prod"`), all of these must be updated:
1. `legacy/types.go` — `SyncableColumns` slice
2. `legacy/handlers.go` — 5 `columnMap` instances (AnalyzeTickets, Create handler, Sync handler, AnalyzeTicketsEnhanced, FilterOptions)
3. `legacy/analysis_service.go` — `columnMap` in `GetTicketsByType` and `case` in `isSyncableSection`
4. `legacy/sync_service.go` — `columnMap` in `SyncTicketsByColumn` and `CreateTicketsByColumn`
5. `legacy/asana_service.go` — `case` in `FilterTasksByColumns`

### Ticket Matching Priority

The system matches Asana tasks to YouTrack issues in this order:
1. **DB mapping** (`ticket_mappings` table) — fastest, used for previously synced tickets
2. **Description extraction** — parses `Asana ID: <gid>` from YouTrack issue description
3. **Title fuzzy match** — normalizes and compares titles (strips `-`, `_`, `/`, lowercases)

When a match is found via fallback (2 or 3), it is saved to the DB mapping table for future O(1) lookups.

### Column Mapping for Status Sync

Asana section names → YouTrack state strings are configured per-user in `UserSettings.ColumnMappings.AsanaToYouTrack`. The `MapStateToYouTrackWithSettings` function uses this; if no mapping is found for a section, it logs a warning and returns `""` (ticket treated as unresolvable).

