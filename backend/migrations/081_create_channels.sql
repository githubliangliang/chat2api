-- [sqlite-converted] from PostgreSQL migration: 081_create_channels.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Create channels table for managing pricing channels.
-- A channel groups multiple groups together and provides custom model pricing.

-- [sqlite] skipped SET LOCAL

-- [sqlite] skipped SET LOCAL


-- 渠道表
CREATE TABLE IF NOT EXISTS channels (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        VARCHAR(100) NOT NULL,
    description TEXT         DEFAULT '',
    status      VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at  TEXT  NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT  NOT NULL DEFAULT (datetime('now'))
);

-- 渠道名称唯一索引
CREATE UNIQUE INDEX IF NOT EXISTS idx_channels_name ON channels (name);
CREATE INDEX IF NOT EXISTS idx_channels_status ON channels (status);

-- 渠道-分组关联表（每个分组只能属于一个渠道）
CREATE TABLE IF NOT EXISTS channel_groups (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id  BIGINT       NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    group_id    BIGINT       NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    created_at  TEXT  NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_channel_groups_group_id ON channel_groups (group_id);
CREATE INDEX IF NOT EXISTS idx_channel_groups_channel_id ON channel_groups (channel_id);

-- 渠道模型定价表（一条定价可绑定多个模型）
CREATE TABLE IF NOT EXISTS channel_model_pricing (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id         BIGINT         NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    models             TEXT          NOT NULL DEFAULT '[]',
    input_price        NUMERIC(20,12),
    output_price       NUMERIC(20,12),
    cache_write_price  NUMERIC(20,12),
    cache_read_price   NUMERIC(20,12),
    image_output_price NUMERIC(20,8),
    created_at         TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at         TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_channel_model_pricing_channel_id ON channel_model_pricing (channel_id);

-- [sqlite] skipped COMMENT ON table

-- [sqlite] skipped COMMENT ON table

-- [sqlite] skipped COMMENT ON table

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

