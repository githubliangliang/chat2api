-- [sqlite-converted] from PostgreSQL migration: 029_add_group_claude_code_restriction.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 029_add_group_claude_code_restriction.sql
-- 添加分组级别的 Claude Code 客户端限制功能

-- 添加 claude_code_only 字段：是否仅允许 Claude Code 客户端
ALTER TABLE groups ADD COLUMN claude_code_only BOOLEAN NOT NULL DEFAULT FALSE;

-- 添加 fallback_group_id 字段：非 Claude Code 请求降级到的分组
ALTER TABLE groups ADD COLUMN fallback_group_id BIGINT REFERENCES groups(id) ON DELETE SET NULL;

-- 添加索引优化查询
CREATE INDEX IF NOT EXISTS idx_groups_claude_code_only
ON groups(claude_code_only) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_groups_fallback_group_id
ON groups(fallback_group_id) WHERE deleted_at IS NULL AND fallback_group_id IS NOT NULL;

-- 添加字段注释
-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

