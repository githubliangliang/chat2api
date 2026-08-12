-- [sqlite-converted] from PostgreSQL migration: 096_payment_provider_instances.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
CREATE TABLE IF NOT EXISTS payment_provider_instances (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider_key VARCHAR(30) NOT NULL,
    name VARCHAR(100) NOT NULL DEFAULT '',
    config TEXT NOT NULL,
    supported_types VARCHAR(200) NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INT NOT NULL DEFAULT 0,
    limits TEXT NOT NULL DEFAULT '',
    refund_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_payment_provider_instances_provider_key ON payment_provider_instances(provider_key);
CREATE INDEX IF NOT EXISTS idx_payment_provider_instances_enabled ON payment_provider_instances(enabled);
