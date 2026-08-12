-- [sqlite-converted] from PostgreSQL migration: 053_add_skip_monitoring_to_error_passthrough.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Add skip_monitoring field to error_passthrough_rules table
-- When true, errors matching this rule will not be recorded in ops_error_logs
ALTER TABLE error_passthrough_rules ADD COLUMN skip_monitoring BOOLEAN NOT NULL DEFAULT false;
