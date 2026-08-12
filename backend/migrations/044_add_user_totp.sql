-- [sqlite-converted] from PostgreSQL migration: 044_add_user_totp.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 为 users 表添加 TOTP 双因素认证字段
ALTER TABLE users ADD COLUMN totp_secret_encrypted TEXT DEFAULT NULL;
ALTER TABLE users ADD COLUMN totp_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN totp_enabled_at TEXT DEFAULT NULL;

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column


-- 创建索引以支持快速查询启用 2FA 的用户
CREATE INDEX IF NOT EXISTS idx_users_totp_enabled ON users(totp_enabled) WHERE deleted_at IS NULL AND totp_enabled = true;
