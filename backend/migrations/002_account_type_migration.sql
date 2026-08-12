-- [sqlite-converted] from PostgreSQL migration: 002_account_type_migration.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Sub2API 账号类型迁移脚本
-- 将 'official' 类型账号迁移为 'oauth' 或 'setup-token'
-- 根据 json_extract(credentials, '$.scope') 字段判断：
--   - 包含 'user:profile' 的是 'oauth' 类型
--   - 只有 'user:inference' 的是 'setup-token' 类型

-- 1. 将包含 profile scope 的 official 账号迁移为 oauth
UPDATE accounts
SET type = 'oauth',
    updated_at = datetime('now')
WHERE type = 'official'
  AND json_extract(credentials, '$.scope') LIKE '%user:profile%';

-- 2. 将只有 inference scope 的 official 账号迁移为 setup-token
UPDATE accounts
SET type = 'setup-token',
    updated_at = datetime('now')
WHERE type = 'official'
  AND (
    json_extract(credentials, '$.scope') = 'user:inference'
    OR json_extract(credentials, '$.scope') NOT LIKE '%user:profile%'
  );

-- 3. 处理没有 scope 字段的旧账号（默认为 oauth）
UPDATE accounts
SET type = 'oauth',
    updated_at = datetime('now')
WHERE type = 'official'
  AND (json_extract(credentials, '$.scope') IS NULL OR json_extract(credentials, '$.scope') = '');

-- 4. 验证迁移结果（查询是否还有 official 类型账号）
-- SELECT COUNT(*) FROM accounts WHERE type = 'official';
-- 如果结果为 0，说明迁移成功
