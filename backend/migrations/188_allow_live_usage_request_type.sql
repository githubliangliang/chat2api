-- [sqlite-converted] from PostgreSQL migration: 188_allow_live_usage_request_type.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- [sqlite] DROP CONSTRAINT usage_logs_request_type_check on usage_logs → try DROP INDEX
DROP INDEX IF EXISTS usage_logs_request_type_check;

-- [sqlite] skipped ADD CONSTRAINT CHECK (not supported via ALTER TABLE)

