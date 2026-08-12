-- [sqlite-converted] from PostgreSQL migration: 195_add_usage_log_upstream_model_mismatch_index_notx.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
CREATE INDEX IF NOT EXISTS idx_usage_logs_upstream_model_mismatch_created_at
    ON usage_logs (created_at DESC, id DESC)
    WHERE upstream_model_mismatch IS TRUE;
