-- [sqlite-converted] from PostgreSQL migration: 174_add_usage_logs_api_key_latest_ip_index_notx.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Support the per-key latest non-empty source IP lookup without scanning full key history.
CREATE INDEX IF NOT EXISTS idx_usage_logs_api_key_latest_ip
    ON usage_logs (api_key_id, created_at DESC, id DESC)
    WHERE ip_address IS NOT NULL AND ip_address <> '';
