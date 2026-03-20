-- BoardSync Neon PostgreSQL Schema
-- Run this once against your Neon database before starting the app.

CREATE TABLE IF NOT EXISTS users (
    id           SERIAL PRIMARY KEY,
    username     TEXT NOT NULL UNIQUE,
    email        TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
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
