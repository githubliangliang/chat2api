-- [sqlite-converted] from PostgreSQL migration: 062_add_scheduler_and_usage_composite_indexes_notx.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
CREATE INDEX IF NOT EXISTS idx_accounts_schedulable_hot
    ON accounts (platform, priority)
    WHERE deleted_at IS NULL AND status = 'active' AND schedulable = true;

CREATE INDEX IF NOT EXISTS idx_accounts_active_schedulable
    ON accounts (priority, status)
    WHERE deleted_at IS NULL AND schedulable = true;

CREATE INDEX IF NOT EXISTS idx_user_subscriptions_user_status_expires_active
    ON user_subscriptions (user_id, status, expires_at)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_usage_logs_group_created_at_not_null
    ON usage_logs (group_id, created_at)
    WHERE group_id IS NOT NULL;
