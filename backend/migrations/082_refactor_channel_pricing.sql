-- [sqlite-converted] from PostgreSQL migration: 082_refactor_channel_pricing.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Extend channel_model_pricing with billing_mode and add context-interval child table.
-- Supports three billing modes: token (per-token with context intervals),
-- per_request (per-request with context-size tiers), and image (per-image).

-- [sqlite] skipped SET LOCAL

-- [sqlite] skipped SET LOCAL


-- 1. 为 channel_model_pricing 添加 billing_mode 列
ALTER TABLE channel_model_pricing ADD COLUMN billing_mode VARCHAR(20) NOT NULL DEFAULT 'token';

-- [sqlite] skipped COMMENT ON column


-- 2. 创建区间定价子表
CREATE TABLE IF NOT EXISTS channel_pricing_intervals (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    pricing_id        BIGINT         NOT NULL REFERENCES channel_model_pricing(id) ON DELETE CASCADE,
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

CREATE INDEX IF NOT EXISTS idx_channel_pricing_intervals_pricing_id
    ON channel_pricing_intervals (pricing_id);

-- [sqlite] skipped COMMENT ON table

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column


-- 3. 迁移现有 flat 定价为单区间 [0, +inf)
-- 仅迁移有明确定价（至少一个价格字段非 NULL）的条目
INSERT INTO channel_pricing_intervals (pricing_id, min_tokens, max_tokens, input_price, output_price, cache_write_price, cache_read_price, sort_order)
SELECT
    cmp.id,
    0,
    NULL,
    cmp.input_price,
    cmp.output_price,
    cmp.cache_write_price,
    cmp.cache_read_price,
    0
FROM channel_model_pricing cmp
WHERE cmp.billing_mode = 'token'
  AND (cmp.input_price IS NOT NULL OR cmp.output_price IS NOT NULL
       OR cmp.cache_write_price IS NOT NULL OR cmp.cache_read_price IS NOT NULL)
  AND NOT EXISTS (
      SELECT 1 FROM channel_pricing_intervals cpi WHERE cpi.pricing_id = cmp.id
  );

-- 4. 迁移 image_output_price 为 image 模式的区间条目
-- 将有 image_output_price 的现有条目复制为 billing_mode='image' 的独立条目
-- 注意：这里不改变原条目的 billing_mode，而是将 image_output_price 作为向后兼容字段保留
-- 实际的 image 计费在未来由独立的 billing_mode='image' 条目处理
