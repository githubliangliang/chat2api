-- [sqlite-converted] from PostgreSQL migration: 061_add_usage_log_request_type.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Add request_type enum for usage_logs while keeping legacy stream/openai_ws_mode compatibility.
ALTER TABLE usage_logs ADD COLUMN request_type SMALLINT NOT NULL DEFAULT 0;

-- [sqlite] skipped PostgreSQL DO $$ ... $$ block


CREATE INDEX IF NOT EXISTS idx_usage_logs_request_type_created_at
    ON usage_logs (request_type, created_at);

-- Backfill from legacy fields in bounded batches.
-- Why bounded:
-- 1) Full-table UPDATE on large usage_logs can block startup for a long time.
-- 2) request_type=0 rows remain query-compatible via legacy fallback logic
--    (stream/openai_ws_mode) in repository filters.
-- 3) Subsequent writes will use explicit request_type and gradually dilute
--    historical unknown rows.
--
-- openai_ws_mode has higher priority than stream.
-- [sqlite] skipped PostgreSQL DO $$ ... $$ block

