-- [sqlite-converted] from PostgreSQL migration: 060_add_usage_log_openai_ws_mode.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Add openai_ws_mode flag to usage_logs to persist exact OpenAI WS transport type.
ALTER TABLE usage_logs ADD COLUMN openai_ws_mode BOOLEAN NOT NULL DEFAULT FALSE;
