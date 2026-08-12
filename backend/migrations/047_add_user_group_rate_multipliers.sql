-- [sqlite-converted] from PostgreSQL migration: 047_add_user_group_rate_multipliers.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 用户专属分组倍率表
-- 允许管理员为特定用户设置分组的专属计费倍率，覆盖分组默认倍率
CREATE TABLE IF NOT EXISTS user_group_rate_multipliers (
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id        BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    rate_multiplier DECIMAL(10,4) NOT NULL,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (user_id, group_id)
);

-- 按 group_id 查询索引（删除分组时清理关联记录）
CREATE INDEX IF NOT EXISTS idx_user_group_rate_multipliers_group_id
    ON user_group_rate_multipliers(group_id);

-- [sqlite] skipped COMMENT ON table

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

