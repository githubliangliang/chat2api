-- [sqlite-converted] from PostgreSQL migration: 152_scheduler_outbox_dedup_key.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
ALTER TABLE scheduler_outbox ADD COLUMN dedup_key TEXT;
