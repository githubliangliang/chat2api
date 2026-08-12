-- [sqlite-converted] from PostgreSQL migration: 174_add_usage_log_long_context_billing.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Snapshot whether long-context pricing changed token prices for a request so
-- usage history can explain the applied charge without inferring from totals.
ALTER TABLE usage_logs ADD COLUMN long_context_billing_applied BOOLEAN NOT NULL DEFAULT FALSE;
