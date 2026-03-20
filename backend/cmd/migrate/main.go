// cmd/migrate/main.go
//
// One-time migration tool: reads the local data.json and inserts all records
// into the Neon (PostgreSQL) database.
//
// Usage:
//   cd backend
//   DATABASE_URL="postgresql://..." go run ./cmd/migrate/main.go
//
// The tool is idempotent: running it twice will not duplicate records
// (users/settings use ON CONFLICT DO NOTHING, mappings use ON CONFLICT DO UPDATE).

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── JSON file shape (mirrors the old DB saveData struct) ─────────────────────

type jsonDB struct {
	Users                    map[string]jsonUser                    `json:"users"`
	Settings                 map[string]jsonSettings                `json:"settings"`
	Operations               map[string]jsonOperation               `json:"operations"`
	IgnoredTickets           map[string]jsonIgnoredTicket           `json:"ignored_tickets"`
	TicketMappings           map[string]jsonTicketMapping           `json:"ticket_mappings"`
	RollbackSnapshots        map[string]jsonSnapshot                `json:"rollback_snapshots"`
	AuditLogs                map[string]jsonAuditLog                `json:"audit_logs"`
	ReverseIgnoredTickets    map[string]jsonReverseIgnored          `json:"reverse_ignored_tickets"`
	ReverseAutoCreateSettings map[string]jsonReverseAutoCreate      `json:"reverse_auto_create_settings"`
}

type jsonUser struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type jsonSettings struct {
	ID                  int             `json:"id"`
	UserID              int             `json:"user_id"`
	AsanaPAT            string          `json:"asana_pat"`
	YouTrackBaseURL     string          `json:"youtrack_base_url"`
	YouTrackToken       string          `json:"youtrack_token"`
	AsanaProjectID      string          `json:"asana_project_id"`
	YouTrackProjectID   string          `json:"youtrack_project_id"`
	YouTrackBoardID     string          `json:"youtrack_board_id"`
	CustomFieldMappings json.RawMessage `json:"custom_field_mappings"`
	ColumnMappings      json.RawMessage `json:"column_mappings"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type jsonOperation struct {
	ID            int             `json:"id"`
	UserID        int             `json:"user_id"`
	OperationType string          `json:"operation_type"`
	OperationData json.RawMessage `json:"operation_data"`
	Status        string          `json:"status"`
	ErrorMessage  *string         `json:"error_message"`
	CreatedAt     time.Time       `json:"created_at"`
	CompletedAt   *time.Time      `json:"completed_at"`
}

type jsonIgnoredTicket struct {
	ID             int       `json:"id"`
	UserID         int       `json:"user_id"`
	AsanaProjectID string    `json:"asana_project_id"`
	TicketID       string    `json:"ticket_id"`
	IgnoreType     string    `json:"ignore_type"`
	CreatedAt      time.Time `json:"created_at"`
}

type jsonTicketMapping struct {
	ID                int       `json:"id"`
	UserID            int       `json:"user_id"`
	AsanaProjectID    string    `json:"asana_project_id"`
	AsanaTaskID       string    `json:"asana_task_id"`
	YouTrackProjectID string    `json:"youtrack_project_id"`
	YouTrackIssueID   string    `json:"youtrack_issue_id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type jsonSnapshot struct {
	ID           int             `json:"id"`
	OperationID  int             `json:"operation_id"`
	UserID       int             `json:"user_id"`
	SnapshotData json.RawMessage `json:"snapshot_data"`
	CreatedAt    time.Time       `json:"created_at"`
	ExpiresAt    time.Time       `json:"expires_at"`
}

type jsonAuditLog struct {
	ID          int       `json:"id"`
	OperationID int       `json:"operation_id"`
	TicketID    string    `json:"ticket_id"`
	Platform    string    `json:"platform"`
	ActionType  string    `json:"action_type"`
	UserEmail   string    `json:"user_email"`
	OldValue    string    `json:"old_value"`
	NewValue    string    `json:"new_value"`
	FieldName   string    `json:"field_name"`
	Timestamp   time.Time `json:"timestamp"`
}

type jsonReverseIgnored struct {
	ID                int       `json:"id"`
	UserID            int       `json:"user_id"`
	YouTrackProjectID string    `json:"youtrack_project_id"`
	TicketID          string    `json:"ticket_id"`
	IgnoreType        string    `json:"ignore_type"`
	CreatedAt         time.Time `json:"created_at"`
}

type jsonReverseAutoCreate struct {
	ID               int        `json:"id"`
	UserID           int        `json:"user_id"`
	Enabled          bool       `json:"enabled"`
	SelectedCreators string     `json:"selected_creators"`
	IntervalSeconds  int        `json:"interval_seconds"`
	LastRunAt        *time.Time `json:"last_run_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	dataFile := os.Getenv("DATA_FILE")
	if dataFile == "" {
		dataFile = "./sync_app.db_data/data.json"
	}

	// Load JSON
	f, err := os.Open(dataFile)
	if err != nil {
		log.Fatalf("Failed to open data file %s: %v", dataFile, err)
	}
	defer f.Close()

	var data jsonDB
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		log.Fatalf("Failed to decode data.json: %v", err)
	}

	// Connect to Neon
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Cannot reach database: %v", err)
	}
	log.Println("Connected to Neon database")

	// Run schema
	if err := runSchema(ctx, pool); err != nil {
		log.Fatalf("Schema migration failed: %v", err)
	}

	// Migrate in FK-safe order
	migrateUsers(ctx, pool, data.Users)
	migrateSettings(ctx, pool, data.Settings)
	migrateOperations(ctx, pool, data.Operations)
	migrateIgnoredTickets(ctx, pool, data.IgnoredTickets)
	migrateTicketMappings(ctx, pool, data.TicketMappings)
	migrateSnapshots(ctx, pool, data.RollbackSnapshots)
	migrateAuditLogs(ctx, pool, data.AuditLogs)
	migrateReverseIgnored(ctx, pool, data.ReverseIgnoredTickets)
	migrateReverseAutoCreate(ctx, pool, data.ReverseAutoCreateSettings)

	log.Println("Migration complete!")
}

func runSchema(ctx context.Context, pool *pgxpool.Pool) error {
	schema := `
CREATE TABLE IF NOT EXISTS users (
    id            SERIAL PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS user_settings (
    id                    SERIAL PRIMARY KEY,
    user_id               INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    asana_pat             TEXT NOT NULL DEFAULT '',
    youtrack_base_url     TEXT NOT NULL DEFAULT '',
    youtrack_token        TEXT NOT NULL DEFAULT '',
    asana_project_id      TEXT NOT NULL DEFAULT '',
    youtrack_project_id   TEXT NOT NULL DEFAULT '',
    youtrack_board_id     TEXT NOT NULL DEFAULT '',
    custom_field_mappings JSONB NOT NULL DEFAULT '{}',
    column_mappings       JSONB NOT NULL DEFAULT '{}',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS sync_operations (
    id               SERIAL PRIMARY KEY,
    user_id          INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    operation_type   TEXT NOT NULL,
    operation_data   JSONB NOT NULL DEFAULT '{}',
    status           TEXT NOT NULL DEFAULT 'pending',
    error_message    TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at     TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS ignored_tickets (
    id               SERIAL PRIMARY KEY,
    user_id          INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    asana_project_id TEXT NOT NULL,
    ticket_id        TEXT NOT NULL,
    ignore_type      TEXT NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, asana_project_id, ticket_id)
);
CREATE TABLE IF NOT EXISTS ticket_mappings (
    id                   SERIAL PRIMARY KEY,
    user_id              INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    asana_project_id     TEXT NOT NULL,
    asana_task_id        TEXT NOT NULL,
    youtrack_project_id  TEXT NOT NULL,
    youtrack_issue_id    TEXT NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, asana_task_id)
);
CREATE TABLE IF NOT EXISTS rollback_snapshots (
    id            SERIAL PRIMARY KEY,
    operation_id  INTEGER NOT NULL REFERENCES sync_operations(id) ON DELETE CASCADE,
    user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    snapshot_data JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at    TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS audit_logs (
    id            SERIAL PRIMARY KEY,
    operation_id  INTEGER NOT NULL REFERENCES sync_operations(id) ON DELETE CASCADE,
    ticket_id     TEXT NOT NULL,
    platform      TEXT NOT NULL,
    action_type   TEXT NOT NULL,
    user_email    TEXT NOT NULL,
    old_value     TEXT NOT NULL DEFAULT '',
    new_value     TEXT NOT NULL DEFAULT '',
    field_name    TEXT NOT NULL DEFAULT '',
    timestamp     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS reverse_ignored_tickets (
    id                   SERIAL PRIMARY KEY,
    user_id              INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    youtrack_project_id  TEXT NOT NULL,
    ticket_id            TEXT NOT NULL,
    ignore_type          TEXT NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, youtrack_project_id, ticket_id)
);
CREATE TABLE IF NOT EXISTS reverse_auto_create_settings (
    id                 SERIAL PRIMARY KEY,
    user_id            INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE UNIQUE,
    enabled            BOOLEAN NOT NULL DEFAULT FALSE,
    selected_creators  TEXT NOT NULL DEFAULT '',
    interval_seconds   INTEGER NOT NULL DEFAULT 300,
    last_run_at        TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`
	_, err := pool.Exec(ctx, schema)
	if err != nil {
		return fmt.Errorf("schema error: %w", err)
	}
	log.Println("Schema ready")
	return nil
}

func migrateUsers(ctx context.Context, pool *pgxpool.Pool, users map[string]jsonUser) {
	count := 0
	for _, u := range users {
		_, err := pool.Exec(ctx,
			`INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6)
			 ON CONFLICT (username) DO NOTHING`,
			u.ID, u.Username, u.Email, u.PasswordHash, u.CreatedAt, u.UpdatedAt,
		)
		if err != nil {
			log.Printf("  WARN: user %s: %v\n", u.Username, err)
			continue
		}
		count++
	}
	// Reset sequence so future inserts don't collide
	pool.Exec(ctx, `SELECT setval('users_id_seq', (SELECT MAX(id) FROM users))`)
	log.Printf("Users migrated: %d\n", count)
}

func migrateSettings(ctx context.Context, pool *pgxpool.Pool, settings map[string]jsonSettings) {
	count := 0
	for _, s := range settings {
		cfm := s.CustomFieldMappings
		if cfm == nil {
			cfm = json.RawMessage(`{}`)
		}
		cm := s.ColumnMappings
		if cm == nil {
			cm = json.RawMessage(`{}`)
		}
		_, err := pool.Exec(ctx,
			`INSERT INTO user_settings (id, user_id, asana_pat, youtrack_base_url, youtrack_token,
			  asana_project_id, youtrack_project_id, youtrack_board_id,
			  custom_field_mappings, column_mappings, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			 ON CONFLICT (id) DO NOTHING`,
			s.ID, s.UserID, s.AsanaPAT, s.YouTrackBaseURL, s.YouTrackToken,
			s.AsanaProjectID, s.YouTrackProjectID, s.YouTrackBoardID,
			cfm, cm, s.CreatedAt, s.UpdatedAt,
		)
		if err != nil {
			log.Printf("  WARN: settings for user %d: %v\n", s.UserID, err)
			continue
		}
		count++
	}
	pool.Exec(ctx, `SELECT setval('user_settings_id_seq', (SELECT MAX(id) FROM user_settings))`)
	log.Printf("Settings migrated: %d\n", count)
}

func migrateOperations(ctx context.Context, pool *pgxpool.Pool, ops map[string]jsonOperation) {
	count := 0
	for _, op := range ops {
		data := op.OperationData
		if data == nil {
			data = json.RawMessage(`{}`)
		}
		_, err := pool.Exec(ctx,
			`INSERT INTO sync_operations (id, user_id, operation_type, operation_data, status, error_message, created_at, completed_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			 ON CONFLICT (id) DO NOTHING`,
			op.ID, op.UserID, op.OperationType, data, op.Status, op.ErrorMessage, op.CreatedAt, op.CompletedAt,
		)
		if err != nil {
			log.Printf("  WARN: operation %d: %v\n", op.ID, err)
			continue
		}
		count++
	}
	pool.Exec(ctx, `SELECT setval('sync_operations_id_seq', (SELECT MAX(id) FROM sync_operations))`)
	log.Printf("Operations migrated: %d\n", count)
}

func migrateIgnoredTickets(ctx context.Context, pool *pgxpool.Pool, tickets map[string]jsonIgnoredTicket) {
	count := 0
	for _, t := range tickets {
		_, err := pool.Exec(ctx,
			`INSERT INTO ignored_tickets (id, user_id, asana_project_id, ticket_id, ignore_type, created_at)
			 VALUES ($1,$2,$3,$4,$5,$6)
			 ON CONFLICT (user_id, asana_project_id, ticket_id) DO NOTHING`,
			t.ID, t.UserID, t.AsanaProjectID, t.TicketID, t.IgnoreType, t.CreatedAt,
		)
		if err != nil {
			log.Printf("  WARN: ignored ticket %s: %v\n", t.TicketID, err)
			continue
		}
		count++
	}
	pool.Exec(ctx, `SELECT setval('ignored_tickets_id_seq', (SELECT MAX(id) FROM ignored_tickets))`)
	log.Printf("Ignored tickets migrated: %d\n", count)
}

func migrateTicketMappings(ctx context.Context, pool *pgxpool.Pool, mappings map[string]jsonTicketMapping) {
	count := 0
	for _, m := range mappings {
		_, err := pool.Exec(ctx,
			`INSERT INTO ticket_mappings (id, user_id, asana_project_id, asana_task_id, youtrack_project_id, youtrack_issue_id, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			 ON CONFLICT (user_id, asana_task_id) DO UPDATE
			   SET youtrack_project_id=EXCLUDED.youtrack_project_id,
			       youtrack_issue_id=EXCLUDED.youtrack_issue_id,
			       updated_at=EXCLUDED.updated_at`,
			m.ID, m.UserID, m.AsanaProjectID, m.AsanaTaskID, m.YouTrackProjectID, m.YouTrackIssueID, m.CreatedAt, m.UpdatedAt,
		)
		if err != nil {
			log.Printf("  WARN: ticket mapping Asana %s: %v\n", m.AsanaTaskID, err)
			continue
		}
		count++
	}
	pool.Exec(ctx, `SELECT setval('ticket_mappings_id_seq', (SELECT MAX(id) FROM ticket_mappings))`)
	log.Printf("Ticket mappings migrated: %d\n", count)
}

func migrateSnapshots(ctx context.Context, pool *pgxpool.Pool, snapshots map[string]jsonSnapshot) {
	count := 0
	for _, s := range snapshots {
		data := s.SnapshotData
		if data == nil {
			data = json.RawMessage(`{}`)
		}
		_, err := pool.Exec(ctx,
			`INSERT INTO rollback_snapshots (id, operation_id, user_id, snapshot_data, created_at, expires_at)
			 VALUES ($1,$2,$3,$4,$5,$6)
			 ON CONFLICT (id) DO NOTHING`,
			s.ID, s.OperationID, s.UserID, data, s.CreatedAt, s.ExpiresAt,
		)
		if err != nil {
			log.Printf("  WARN: snapshot %d: %v\n", s.ID, err)
			continue
		}
		count++
	}
	if count > 0 {
		pool.Exec(ctx, `SELECT setval('rollback_snapshots_id_seq', (SELECT MAX(id) FROM rollback_snapshots))`)
	}
	log.Printf("Snapshots migrated: %d\n", count)
}

func migrateAuditLogs(ctx context.Context, pool *pgxpool.Pool, logs map[string]jsonAuditLog) {
	count := 0
	for _, l := range logs {
		_, err := pool.Exec(ctx,
			`INSERT INTO audit_logs (id, operation_id, ticket_id, platform, action_type, user_email, old_value, new_value, field_name, timestamp)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			 ON CONFLICT (id) DO NOTHING`,
			l.ID, l.OperationID, l.TicketID, l.Platform, l.ActionType,
			l.UserEmail, l.OldValue, l.NewValue, l.FieldName, l.Timestamp,
		)
		if err != nil {
			log.Printf("  WARN: audit log %d: %v\n", l.ID, err)
			continue
		}
		count++
	}
	if count > 0 {
		pool.Exec(ctx, `SELECT setval('audit_logs_id_seq', (SELECT MAX(id) FROM audit_logs))`)
	}
	log.Printf("Audit logs migrated: %d\n", count)
}

func migrateReverseIgnored(ctx context.Context, pool *pgxpool.Pool, tickets map[string]jsonReverseIgnored) {
	count := 0
	for _, t := range tickets {
		_, err := pool.Exec(ctx,
			`INSERT INTO reverse_ignored_tickets (id, user_id, youtrack_project_id, ticket_id, ignore_type, created_at)
			 VALUES ($1,$2,$3,$4,$5,$6)
			 ON CONFLICT (user_id, youtrack_project_id, ticket_id) DO NOTHING`,
			t.ID, t.UserID, t.YouTrackProjectID, t.TicketID, t.IgnoreType, t.CreatedAt,
		)
		if err != nil {
			log.Printf("  WARN: reverse ignored %s: %v\n", t.TicketID, err)
			continue
		}
		count++
	}
	if count > 0 {
		pool.Exec(ctx, `SELECT setval('reverse_ignored_tickets_id_seq', (SELECT MAX(id) FROM reverse_ignored_tickets))`)
	}
	log.Printf("Reverse ignored tickets migrated: %d\n", count)
}

func migrateReverseAutoCreate(ctx context.Context, pool *pgxpool.Pool, settings map[string]jsonReverseAutoCreate) {
	count := 0
	for _, s := range settings {
		_, err := pool.Exec(ctx,
			`INSERT INTO reverse_auto_create_settings (id, user_id, enabled, selected_creators, interval_seconds, last_run_at, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			 ON CONFLICT (user_id) DO NOTHING`,
			s.ID, s.UserID, s.Enabled, s.SelectedCreators, s.IntervalSeconds, s.LastRunAt, s.CreatedAt, s.UpdatedAt,
		)
		if err != nil {
			log.Printf("  WARN: reverse auto-create for user %d: %v\n", s.UserID, err)
			continue
		}
		count++
	}
	if count > 0 {
		pool.Exec(ctx, `SELECT setval('reverse_auto_create_settings_id_seq', (SELECT MAX(id) FROM reverse_auto_create_settings))`)
	}
	log.Printf("Reverse auto-create settings migrated: %d\n", count)
}
