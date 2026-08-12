-- [sqlite-converted] from PostgreSQL migration: 071_add_usage_billing_dedup.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 窄表账务幂等键：将“是否已扣费”从 usage_logs 解耦出来
-- 幂等执行：可重复运行

CREATE TABLE IF NOT EXISTS usage_billing_dedup (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    request_id VARCHAR(255) NOT NULL,
    api_key_id BIGINT NOT NULL,
    request_fingerprint VARCHAR(64) NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_usage_billing_dedup_request_api_key
    ON usage_billing_dedup (request_id, api_key_id);
