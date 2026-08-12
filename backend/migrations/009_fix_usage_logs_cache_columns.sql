-- [sqlite-converted] from PostgreSQL migration: 009_fix_usage_logs_cache_columns.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Ensure usage_logs cache token columns use the underscored names expected by code.
-- Backfill from legacy column names if they exist.

ALTER TABLE usage_logs ADD COLUMN cache_creation_5m_tokens INT NOT NULL DEFAULT 0;

ALTER TABLE usage_logs ADD COLUMN cache_creation_1h_tokens INT NOT NULL DEFAULT 0;

-- [sqlite] skipped PostgreSQL DO $$ ... $$ block

