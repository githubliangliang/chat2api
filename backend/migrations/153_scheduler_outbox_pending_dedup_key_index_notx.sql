-- [sqlite-converted] from PostgreSQL migration: 153_scheduler_outbox_pending_dedup_key_index_notx.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
CREATE UNIQUE INDEX IF NOT EXISTS idx_scheduler_outbox_pending_dedup_key
    ON scheduler_outbox (dedup_key)
    WHERE dedup_key IS NOT NULL;
