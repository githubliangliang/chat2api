-- [sqlite-converted] from PostgreSQL migration: 072_add_usage_billing_dedup_created_at_brin_notx.sql
-- SQLite: BRIN indexes are not supported; use a regular index instead.
CREATE INDEX IF NOT EXISTS idx_usage_logs_billing_dedup_created_at
    ON usage_logs (created_at);
