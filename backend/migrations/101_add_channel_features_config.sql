-- [sqlite-converted] from PostgreSQL migration: 101_add_channel_features_config.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
ALTER TABLE channels ADD COLUMN features_config TEXT NOT NULL DEFAULT '{}';
-- [sqlite] skipped COMMENT ON column

