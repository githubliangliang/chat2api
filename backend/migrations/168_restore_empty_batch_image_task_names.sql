-- [sqlite-converted] from PostgreSQL migration: 168_restore_empty_batch_image_task_names.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
UPDATE batch_image_jobs
SET task_name = strftime('%Y-%m-%d %H:%M:%S', created_at)
WHERE task_name = '';

-- [sqlite] skipped COMMENT ON column

