-- [sqlite-converted] from PostgreSQL migration: 136_remove_ops_retry_replay.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Remove unused Ops retry/replay storage.
-- The retry endpoints are no longer exposed, so keeping request bodies and
-- retry audit rows only increases write width, memory retention, and DB size.

DROP TABLE IF EXISTS ops_retry_attempts;

-- [sqlite] best-effort DROP COLUMN
ALTER TABLE ops_error_logs DROP COLUMN request_body;
-- [sqlite] best-effort DROP COLUMN
ALTER TABLE ops_error_logs DROP COLUMN request_headers;
-- [sqlite] best-effort DROP COLUMN
ALTER TABLE ops_error_logs DROP COLUMN request_body_truncated;
-- [sqlite] best-effort DROP COLUMN
ALTER TABLE ops_error_logs DROP COLUMN request_body_bytes;
-- [sqlite] best-effort DROP COLUMN
ALTER TABLE ops_error_logs DROP COLUMN is_retryable;
-- [sqlite] best-effort DROP COLUMN
ALTER TABLE ops_error_logs DROP COLUMN retry_count;
-- [sqlite] best-effort DROP COLUMN
ALTER TABLE ops_error_logs DROP COLUMN resolved_retry_id;

-- [sqlite] skipped COMMENT ON table
 -- request replay storage removed
