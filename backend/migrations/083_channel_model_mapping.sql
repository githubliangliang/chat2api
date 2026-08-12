-- [sqlite-converted] from PostgreSQL migration: 083_channel_model_mapping.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- [sqlite] skipped SET LOCAL

-- [sqlite] skipped SET LOCAL


ALTER TABLE channels ADD COLUMN model_mapping TEXT DEFAULT '{}';
-- [sqlite] skipped COMMENT ON column

