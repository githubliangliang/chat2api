-- [sqlite-converted] from PostgreSQL migration: 077_add_usage_log_requested_model.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Add requested_model field to usage_logs for normalized request/upstream model tracking.
-- NULL means historical rows written before requested_model dual-write was introduced.
ALTER TABLE usage_logs ADD COLUMN requested_model VARCHAR(100);
