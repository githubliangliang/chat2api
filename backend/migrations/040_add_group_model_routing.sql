-- [sqlite-converted] from PostgreSQL migration: 040_add_group_model_routing.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 040_add_group_model_routing.sql
-- 添加分组级别的模型路由配置功能

-- 添加 model_routing 字段：模型路由配置（TEXT 格式）
-- 格式: {"model_pattern": [account_id1, account_id2], ...}
-- 例如: {"claude-opus-*": [1, 2], "claude-sonnet-*": [3, 4, 5]}
ALTER TABLE groups ADD COLUMN model_routing TEXT DEFAULT '{}';

-- 添加字段注释
-- [sqlite] skipped COMMENT ON column

