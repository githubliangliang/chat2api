-- [sqlite-converted] from PostgreSQL migration: 014_drop_legacy_allowed_groups.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 014: 删除 legacy users.allowed_groups 列
-- 任务：fix-medium-data-hygiene 3.3
--
-- 前置条件：
--   - 迁移 007 已将数据回填到 user_allowed_groups 联接表
--   - 迁移 013 已记录所有孤立的 group_id 到审计表
--   - 应用代码已停止写入该列（3.2 完成）
--
-- 该列现已废弃，所有读写操作均使用 user_allowed_groups 联接表。

-- 删除 allowed_groups 列
-- [sqlite] DROP COLUMN allowed_groups (best-effort)
-- -- [sqlite] best-effort DROP COLUMN
ALTER TABLE users DROP COLUMN allowed_groups;

-- 添加注释记录删除原因
-- [sqlite] skipped COMMENT ON table

