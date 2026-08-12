-- [sqlite-converted] from PostgreSQL migration: 076_add_usage_log_upstream_model_index_notx.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Support upstream_model / mapping model distribution aggregations with time-range filters.
CREATE INDEX IF NOT EXISTS idx_usage_logs_created_model_upstream_model
ON usage_logs (created_at, model, upstream_model);
