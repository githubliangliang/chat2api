-- [sqlite-converted] from PostgreSQL migration: 148_add_ops_error_logs_user_time_index_notx.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 148_add_ops_error_logs_user_time_index_notx.sql
-- 用户侧"错误请求"按 user_id 时间倒序分页所需的部分索引。
-- 非事务迁移（_notx）：CREATE INDEX 不可在事务内执行。
CREATE INDEX IF NOT EXISTS idx_ops_error_logs_user_time
  ON ops_error_logs (user_id, created_at DESC)
  WHERE user_id IS NOT NULL;
