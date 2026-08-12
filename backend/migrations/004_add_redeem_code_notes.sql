-- [sqlite-converted] from PostgreSQL migration: 004_add_redeem_code_notes.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 为 redeem_codes 表添加备注字段

ALTER TABLE redeem_codes ADD COLUMN notes TEXT DEFAULT NULL;

-- [sqlite] skipped COMMENT ON column

