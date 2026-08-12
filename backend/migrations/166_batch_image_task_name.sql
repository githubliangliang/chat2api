-- [sqlite-converted] from PostgreSQL migration: 166_batch_image_task_name.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
ALTER TABLE batch_image_jobs ADD COLUMN task_name VARCHAR(255) NOT NULL DEFAULT '';

UPDATE batch_image_jobs
SET task_name = strftime('%Y-%m-%d %H:%M:%S', created_at)
WHERE task_name = '';

CREATE INDEX IF NOT EXISTS batch_image_jobs_task_name_idx ON batch_image_jobs (task_name);

-- [sqlite] skipped COMMENT ON column

