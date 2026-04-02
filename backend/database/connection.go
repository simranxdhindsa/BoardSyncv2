package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a pgxpool connection pool.
// All methods keep the same signatures as the old JSON-file implementation
// so the rest of the codebase requires no changes.
type DB struct {
	pool  *pgxpool.Pool
	mutex sync.RWMutex // kept for any in-memory helpers
}

var database *DB

// InitDB connects to PostgreSQL (Neon).
// dbPath is ignored when DATABASE_URL is set; it is kept as a parameter
// so main.go needs no signature change.
func InitDB(dbPath string) (*DB, error) {
	connStr := getEnvDefault("DATABASE_URL", "")
	if connStr == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required for PostgreSQL/Neon connection")
	}

	ctx := context.Background()
	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to connect to Neon database: %w", err)
	}

	database = &DB{pool: pool}

	if err := database.runMigrations(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to run schema migrations: %w", err)
	}

	log.Println("PostgreSQL/Neon database initialized successfully")
	return database, nil
}

// GetDB returns the singleton database instance.
func GetDB() *DB {
	return database
}

// Close closes the connection pool.
func (db *DB) Close() error {
	db.pool.Close()
	return nil
}

// runMigrations creates all tables if they don't exist.
func (db *DB) runMigrations(ctx context.Context) error {
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
	_, err := db.pool.Exec(ctx, schema)
	return err
}

// ─── User Operations ─────────────────────────────────────────────────────────

func (db *DB) CreateUser(username, email, passwordHash string) (*User, error) {
	ctx := context.Background()

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	user := &User{}
	err = tx.QueryRow(ctx,
		`INSERT INTO users (username, email, password_hash, created_at, updated_at)
		 VALUES ($1, $2, $3, NOW(), NOW())
		 RETURNING id, username, email, password_hash, created_at, updated_at`,
		username, email, passwordHash,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("user already exists or db error: %w", err)
	}

	// Create default settings
	cfm := CustomFieldMappings{
		TagMapping:      make(map[string]string),
		PriorityMapping: make(map[string]string),
		StatusMapping:   make(map[string]string),
		CustomFields:    make(map[string]string),
	}
	cfmJSON, _ := json.Marshal(cfm)
	cmJSON, _ := json.Marshal(ColumnMappings{AsanaToYouTrack: []ColumnMapping{}, YouTrackToAsana: []ColumnMapping{}})

	_, err = tx.Exec(ctx,
		`INSERT INTO user_settings (user_id, custom_field_mappings, column_mappings, created_at, updated_at)
		 VALUES ($1, $2, $3, NOW(), NOW())`,
		user.ID, cfmJSON, cmJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create default settings: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	log.Printf("DB: User created: %s (ID: %d)\n", username, user.ID)
	return user, nil
}

func (db *DB) GetUserByUsername(username string) (*User, error) {
	ctx := context.Background()
	user := &User{}
	err := db.pool.QueryRow(ctx,
		`SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE username = $1`,
		username,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

func (db *DB) GetUserByEmail(email string) (*User, error) {
	ctx := context.Background()
	user := &User{}
	err := db.pool.QueryRow(ctx,
		`SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

func (db *DB) GetUserByID(id int) (*User, error) {
	ctx := context.Background()
	user := &User{}
	err := db.pool.QueryRow(ctx,
		`SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

func (db *DB) UpdateUserPassword(userID int, passwordHash string) error {
	ctx := context.Background()
	_, err := db.pool.Exec(ctx,
		`UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`,
		passwordHash, userID,
	)
	return err
}

func (db *DB) DeleteUser(userID int) error {
	ctx := context.Background()
	// Cascades handle related rows via FK ON DELETE CASCADE
	_, err := db.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	log.Printf("DB: Deleted user ID %d and all associated data\n", userID)
	return nil
}

func (db *DB) GetUserDataSummary(userID int) (map[string]int, error) {
	ctx := context.Background()
	summary := map[string]int{
		"settings":        0,
		"operations":      0,
		"ignored_tickets": 0,
		"ticket_mappings": 0,
	}

	tables := map[string]string{
		"settings":        "user_settings",
		"operations":      "sync_operations",
		"ignored_tickets": "ignored_tickets",
		"ticket_mappings": "ticket_mappings",
	}
	for key, table := range tables {
		var count int
		db.pool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE user_id = $1`, table), userID).Scan(&count)
		summary[key] = count
	}
	return summary, nil
}

// ─── Settings Operations ──────────────────────────────────────────────────────

func (db *DB) GetUserSettings(userID int) (*UserSettings, error) {
	ctx := context.Background()
	s := &UserSettings{}
	var cfmJSON, cmJSON []byte
	err := db.pool.QueryRow(ctx,
		`SELECT id, user_id, asana_pat, youtrack_base_url, youtrack_token,
		        asana_project_id, youtrack_project_id, youtrack_board_id,
		        custom_field_mappings, column_mappings, created_at, updated_at
		 FROM user_settings WHERE user_id = $1`,
		userID,
	).Scan(&s.ID, &s.UserID, &s.AsanaPAT, &s.YouTrackBaseURL, &s.YouTrackToken,
		&s.AsanaProjectID, &s.YouTrackProjectID, &s.YouTrackBoardID,
		&cfmJSON, &cmJSON, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("settings not found")
	}
	json.Unmarshal(cfmJSON, &s.CustomFieldMappings)
	json.Unmarshal(cmJSON, &s.ColumnMappings)
	return s, nil
}

func (db *DB) UpdateUserSettings(userID int, asanaPAT, youtrackBaseURL, youtrackToken, asanaProjectID, youtrackProjectID, youtrackBoardID string, mappings CustomFieldMappings, columnMappings ColumnMappings) (*UserSettings, error) {
	ctx := context.Background()
	cfmJSON, _ := json.Marshal(mappings)
	cmJSON, _ := json.Marshal(columnMappings)

	s := &UserSettings{}
	var cfmOut, cmOut []byte
	err := db.pool.QueryRow(ctx,
		`UPDATE user_settings
		 SET asana_pat=$1, youtrack_base_url=$2, youtrack_token=$3,
		     asana_project_id=$4, youtrack_project_id=$5, youtrack_board_id=$6,
		     custom_field_mappings=$7, column_mappings=$8, updated_at=NOW()
		 WHERE user_id=$9
		 RETURNING id, user_id, asana_pat, youtrack_base_url, youtrack_token,
		           asana_project_id, youtrack_project_id, youtrack_board_id,
		           custom_field_mappings, column_mappings, created_at, updated_at`,
		asanaPAT, youtrackBaseURL, youtrackToken,
		asanaProjectID, youtrackProjectID, youtrackBoardID,
		cfmJSON, cmJSON, userID,
	).Scan(&s.ID, &s.UserID, &s.AsanaPAT, &s.YouTrackBaseURL, &s.YouTrackToken,
		&s.AsanaProjectID, &s.YouTrackProjectID, &s.YouTrackBoardID,
		&cfmOut, &cmOut, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("settings not found")
	}
	json.Unmarshal(cfmOut, &s.CustomFieldMappings)
	json.Unmarshal(cmOut, &s.ColumnMappings)
	return s, nil
}

// ─── Operation Operations ─────────────────────────────────────────────────────

func (db *DB) CreateOperation(userID int, operationType string, operationData map[string]interface{}) (*SyncOperation, error) {
	ctx := context.Background()
	dataJSON, _ := json.Marshal(operationData)

	op := &SyncOperation{}
	var dataOut []byte
	err := db.pool.QueryRow(ctx,
		`INSERT INTO sync_operations (user_id, operation_type, operation_data, status, created_at)
		 VALUES ($1, $2, $3, 'pending', NOW())
		 RETURNING id, user_id, operation_type, operation_data, status, error_message, created_at, completed_at`,
		userID, operationType, dataJSON,
	).Scan(&op.ID, &op.UserID, &op.OperationType, &dataOut, &op.Status, &op.ErrorMessage, &op.CreatedAt, &op.CompletedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(dataOut, &op.OperationData)
	return op, nil
}

func (db *DB) GetOperation(operationID int) (*SyncOperation, error) {
	ctx := context.Background()
	op := &SyncOperation{}
	var dataOut []byte
	err := db.pool.QueryRow(ctx,
		`SELECT id, user_id, operation_type, operation_data, status, error_message, created_at, completed_at
		 FROM sync_operations WHERE id = $1`,
		operationID,
	).Scan(&op.ID, &op.UserID, &op.OperationType, &dataOut, &op.Status, &op.ErrorMessage, &op.CreatedAt, &op.CompletedAt)
	if err != nil {
		return nil, fmt.Errorf("operation not found")
	}
	json.Unmarshal(dataOut, &op.OperationData)
	return op, nil
}

func (db *DB) UpdateOperationStatus(operationID int, status string, errorMessage *string) error {
	ctx := context.Background()
	var completedAt *time.Time
	if status == "completed" || status == "failed" || status == "rolled_back" {
		now := time.Now()
		completedAt = &now
	}
	_, err := db.pool.Exec(ctx,
		`UPDATE sync_operations SET status=$1, error_message=$2, completed_at=$3 WHERE id=$4`,
		status, errorMessage, completedAt, operationID,
	)
	return err
}

func (db *DB) GetUserOperations(userID int, limit int) ([]*SyncOperation, error) {
	ctx := context.Background()
	rows, err := db.pool.Query(ctx,
		`SELECT id, user_id, operation_type, operation_data, status, error_message, created_at, completed_at
		 FROM sync_operations WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ops []*SyncOperation
	for rows.Next() {
		op := &SyncOperation{}
		var dataOut []byte
		if err := rows.Scan(&op.ID, &op.UserID, &op.OperationType, &dataOut, &op.Status, &op.ErrorMessage, &op.CreatedAt, &op.CompletedAt); err != nil {
			continue
		}
		json.Unmarshal(dataOut, &op.OperationData)
		ops = append(ops, op)
	}
	return ops, nil
}

// ─── Ignored Ticket Operations ────────────────────────────────────────────────

func (db *DB) AddIgnoredTicket(userID int, asanaProjectID, ticketID, ignoreType string) (*IgnoredTicket, error) {
	ctx := context.Background()
	t := &IgnoredTicket{}
	err := db.pool.QueryRow(ctx,
		`INSERT INTO ignored_tickets (user_id, asana_project_id, ticket_id, ignore_type, created_at)
		 VALUES ($1, $2, $3, $4, NOW())
		 ON CONFLICT (user_id, asana_project_id, ticket_id) DO UPDATE
		   SET ignore_type=EXCLUDED.ignore_type, created_at=NOW()
		 RETURNING id, user_id, asana_project_id, ticket_id, ignore_type, created_at`,
		userID, asanaProjectID, ticketID, ignoreType,
	).Scan(&t.ID, &t.UserID, &t.AsanaProjectID, &t.TicketID, &t.IgnoreType, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (db *DB) RemoveIgnoredTicket(userID int, asanaProjectID, ticketID, ignoreType string) error {
	ctx := context.Background()
	query := `DELETE FROM ignored_tickets WHERE user_id=$1 AND asana_project_id=$2 AND ticket_id=$3`
	args := []interface{}{userID, asanaProjectID, ticketID}
	if ignoreType != "" {
		query += ` AND ignore_type=$4`
		args = append(args, ignoreType)
	}
	_, err := db.pool.Exec(ctx, query, args...)
	return err
}

func (db *DB) GetIgnoredTickets(userID int, asanaProjectID string) ([]*IgnoredTicket, error) {
	ctx := context.Background()
	rows, err := db.pool.Query(ctx,
		`SELECT id, user_id, asana_project_id, ticket_id, ignore_type, created_at
		 FROM ignored_tickets WHERE user_id=$1 AND asana_project_id=$2`,
		userID, asanaProjectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []*IgnoredTicket
	for rows.Next() {
		t := &IgnoredTicket{}
		if err := rows.Scan(&t.ID, &t.UserID, &t.AsanaProjectID, &t.TicketID, &t.IgnoreType, &t.CreatedAt); err != nil {
			continue
		}
		tickets = append(tickets, t)
	}
	return tickets, nil
}

func (db *DB) IsTicketIgnored(userID int, asanaProjectID, ticketID string) (bool, string) {
	ctx := context.Background()
	var ignoreType string
	err := db.pool.QueryRow(ctx,
		`SELECT ignore_type FROM ignored_tickets WHERE user_id=$1 AND asana_project_id=$2 AND ticket_id=$3`,
		userID, asanaProjectID, ticketID,
	).Scan(&ignoreType)
	if err != nil {
		return false, ""
	}
	return true, ignoreType
}

func (db *DB) ClearIgnoredTickets(userID int, asanaProjectID, ignoreType string) error {
	ctx := context.Background()
	if ignoreType != "" {
		_, err := db.pool.Exec(ctx,
			`DELETE FROM ignored_tickets WHERE user_id=$1 AND asana_project_id=$2 AND ignore_type=$3`,
			userID, asanaProjectID, ignoreType,
		)
		return err
	}
	_, err := db.pool.Exec(ctx,
		`DELETE FROM ignored_tickets WHERE user_id=$1 AND asana_project_id=$2`,
		userID, asanaProjectID,
	)
	return err
}

// ─── Ticket Mapping Operations ────────────────────────────────────────────────

func (db *DB) CreateTicketMapping(userID int, asanaProjectID, asanaTaskID, youtrackProjectID, youtrackIssueID string) (*TicketMapping, error) {
	ctx := context.Background()
	m := &TicketMapping{}
	err := db.pool.QueryRow(ctx,
		`INSERT INTO ticket_mappings (user_id, asana_project_id, asana_task_id, youtrack_project_id, youtrack_issue_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		 ON CONFLICT (user_id, asana_task_id) DO UPDATE
		   SET youtrack_project_id=EXCLUDED.youtrack_project_id,
		       youtrack_issue_id=EXCLUDED.youtrack_issue_id,
		       updated_at=NOW()
		 RETURNING id, user_id, asana_project_id, asana_task_id, youtrack_project_id, youtrack_issue_id, created_at, updated_at`,
		userID, asanaProjectID, asanaTaskID, youtrackProjectID, youtrackIssueID,
	).Scan(&m.ID, &m.UserID, &m.AsanaProjectID, &m.AsanaTaskID, &m.YouTrackProjectID, &m.YouTrackIssueID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	log.Printf("DB: Upserted ticket mapping: Asana %s <-> YouTrack %s for user %d\n", asanaTaskID, youtrackIssueID, userID)
	return m, nil
}

func (db *DB) GetTicketMappingByAsanaID(userID int, asanaTaskID string) (*TicketMapping, error) {
	ctx := context.Background()
	m := &TicketMapping{}
	err := db.pool.QueryRow(ctx,
		`SELECT id, user_id, asana_project_id, asana_task_id, youtrack_project_id, youtrack_issue_id, created_at, updated_at
		 FROM ticket_mappings WHERE user_id=$1 AND asana_task_id=$2`,
		userID, asanaTaskID,
	).Scan(&m.ID, &m.UserID, &m.AsanaProjectID, &m.AsanaTaskID, &m.YouTrackProjectID, &m.YouTrackIssueID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("mapping not found for Asana task %s", asanaTaskID)
	}
	return m, nil
}

func (db *DB) GetTicketMappingByYouTrackID(userID int, youtrackIssueID string) (*TicketMapping, error) {
	ctx := context.Background()
	m := &TicketMapping{}
	err := db.pool.QueryRow(ctx,
		`SELECT id, user_id, asana_project_id, asana_task_id, youtrack_project_id, youtrack_issue_id, created_at, updated_at
		 FROM ticket_mappings WHERE user_id=$1 AND youtrack_issue_id=$2`,
		userID, youtrackIssueID,
	).Scan(&m.ID, &m.UserID, &m.AsanaProjectID, &m.AsanaTaskID, &m.YouTrackProjectID, &m.YouTrackIssueID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("mapping not found for YouTrack issue %s", youtrackIssueID)
	}
	return m, nil
}

func (db *DB) GetAllTicketMappings(userID int) ([]*TicketMapping, error) {
	ctx := context.Background()
	rows, err := db.pool.Query(ctx,
		`SELECT id, user_id, asana_project_id, asana_task_id, youtrack_project_id, youtrack_issue_id, created_at, updated_at
		 FROM ticket_mappings WHERE user_id=$1`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mappings []*TicketMapping
	for rows.Next() {
		m := &TicketMapping{}
		if err := rows.Scan(&m.ID, &m.UserID, &m.AsanaProjectID, &m.AsanaTaskID, &m.YouTrackProjectID, &m.YouTrackIssueID, &m.CreatedAt, &m.UpdatedAt); err != nil {
			continue
		}
		mappings = append(mappings, m)
	}
	return mappings, nil
}

func (db *DB) DeleteTicketMapping(userID, mappingID int) error {
	ctx := context.Background()
	result, err := db.pool.Exec(ctx,
		`DELETE FROM ticket_mappings WHERE id=$1 AND user_id=$2`,
		mappingID, userID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("mapping not found or access denied")
	}
	return nil
}

func (db *DB) HasTicketMapping(userID int, asanaTaskID, youtrackIssueID string) bool {
	ctx := context.Background()
	var exists bool
	db.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM ticket_mappings WHERE user_id=$1 AND asana_task_id=$2 AND youtrack_issue_id=$3)`,
		userID, asanaTaskID, youtrackIssueID,
	).Scan(&exists)
	return exists
}

// ─── Reverse Ignored Ticket Operations ───────────────────────────────────────

func (db *DB) AddReverseIgnoredTicket(userID int, youtrackProjectID, ticketID, ignoreType string) (*ReverseIgnoredTicket, error) {
	ctx := context.Background()
	t := &ReverseIgnoredTicket{}
	err := db.pool.QueryRow(ctx,
		`INSERT INTO reverse_ignored_tickets (user_id, youtrack_project_id, ticket_id, ignore_type, created_at)
		 VALUES ($1, $2, $3, $4, NOW())
		 ON CONFLICT (user_id, youtrack_project_id, ticket_id) DO UPDATE
		   SET ignore_type=EXCLUDED.ignore_type
		 RETURNING id, user_id, youtrack_project_id, ticket_id, ignore_type, created_at`,
		userID, youtrackProjectID, ticketID, ignoreType,
	).Scan(&t.ID, &t.UserID, &t.YouTrackProjectID, &t.TicketID, &t.IgnoreType, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (db *DB) RemoveReverseIgnoredTicket(userID int, youtrackProjectID, ticketID, ignoreType string) error {
	ctx := context.Background()
	query := `DELETE FROM reverse_ignored_tickets WHERE user_id=$1 AND youtrack_project_id=$2 AND ticket_id=$3`
	args := []interface{}{userID, youtrackProjectID, ticketID}
	if ignoreType != "" {
		query += ` AND ignore_type=$4`
		args = append(args, ignoreType)
	}
	_, err := db.pool.Exec(ctx, query, args...)
	return err
}

func (db *DB) GetReverseIgnoredTickets(userID int, youtrackProjectID string) ([]*ReverseIgnoredTicket, error) {
	ctx := context.Background()
	rows, err := db.pool.Query(ctx,
		`SELECT id, user_id, youtrack_project_id, ticket_id, ignore_type, created_at
		 FROM reverse_ignored_tickets WHERE user_id=$1 AND youtrack_project_id=$2`,
		userID, youtrackProjectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []*ReverseIgnoredTicket
	for rows.Next() {
		t := &ReverseIgnoredTicket{}
		if err := rows.Scan(&t.ID, &t.UserID, &t.YouTrackProjectID, &t.TicketID, &t.IgnoreType, &t.CreatedAt); err != nil {
			continue
		}
		tickets = append(tickets, t)
	}
	return tickets, nil
}

func (db *DB) IsReverseTicketIgnored(userID int, youtrackProjectID, ticketID string) (bool, string) {
	ctx := context.Background()
	var ignoreType string
	err := db.pool.QueryRow(ctx,
		`SELECT ignore_type FROM reverse_ignored_tickets WHERE user_id=$1 AND youtrack_project_id=$2 AND ticket_id=$3`,
		userID, youtrackProjectID, ticketID,
	).Scan(&ignoreType)
	if err != nil {
		return false, ""
	}
	return true, ignoreType
}

func (db *DB) ClearReverseIgnoredTickets(userID int, youtrackProjectID, ignoreType string) error {
	ctx := context.Background()
	if ignoreType != "" {
		_, err := db.pool.Exec(ctx,
			`DELETE FROM reverse_ignored_tickets WHERE user_id=$1 AND youtrack_project_id=$2 AND ignore_type=$3`,
			userID, youtrackProjectID, ignoreType,
		)
		return err
	}
	_, err := db.pool.Exec(ctx,
		`DELETE FROM reverse_ignored_tickets WHERE user_id=$1 AND youtrack_project_id=$2`,
		userID, youtrackProjectID,
	)
	return err
}

// ─── Reverse Auto-Create Settings Operations ──────────────────────────────────

func (db *DB) GetReverseAutoCreateSettings(userID int) (*ReverseAutoCreateSettings, error) {
	ctx := context.Background()
	s := &ReverseAutoCreateSettings{}
	err := db.pool.QueryRow(ctx,
		`SELECT id, user_id, enabled, selected_creators, interval_seconds, last_run_at, created_at, updated_at
		 FROM reverse_auto_create_settings WHERE user_id=$1`,
		userID,
	).Scan(&s.ID, &s.UserID, &s.Enabled, &s.SelectedCreators, &s.IntervalSeconds, &s.LastRunAt, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, nil // not found is valid — caller handles nil
	}
	return s, nil
}

func (db *DB) UpsertReverseAutoCreateSettings(userID int, enabled bool, selectedCreators string, intervalSeconds int) (*ReverseAutoCreateSettings, error) {
	ctx := context.Background()
	s := &ReverseAutoCreateSettings{}
	err := db.pool.QueryRow(ctx,
		`INSERT INTO reverse_auto_create_settings (user_id, enabled, selected_creators, interval_seconds, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, NOW(), NOW())
		 ON CONFLICT (user_id) DO UPDATE
		   SET enabled=$2, selected_creators=$3, interval_seconds=$4, updated_at=NOW()
		 RETURNING id, user_id, enabled, selected_creators, interval_seconds, last_run_at, created_at, updated_at`,
		userID, enabled, selectedCreators, intervalSeconds,
	).Scan(&s.ID, &s.UserID, &s.Enabled, &s.SelectedCreators, &s.IntervalSeconds, &s.LastRunAt, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (db *DB) UpdateReverseAutoCreateLastRun(userID int, lastRunAt time.Time) error {
	ctx := context.Background()
	_, err := db.pool.Exec(ctx,
		`UPDATE reverse_auto_create_settings SET last_run_at=$1, updated_at=NOW() WHERE user_id=$2`,
		lastRunAt, userID,
	)
	return err
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func getEnvDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
