-- [sqlite-converted] from PostgreSQL migration: 093_payment_audit_logs.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
CREATE TABLE IF NOT EXISTS payment_audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id VARCHAR(64) NOT NULL,
    action VARCHAR(50) NOT NULL,
    detail TEXT NOT NULL DEFAULT '',
    operator VARCHAR(100) NOT NULL DEFAULT 'system',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_payment_audit_logs_order_id ON payment_audit_logs(order_id);
