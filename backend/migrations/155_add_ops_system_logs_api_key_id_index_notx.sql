-- [sqlite-converted] from PostgreSQL migration: 155_add_ops_system_logs_api_key_id_index_notx.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 155_add_ops_system_logs_api_key_id_index_notx.sql
-- Non-transactional migration: CREATE INDEX cannot run in a transaction.

CREATE INDEX IF NOT EXISTS idx_ops_system_logs_api_key_id_created_at
  ON ops_system_logs (api_key_id, created_at DESC);
