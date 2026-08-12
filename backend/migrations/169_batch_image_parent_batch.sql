-- [sqlite-converted] from PostgreSQL migration: 169_batch_image_parent_batch.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
ALTER TABLE batch_image_jobs ADD COLUMN parent_batch_id VARCHAR(64);

CREATE INDEX IF NOT EXISTS batch_image_jobs_parent_batch_id_idx
    ON batch_image_jobs (parent_batch_id)
    WHERE parent_batch_id IS NOT NULL AND parent_batch_id <> '';

-- [sqlite] skipped COMMENT ON column

