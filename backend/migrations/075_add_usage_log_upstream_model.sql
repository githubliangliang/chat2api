-- [sqlite-converted] from PostgreSQL migration: 075_add_usage_log_upstream_model.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Add upstream_model field to usage_logs.
-- Stores the actual upstream model name when it differs from the requested model
-- (i.e., when model mapping is applied). NULL means no mapping was applied.
ALTER TABLE usage_logs ADD COLUMN upstream_model VARCHAR(100);
