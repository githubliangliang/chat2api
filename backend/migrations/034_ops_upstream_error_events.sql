-- [sqlite-converted] from PostgreSQL migration: 034_ops_upstream_error_events.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Add upstream error events list (TEXT) to ops_error_logs for per-request correlation.
--
-- This is intentionally idempotent.

ALTER TABLE ops_error_logs ADD COLUMN upstream_errors TEXT;

-- [sqlite] skipped COMMENT ON column
-- [sqlite] removed stray
