-- [sqlite-converted] from PostgreSQL migration: 010_add_usage_logs_aggregated_indexes.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 为聚合查询补充复合索引
CREATE INDEX IF NOT EXISTS idx_usage_logs_account_created_at ON usage_logs(account_id, created_at);
CREATE INDEX IF NOT EXISTS idx_usage_logs_api_key_created_at ON usage_logs(api_key_id, created_at);
CREATE INDEX IF NOT EXISTS idx_usage_logs_model_created_at ON usage_logs(model, created_at);
