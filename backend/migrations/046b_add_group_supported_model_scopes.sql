-- [sqlite-converted] from PostgreSQL migration: 046b_add_group_supported_model_scopes.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 添加分组支持的模型系列字段
ALTER TABLE groups ADD COLUMN supported_model_scopes TEXT NOT NULL
DEFAULT '["claude", "gemini_text", "gemini_image"]';

-- [sqlite] skipped COMMENT ON column

