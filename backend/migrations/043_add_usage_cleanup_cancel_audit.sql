-- [sqlite-converted] from PostgreSQL migration: 043_add_usage_cleanup_cancel_audit.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 043_add_usage_cleanup_cancel_audit.sql
-- usage_cleanup_tasks 取消任务审计字段

ALTER TABLE usage_cleanup_tasks ADD COLUMN canceled_by BIGINT REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE usage_cleanup_tasks ADD COLUMN canceled_at DATETIME;

CREATE INDEX IF NOT EXISTS idx_usage_cleanup_tasks_canceled_at
    ON usage_cleanup_tasks(canceled_at DESC);

