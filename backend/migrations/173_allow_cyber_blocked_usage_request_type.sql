-- [sqlite-converted] from PostgreSQL migration: 173_allow_cyber_blocked_usage_request_type.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Cyber-policy blocks are recorded as request_type=4 so they remain visible in
-- usage audits without being confused with legacy request_type=0 rows.
-- [sqlite] DROP CONSTRAINT usage_logs_request_type_check on usage_logs → try DROP INDEX
DROP INDEX IF EXISTS usage_logs_request_type_check;

-- [sqlite] skipped ADD CONSTRAINT

