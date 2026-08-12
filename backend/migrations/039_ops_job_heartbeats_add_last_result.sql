-- [sqlite-converted] from PostgreSQL migration: 039_ops_job_heartbeats_add_last_result.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Add last_result to ops_job_heartbeats for UI job details.

ALTER TABLE ops_job_heartbeats ADD COLUMN last_result TEXT;

-- [sqlite] skipped COMMENT ON column

