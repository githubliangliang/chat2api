-- [sqlite-converted] from PostgreSQL migration: 078_add_usage_log_requested_model_index_notx.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Support requested_model / upstream_model aggregations with time-range filters.
CREATE INDEX IF NOT EXISTS idx_usage_logs_created_requested_model_upstream_model
ON usage_logs (created_at, requested_model, upstream_model);
