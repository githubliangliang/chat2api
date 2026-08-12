-- [sqlite-converted] from PostgreSQL migration: 175a_add_ops_system_logs_host_index_notx.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
CREATE INDEX IF NOT EXISTS idx_ops_system_logs_host_created_at
  ON ops_system_logs (host, created_at DESC);
