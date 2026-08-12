-- [sqlite-converted] from PostgreSQL migration: 036_scheduler_outbox.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
CREATE TABLE IF NOT EXISTS scheduler_outbox (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL,
    account_id BIGINT NULL,
    group_id BIGINT NULL,
    payload TEXT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_scheduler_outbox_created_at ON scheduler_outbox (created_at);
