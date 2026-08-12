-- [sqlite-converted] from PostgreSQL migration: 053_add_security_secrets.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 存储系统级密钥（如 JWT 签名密钥、TOTP 加密密钥）
CREATE TABLE IF NOT EXISTS security_secrets (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  key VARCHAR(100) NOT NULL UNIQUE,
  value TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT (datetime('now')),
  updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_security_secrets_key ON security_secrets (key);
