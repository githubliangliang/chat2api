-- [sqlite-converted] from PostgreSQL migration: 175_add_ops_system_logs_host.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Track the application host that emitted each indexed system log.
ALTER TABLE ops_system_logs ADD COLUMN host VARCHAR(255);
