-- [sqlite-converted] from PostgreSQL migration: 089_usage_log_image_output_tokens.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
ALTER TABLE usage_logs ADD COLUMN image_output_tokens INTEGER NOT NULL DEFAULT 0;
ALTER TABLE usage_logs ADD COLUMN image_output_cost DECIMAL(20, 10) NOT NULL DEFAULT 0;
