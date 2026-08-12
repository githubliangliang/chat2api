-- [sqlite-converted] from PostgreSQL migration: 156_content_moderation_matched_keyword.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 风控中心：记录关键词拦截命中的具体关键词

ALTER TABLE content_moderation_logs ADD COLUMN matched_keyword VARCHAR(255) NOT NULL DEFAULT '';
