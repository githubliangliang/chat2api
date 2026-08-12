-- [sqlite-converted] from PostgreSQL migration: 154_add_ops_system_logs_api_key_id.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 154_add_ops_system_logs_api_key_id.sql
-- Persist API key database id as a queryable system log index column.

ALTER TABLE ops_system_logs ADD COLUMN api_key_id BIGINT;
