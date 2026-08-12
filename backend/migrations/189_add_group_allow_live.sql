-- [sqlite-converted] from PostgreSQL migration: 189_add_group_allow_live.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
ALTER TABLE groups ADD COLUMN allow_live BOOLEAN NOT NULL DEFAULT false;
