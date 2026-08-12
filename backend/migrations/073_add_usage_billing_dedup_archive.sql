-- [sqlite-converted] from PostgreSQL migration: 073_add_usage_billing_dedup_archive.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 冷归档旧账务幂等键，缩小热表索引与清理范围，同时不丢失长期去重能力。

CREATE TABLE IF NOT EXISTS usage_billing_dedup_archive (
    request_id VARCHAR(255) NOT NULL,
    api_key_id BIGINT NOT NULL,
    request_fingerprint VARCHAR(64) NOT NULL,
    created_at DATETIME NOT NULL,
    archived_at DATETIME NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (request_id, api_key_id)
);
