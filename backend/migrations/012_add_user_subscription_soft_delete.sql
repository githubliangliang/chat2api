-- [sqlite-converted] from PostgreSQL migration: 012_add_user_subscription_soft_delete.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 012: 为 user_subscriptions 表添加软删除支持
-- 任务：fix-medium-data-hygiene 1.1

-- 添加 deleted_at 字段
ALTER TABLE user_subscriptions ADD COLUMN deleted_at TEXT DEFAULT NULL;

-- 添加 deleted_at 索引以优化软删除查询
CREATE INDEX IF NOT EXISTS usersubscription_deleted_at
ON user_subscriptions (deleted_at);

-- 注释：与其他使用软删除的实体保持一致
-- [sqlite] skipped COMMENT ON column

