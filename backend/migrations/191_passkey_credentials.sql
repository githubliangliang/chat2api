-- [sqlite-converted] from PostgreSQL migration: 191_passkey_credentials.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
CREATE TABLE IF NOT EXISTS passkey_user_handles (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    user_handle BLOB NOT NULL UNIQUE,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS passkey_credentials (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id BLOB NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL DEFAULT 'Passkey',
    credential_data TEXT NOT NULL,
    last_used_at TEXT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS passkey_credentials_user_id_idx
    ON passkey_credentials (user_id);

CREATE INDEX IF NOT EXISTS passkey_credentials_last_used_at_idx
    ON passkey_credentials (last_used_at);
