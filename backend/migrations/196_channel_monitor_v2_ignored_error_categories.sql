-- [sqlite-converted] from PostgreSQL migration: 196_channel_monitor_v2_ignored_error_categories.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Categories excluded from error_rate / health scoring (still shown in error breakdown).
ALTER TABLE channel_monitor_v2_config ADD COLUMN ignored_error_categories TEXT NOT NULL DEFAULT '{}';
