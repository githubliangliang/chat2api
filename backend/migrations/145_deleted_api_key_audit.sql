-- [sqlite-converted] from PostgreSQL migration: 145_deleted_api_key_audit.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 已删除 API key 审计表:删除 key 时同步留存(明文 key、所有者、key 信息),
-- 供认证失败(INVALID_API_KEY)反查"这个失效 key 曾属于谁"。
-- 仅对本表上线后删除的 key 生效;此前已删的 key 原值已被 tombstone 覆盖,无法补录。
-- [sqlite] skipped SET LOCAL

-- [sqlite] skipped SET LOCAL


CREATE TABLE IF NOT EXISTS deleted_api_key_audits (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    key          VARCHAR(128) NOT NULL,            -- 原 key 明文(复用 api_keys 策略),非唯一
    api_key_id   BIGINT NOT NULL,                  -- 原 api_keys.id
    user_id      BIGINT NOT NULL,                  -- 原所有者(不加外键,与 ops 表设计哲学一致)
    key_name     VARCHAR(100) NOT NULL DEFAULT '', -- 原 key 名称,便于展示
    deleted_at DATETIME  NOT NULL DEFAULT (datetime('now')),
    created_at DATETIME  NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS deletedapikeyaudit_key     ON deleted_api_key_audits (key);
CREATE INDEX IF NOT EXISTS deletedapikeyaudit_user_id ON deleted_api_key_audits (user_id);

ALTER TABLE ops_error_logs ADD COLUMN attempted_key_prefix      VARCHAR(32);
ALTER TABLE ops_error_logs ADD COLUMN deleted_key_owner_user_id BIGINT;
ALTER TABLE ops_error_logs ADD COLUMN deleted_key_name          VARCHAR(100);
