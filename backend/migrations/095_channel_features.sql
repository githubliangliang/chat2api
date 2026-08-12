-- [sqlite-converted] from PostgreSQL migration: 095_channel_features.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
ALTER TABLE channels ADD COLUMN features TEXT NOT NULL DEFAULT '';
-- [sqlite] skipped COMMENT ON column

