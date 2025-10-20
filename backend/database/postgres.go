package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// PostgresDB wraps the SQL database connection
type PostgresDB struct {
	conn *sql.DB
}

var postgresDB *PostgresDB

// InitPostgres initializes PostgreSQL database connection
func InitPostgres() (*PostgresDB, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is not set")
	}

	log.Println("Connecting to PostgreSQL database...")

	// Open database connection
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	postgresDB = &PostgresDB{conn: conn}

	log.Println("PostgreSQL database connected successfully")

	// Run migrations
	if err := postgresDB.runMigrations(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return postgresDB, nil
}

// GetPostgresDB returns the PostgreSQL database instance
func GetPostgresDB() *PostgresDB {
	return postgresDB
}

// Close closes the database connection
func (db *PostgresDB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// runMigrations creates all necessary tables
func (db *PostgresDB) runMigrations() error {
	log.Println("Running database migrations...")

	// Read schema file
	schemaPath := "database/schema.sql"
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		// If file doesn't exist, use embedded schema
		log.Printf("Warning: Could not read %s, using embedded schema\n", schemaPath)
		schema = []byte(embeddedSchema)
	}

	// Execute schema
	_, err = db.conn.Exec(string(schema))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}

// embeddedSchema contains the SQL schema as a fallback
const embeddedSchema = `
-- BoardSync PostgreSQL Schema

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

CREATE TABLE IF NOT EXISTS user_settings (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    asana_pat TEXT,
    youtrack_base_url TEXT,
    youtrack_token TEXT,
    asana_project_id VARCHAR(255),
    youtrack_project_id VARCHAR(255),
    youtrack_board_id VARCHAR(255),
    custom_field_mappings JSONB DEFAULT '{}',
    column_mappings JSONB DEFAULT '{"asana_to_youtrack":[],"youtrack_to_asana":[]}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id)
);

CREATE INDEX IF NOT EXISTS idx_user_settings_user_id ON user_settings(user_id);

CREATE TABLE IF NOT EXISTS ticket_mappings (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    asana_project_id VARCHAR(255) NOT NULL,
    asana_task_id VARCHAR(255) NOT NULL,
    youtrack_project_id VARCHAR(255) NOT NULL,
    youtrack_issue_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, asana_task_id)
);

CREATE INDEX IF NOT EXISTS idx_ticket_mappings_user_id ON ticket_mappings(user_id);
CREATE INDEX IF NOT EXISTS idx_ticket_mappings_asana_task ON ticket_mappings(asana_task_id);
CREATE INDEX IF NOT EXISTS idx_ticket_mappings_youtrack_issue ON ticket_mappings(youtrack_issue_id);

CREATE TABLE IF NOT EXISTS ignored_tickets (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    asana_project_id VARCHAR(255) NOT NULL,
    ticket_id VARCHAR(255) NOT NULL,
    ignore_type VARCHAR(50) NOT NULL CHECK (ignore_type IN ('temp', 'forever')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, asana_project_id, ticket_id)
);

CREATE INDEX IF NOT EXISTS idx_ignored_tickets_user_id ON ignored_tickets(user_id);
CREATE INDEX IF NOT EXISTS idx_ignored_tickets_ticket_id ON ignored_tickets(ticket_id);

CREATE TABLE IF NOT EXISTS sync_operations (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    operation_type VARCHAR(100) NOT NULL,
    operation_data JSONB DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sync_operations_user_id ON sync_operations(user_id);
CREATE INDEX IF NOT EXISTS idx_sync_operations_status ON sync_operations(status);
CREATE INDEX IF NOT EXISTS idx_sync_operations_created_at ON sync_operations(created_at DESC);

CREATE TABLE IF NOT EXISTS rollback_snapshots (
    id SERIAL PRIMARY KEY,
    operation_id INTEGER NOT NULL REFERENCES sync_operations(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    snapshot_data JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP + INTERVAL '30 days'),
    UNIQUE(operation_id)
);

CREATE INDEX IF NOT EXISTS idx_rollback_snapshots_operation_id ON rollback_snapshots(operation_id);
CREATE INDEX IF NOT EXISTS idx_rollback_snapshots_user_id ON rollback_snapshots(user_id);
CREATE INDEX IF NOT EXISTS idx_rollback_snapshots_expires_at ON rollback_snapshots(expires_at);

CREATE TABLE IF NOT EXISTS audit_logs (
    id SERIAL PRIMARY KEY,
    operation_id INTEGER REFERENCES sync_operations(id) ON DELETE SET NULL,
    ticket_id VARCHAR(255) NOT NULL,
    platform VARCHAR(50) NOT NULL,
    action_type VARCHAR(50) NOT NULL,
    user_email VARCHAR(255) NOT NULL,
    old_value TEXT,
    new_value TEXT,
    field_name VARCHAR(255),
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_operation_id ON audit_logs(operation_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_ticket_id ON audit_logs(ticket_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_platform ON audit_logs(platform);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action_type ON audit_logs(action_type);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_email ON audit_logs(user_email);
CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp DESC);

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER IF NOT EXISTS update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER IF NOT EXISTS update_user_settings_updated_at
    BEFORE UPDATE ON user_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER IF NOT EXISTS update_ticket_mappings_updated_at
    BEFORE UPDATE ON ticket_mappings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
`

// =============================================================================
// USER OPERATIONS
// =============================================================================

func (db *PostgresDB) CreateUser(username, email, passwordHash string) (*User, error) {
	query := `
		INSERT INTO users (username, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, username, email, password_hash, created_at, updated_at
	`

	now := time.Now()
	user := &User{}

	err := db.conn.QueryRow(query, username, email, passwordHash, now, now).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	log.Printf("DB: Created user: %s (ID: %d)\n", username, user.ID)

	// Create default settings
	_, err = db.CreateUserSettings(user.ID)
	if err != nil {
		log.Printf("Warning: Failed to create default settings for user %d: %v\n", user.ID, err)
	}

	return user, nil
}

func (db *PostgresDB) GetUserByEmail(email string) (*User, error) {
	query := `SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE email = $1`

	user := &User{}
	err := db.conn.QueryRow(query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (db *PostgresDB) GetUserByID(id int) (*User, error) {
	query := `SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE id = $1`

	user := &User{}
	err := db.conn.QueryRow(query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (db *PostgresDB) UpdateUserPassword(userID int, passwordHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`

	_, err := db.conn.Exec(query, passwordHash, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	log.Printf("DB: Updated password for user ID: %d\n", userID)
	return nil
}

func (db *PostgresDB) DeleteUser(userID int) error {
	query := `DELETE FROM users WHERE id = $1`

	_, err := db.conn.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	log.Printf("DB: Deleted user ID: %d\n", userID)
	return nil
}

// =============================================================================
// USER SETTINGS OPERATIONS
// =============================================================================

func (db *PostgresDB) CreateUserSettings(userID int) (*UserSettings, error) {
	query := `
		INSERT INTO user_settings (
			user_id, asana_pat, youtrack_base_url, youtrack_token,
			asana_project_id, youtrack_project_id, youtrack_board_id,
			custom_field_mappings, column_mappings, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, user_id, asana_pat, youtrack_base_url, youtrack_token,
				  asana_project_id, youtrack_project_id, youtrack_board_id,
				  custom_field_mappings, column_mappings, created_at, updated_at
	`

	now := time.Now()
	settings := &UserSettings{
		UserID:              userID,
		CustomFieldMappings: CustomFieldMappings{},
		ColumnMappings: ColumnMappings{
			AsanaToYouTrack: []ColumnMapping{},
			YouTrackToAsana: []ColumnMapping{},
		},
	}

	err := db.conn.QueryRow(
		query, userID, "", "", "", "", "", "",
		settings.CustomFieldMappings, settings.ColumnMappings, now, now,
	).Scan(
		&settings.ID,
		&settings.UserID,
		&settings.AsanaPAT,
		&settings.YouTrackBaseURL,
		&settings.YouTrackToken,
		&settings.AsanaProjectID,
		&settings.YouTrackProjectID,
		&settings.YouTrackBoardID,
		&settings.CustomFieldMappings,
		&settings.ColumnMappings,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create user settings: %w", err)
	}

	log.Printf("DB: Created settings for user ID: %d\n", userID)
	return settings, nil
}

func (db *PostgresDB) GetUserSettings(userID int) (*UserSettings, error) {
	query := `
		SELECT id, user_id, asana_pat, youtrack_base_url, youtrack_token,
			   asana_project_id, youtrack_project_id, youtrack_board_id,
			   custom_field_mappings, column_mappings, created_at, updated_at
		FROM user_settings
		WHERE user_id = $1
	`

	settings := &UserSettings{}
	err := db.conn.QueryRow(query, userID).Scan(
		&settings.ID,
		&settings.UserID,
		&settings.AsanaPAT,
		&settings.YouTrackBaseURL,
		&settings.YouTrackToken,
		&settings.AsanaProjectID,
		&settings.YouTrackProjectID,
		&settings.YouTrackBoardID,
		&settings.CustomFieldMappings,
		&settings.ColumnMappings,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("settings not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	return settings, nil
}

func (db *PostgresDB) UpdateUserSettings(settings *UserSettings) error {
	query := `
		UPDATE user_settings
		SET asana_pat = $1, youtrack_base_url = $2, youtrack_token = $3,
			asana_project_id = $4, youtrack_project_id = $5, youtrack_board_id = $6,
			custom_field_mappings = $7, column_mappings = $8, updated_at = $9
		WHERE user_id = $10
	`

	_, err := db.conn.Exec(
		query,
		settings.AsanaPAT,
		settings.YouTrackBaseURL,
		settings.YouTrackToken,
		settings.AsanaProjectID,
		settings.YouTrackProjectID,
		settings.YouTrackBoardID,
		settings.CustomFieldMappings,
		settings.ColumnMappings,
		time.Now(),
		settings.UserID,
	)

	if err != nil {
		return fmt.Errorf("failed to update settings: %w", err)
	}

	log.Printf("DB: Updated settings for user ID: %d\n", settings.UserID)
	return nil
}

// =============================================================================
// TICKET MAPPING OPERATIONS
// =============================================================================

func (db *PostgresDB) CreateTicketMapping(userID int, asanaProjectID, asanaTaskID, youtrackProjectID, youtrackIssueID string) (*TicketMapping, error) {
	// Check if exact mapping already exists
	existingQuery := `
		SELECT id FROM ticket_mappings
		WHERE user_id = $1 AND asana_task_id = $2 AND youtrack_issue_id = $3
	`
	var existingID int
	err := db.conn.QueryRow(existingQuery, userID, asanaTaskID, youtrackIssueID).Scan(&existingID)
	if err == nil {
		// Mapping already exists, fetch and return it
		return db.GetTicketMappingByID(existingID)
	}

	// Check if mapping exists for this Asana task with different YouTrack ID
	updateQuery := `
		UPDATE ticket_mappings
		SET youtrack_project_id = $1, youtrack_issue_id = $2, updated_at = $3
		WHERE user_id = $4 AND asana_task_id = $5
		RETURNING id, user_id, asana_project_id, asana_task_id, youtrack_project_id, youtrack_issue_id, created_at, updated_at
	`

	mapping := &TicketMapping{}
	err = db.conn.QueryRow(updateQuery, youtrackProjectID, youtrackIssueID, time.Now(), userID, asanaTaskID).Scan(
		&mapping.ID,
		&mapping.UserID,
		&mapping.AsanaProjectID,
		&mapping.AsanaTaskID,
		&mapping.YouTrackProjectID,
		&mapping.YouTrackIssueID,
		&mapping.CreatedAt,
		&mapping.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// No existing mapping, create new one
		insertQuery := `
			INSERT INTO ticket_mappings (user_id, asana_project_id, asana_task_id, youtrack_project_id, youtrack_issue_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id, user_id, asana_project_id, asana_task_id, youtrack_project_id, youtrack_issue_id, created_at, updated_at
		`

		now := time.Now()
		err = db.conn.QueryRow(insertQuery, userID, asanaProjectID, asanaTaskID, youtrackProjectID, youtrackIssueID, now, now).Scan(
			&mapping.ID,
			&mapping.UserID,
			&mapping.AsanaProjectID,
			&mapping.AsanaTaskID,
			&mapping.YouTrackProjectID,
			&mapping.YouTrackIssueID,
			&mapping.CreatedAt,
			&mapping.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to create ticket mapping: %w", err)
		}

		log.Printf("DB: Created new mapping: Asana %s <-> YouTrack %s\n", asanaTaskID, youtrackIssueID)
	} else if err != nil {
		return nil, fmt.Errorf("failed to update ticket mapping: %w", err)
	} else {
		log.Printf("DB: Updated mapping: Asana %s <-> YouTrack %s (was different YouTrack ID)\n", asanaTaskID, youtrackIssueID)
	}

	return mapping, nil
}

func (db *PostgresDB) GetTicketMappingByID(id int) (*TicketMapping, error) {
	query := `
		SELECT id, user_id, asana_project_id, asana_task_id, youtrack_project_id, youtrack_issue_id, created_at, updated_at
		FROM ticket_mappings
		WHERE id = $1
	`

	mapping := &TicketMapping{}
	err := db.conn.QueryRow(query, id).Scan(
		&mapping.ID,
		&mapping.UserID,
		&mapping.AsanaProjectID,
		&mapping.AsanaTaskID,
		&mapping.YouTrackProjectID,
		&mapping.YouTrackIssueID,
		&mapping.CreatedAt,
		&mapping.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("mapping not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get mapping: %w", err)
	}

	return mapping, nil
}

func (db *PostgresDB) GetAllTicketMappings(userID int) ([]*TicketMapping, error) {
	query := `
		SELECT id, user_id, asana_project_id, asana_task_id, youtrack_project_id, youtrack_issue_id, created_at, updated_at
		FROM ticket_mappings
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := db.conn.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get mappings: %w", err)
	}
	defer rows.Close()

	mappings := []*TicketMapping{}
	for rows.Next() {
		mapping := &TicketMapping{}
		err := rows.Scan(
			&mapping.ID,
			&mapping.UserID,
			&mapping.AsanaProjectID,
			&mapping.AsanaTaskID,
			&mapping.YouTrackProjectID,
			&mapping.YouTrackIssueID,
			&mapping.CreatedAt,
			&mapping.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan mapping: %w", err)
		}
		mappings = append(mappings, mapping)
	}

	return mappings, nil
}

func (db *PostgresDB) FindMappingByAsanaID(userID int, asanaTaskID string) (*TicketMapping, error) {
	query := `
		SELECT id, user_id, asana_project_id, asana_task_id, youtrack_project_id, youtrack_issue_id, created_at, updated_at
		FROM ticket_mappings
		WHERE user_id = $1 AND asana_task_id = $2
	`

	mapping := &TicketMapping{}
	err := db.conn.QueryRow(query, userID, asanaTaskID).Scan(
		&mapping.ID,
		&mapping.UserID,
		&mapping.AsanaProjectID,
		&mapping.AsanaTaskID,
		&mapping.YouTrackProjectID,
		&mapping.YouTrackIssueID,
		&mapping.CreatedAt,
		&mapping.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("mapping not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find mapping: %w", err)
	}

	return mapping, nil
}

func (db *PostgresDB) FindMappingByYouTrackID(userID int, youtrackIssueID string) (*TicketMapping, error) {
	query := `
		SELECT id, user_id, asana_project_id, asana_task_id, youtrack_project_id, youtrack_issue_id, created_at, updated_at
		FROM ticket_mappings
		WHERE user_id = $1 AND youtrack_issue_id = $2
	`

	mapping := &TicketMapping{}
	err := db.conn.QueryRow(query, userID, youtrackIssueID).Scan(
		&mapping.ID,
		&mapping.UserID,
		&mapping.AsanaProjectID,
		&mapping.AsanaTaskID,
		&mapping.YouTrackProjectID,
		&mapping.YouTrackIssueID,
		&mapping.CreatedAt,
		&mapping.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("mapping not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find mapping: %w", err)
	}

	return mapping, nil
}

func (db *PostgresDB) DeleteTicketMapping(id int) error {
	query := `DELETE FROM ticket_mappings WHERE id = $1`

	_, err := db.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete mapping: %w", err)
	}

	log.Printf("DB: Deleted mapping ID: %d\n", id)
	return nil
}

// =============================================================================
// IGNORED TICKETS OPERATIONS
// =============================================================================

func (db *PostgresDB) AddIgnoredTicket(userID int, asanaProjectID, ticketID, ignoreType string) error {
	query := `
		INSERT INTO ignored_tickets (user_id, asana_project_id, ticket_id, ignore_type, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, asana_project_id, ticket_id)
		DO UPDATE SET ignore_type = $4, created_at = $5
	`

	_, err := db.conn.Exec(query, userID, asanaProjectID, ticketID, ignoreType, time.Now())
	if err != nil {
		return fmt.Errorf("failed to add ignored ticket: %w", err)
	}

	log.Printf("DB: Added ignored ticket: %s (type: %s)\n", ticketID, ignoreType)
	return nil
}

func (db *PostgresDB) RemoveIgnoredTicket(userID int, asanaProjectID, ticketID string) error {
	query := `DELETE FROM ignored_tickets WHERE user_id = $1 AND asana_project_id = $2 AND ticket_id = $3`

	_, err := db.conn.Exec(query, userID, asanaProjectID, ticketID)
	if err != nil {
		return fmt.Errorf("failed to remove ignored ticket: %w", err)
	}

	log.Printf("DB: Removed ignored ticket: %s\n", ticketID)
	return nil
}

func (db *PostgresDB) GetIgnoredTickets(userID int, asanaProjectID string) ([]string, error) {
	query := `SELECT ticket_id FROM ignored_tickets WHERE user_id = $1 AND asana_project_id = $2`

	rows, err := db.conn.Query(query, userID, asanaProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ignored tickets: %w", err)
	}
	defer rows.Close()

	tickets := []string{}
	for rows.Next() {
		var ticketID string
		if err := rows.Scan(&ticketID); err != nil {
			return nil, fmt.Errorf("failed to scan ticket: %w", err)
		}
		tickets = append(tickets, ticketID)
	}

	return tickets, nil
}

func (db *PostgresDB) IsTicketIgnored(userID int, asanaProjectID, ticketID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM ignored_tickets WHERE user_id = $1 AND asana_project_id = $2 AND ticket_id = $3)`

	var exists bool
	err := db.conn.QueryRow(query, userID, asanaProjectID, ticketID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if ticket is ignored: %w", err)
	}

	return exists, nil
}

func (db *PostgresDB) ClearTemporaryIgnores(userID int, asanaProjectID string) error {
	query := `DELETE FROM ignored_tickets WHERE user_id = $1 AND asana_project_id = $2 AND ignore_type = 'temp'`

	_, err := db.conn.Exec(query, userID, asanaProjectID)
	if err != nil {
		return fmt.Errorf("failed to clear temporary ignores: %w", err)
	}

	log.Printf("DB: Cleared temporary ignores for user %d\n", userID)
	return nil
}
