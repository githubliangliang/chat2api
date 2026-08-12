-- [sqlite-converted] from PostgreSQL migration: 194_add_usage_log_upstream_response_model.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
ALTER TABLE usage_logs ADD COLUMN upstream_response_model VARCHAR(200);
ALTER TABLE usage_logs ADD COLUMN upstream_model_mismatch BOOLEAN;
