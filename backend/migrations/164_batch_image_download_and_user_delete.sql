-- [sqlite-converted] from PostgreSQL migration: 164_batch_image_download_and_user_delete.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
ALTER TABLE batch_image_jobs ADD COLUMN downloaded_at TEXT;
ALTER TABLE batch_image_jobs ADD COLUMN user_deleted_at TEXT;

CREATE INDEX IF NOT EXISTS batch_image_jobs_downloaded_at_idx ON batch_image_jobs (downloaded_at);
CREATE INDEX IF NOT EXISTS batch_image_jobs_user_deleted_at_idx ON batch_image_jobs (user_deleted_at);

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

