-- BoardSync PostgreSQL Schema
-- This file contains all table definitions for the BoardSync application

-- Enable UUID extension (optional, for future use)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =============================================================================
-- USERS TABLE
-- =============================================================================
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

-- =============================================================================
-- USER SETTINGS TABLE
-- =============================================================================
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

-- Index for faster user lookups
CREATE INDEX IF NOT EXISTS idx_user_settings_user_id ON user_settings(user_id);

-- =============================================================================
-- TICKET MAPPINGS TABLE
-- =============================================================================
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

-- Indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_ticket_mappings_user_id ON ticket_mappings(user_id);
CREATE INDEX IF NOT EXISTS idx_ticket_mappings_asana_task ON ticket_mappings(asana_task_id);
CREATE INDEX IF NOT EXISTS idx_ticket_mappings_youtrack_issue ON ticket_mappings(youtrack_issue_id);

-- =============================================================================
-- IGNORED TICKETS TABLE
-- =============================================================================
CREATE TABLE IF NOT EXISTS ignored_tickets (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    asana_project_id VARCHAR(255) NOT NULL,
    ticket_id VARCHAR(255) NOT NULL,
    ignore_type VARCHAR(50) NOT NULL CHECK (ignore_type IN ('temp', 'forever')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, asana_project_id, ticket_id)
);

-- Indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_ignored_tickets_user_id ON ignored_tickets(user_id);
CREATE INDEX IF NOT EXISTS idx_ignored_tickets_ticket_id ON ignored_tickets(ticket_id);

-- =============================================================================
-- SYNC OPERATIONS TABLE
-- =============================================================================
CREATE TABLE IF NOT EXISTS sync_operations (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    operation_type VARCHAR(100) NOT NULL,
    operation_data JSONB DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed', 'rolled_back')),
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

-- Indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_sync_operations_user_id ON sync_operations(user_id);
CREATE INDEX IF NOT EXISTS idx_sync_operations_status ON sync_operations(status);
CREATE INDEX IF NOT EXISTS idx_sync_operations_created_at ON sync_operations(created_at DESC);

-- =============================================================================
-- ROLLBACK SNAPSHOTS TABLE
-- =============================================================================
CREATE TABLE IF NOT EXISTS rollback_snapshots (
    id SERIAL PRIMARY KEY,
    operation_id INTEGER NOT NULL REFERENCES sync_operations(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    snapshot_data JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP + INTERVAL '30 days'),
    UNIQUE(operation_id)
);

-- Indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_rollback_snapshots_operation_id ON rollback_snapshots(operation_id);
CREATE INDEX IF NOT EXISTS idx_rollback_snapshots_user_id ON rollback_snapshots(user_id);
CREATE INDEX IF NOT EXISTS idx_rollback_snapshots_expires_at ON rollback_snapshots(expires_at);

-- =============================================================================
-- AUDIT LOGS TABLE
-- =============================================================================
CREATE TABLE IF NOT EXISTS audit_logs (
    id SERIAL PRIMARY KEY,
    operation_id INTEGER REFERENCES sync_operations(id) ON DELETE SET NULL,
    ticket_id VARCHAR(255) NOT NULL,
    platform VARCHAR(50) NOT NULL CHECK (platform IN ('asana', 'youtrack')),
    action_type VARCHAR(50) NOT NULL,
    user_email VARCHAR(255) NOT NULL,
    old_value TEXT,
    new_value TEXT,
    field_name VARCHAR(255),
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for faster queries and filtering
CREATE INDEX IF NOT EXISTS idx_audit_logs_operation_id ON audit_logs(operation_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_ticket_id ON audit_logs(ticket_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_platform ON audit_logs(platform);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action_type ON audit_logs(action_type);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_email ON audit_logs(user_email);
CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp DESC);

-- =============================================================================
-- TRIGGERS FOR UPDATED_AT
-- =============================================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger for users table
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger for user_settings table
CREATE TRIGGER update_user_settings_updated_at
    BEFORE UPDATE ON user_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger for ticket_mappings table
CREATE TRIGGER update_ticket_mappings_updated_at
    BEFORE UPDATE ON ticket_mappings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- CLEANUP FUNCTION FOR EXPIRED SNAPSHOTS
-- =============================================================================

-- Function to clean up expired snapshots
CREATE OR REPLACE FUNCTION cleanup_expired_snapshots()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM rollback_snapshots
    WHERE expires_at < CURRENT_TIMESTAMP;

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Optional: Create a scheduled job to run cleanup (if using pg_cron extension)
-- This is commented out as it requires pg_cron extension
-- SELECT cron.schedule('cleanup-expired-snapshots', '0 2 * * *', 'SELECT cleanup_expired_snapshots()');
