-- [sqlite-converted] from PostgreSQL migration: 172_video_per_second_billing_metadata.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Grok video billing is per second of generated output (xAI rate card), so usage
-- rows must record the billed resolution and duration for auditability. The
-- image-size check constraint must also exempt any video row by video_count
-- instead of billing_mode='video' alone: a video request billed through a
-- token-mode channel price produces billing_mode='token' with image_count=1
-- (legacy media counter) and no image_size, which the previous constraint
-- rejected and silently dropped the whole billing transaction.

ALTER TABLE usage_logs ADD COLUMN video_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE usage_logs ADD COLUMN video_resolution VARCHAR(10);
ALTER TABLE usage_logs ADD COLUMN video_duration_seconds INTEGER;

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column


-- [sqlite] DROP CONSTRAINT usage_logs_image_billing_size_check on usage_logs → try DROP INDEX
DROP INDEX IF EXISTS usage_logs_image_billing_size_check;

-- [sqlite] skipped ADD CONSTRAINT


-- Group video prices are per-second rates (USD/s), matching the xAI rate card;
-- total cost = per-second price x duration seconds. Clarify the column docs
-- introduced by migration 170, which read as per-video prices.
-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

