-- [sqlite-converted] from PostgreSQL migration: 048_add_error_passthrough_rules.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Error Passthrough Rules table
-- Allows administrators to configure how upstream errors are passed through to clients

CREATE TABLE IF NOT EXISTS error_passthrough_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(100) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    priority INTEGER NOT NULL DEFAULT 0,
    error_codes TEXT DEFAULT '[]',
    keywords TEXT DEFAULT '[]',
    match_mode VARCHAR(10) NOT NULL DEFAULT 'any',
    platforms TEXT DEFAULT '[]',
    passthrough_code BOOLEAN NOT NULL DEFAULT true,
    response_code INTEGER,
    passthrough_body BOOLEAN NOT NULL DEFAULT true,
    custom_message TEXT,
    description TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_error_passthrough_rules_enabled ON error_passthrough_rules (enabled);
CREATE INDEX IF NOT EXISTS idx_error_passthrough_rules_priority ON error_passthrough_rules (priority);
