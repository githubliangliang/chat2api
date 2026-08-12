-- [sqlite-converted] from PostgreSQL migration: 068_add_announcement_notify_mode.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
ALTER TABLE announcements ADD COLUMN notify_mode VARCHAR(20) NOT NULL DEFAULT 'silent';
