-- [sqlite-converted] from PostgreSQL migration: 106_add_account_stats_pricing_intervals.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Add intervals table for account stats pricing rules (mirrors channel_pricing_intervals).
CREATE TABLE IF NOT EXISTS channel_account_stats_pricing_intervals (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    pricing_id        BIGINT         NOT NULL REFERENCES channel_account_stats_model_pricing(id) ON DELETE CASCADE,
    min_tokens        INT            NOT NULL DEFAULT 0,
    max_tokens        INT,
    tier_label        VARCHAR(50),
    input_price       NUMERIC(20,12),
    output_price      NUMERIC(20,12),
    cache_write_price NUMERIC(20,12),
    cache_read_price  NUMERIC(20,12),
    per_request_price NUMERIC(20,12),
    sort_order        INT            NOT NULL DEFAULT 0,
    created_at DATETIME    NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME    NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_account_stats_pricing_intervals_pricing_id
    ON channel_account_stats_pricing_intervals (pricing_id);
