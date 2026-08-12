-- [sqlite-converted] from PostgreSQL migration: 149_proxy_expiry_fallback.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- proxies: 有效期 + 失败回退
ALTER TABLE proxies ADD COLUMN expires_at TEXT;
ALTER TABLE proxies ADD COLUMN fallback_mode varchar(20) NOT NULL DEFAULT 'none';
ALTER TABLE proxies ADD COLUMN backup_proxy_id BIGINT REFERENCES proxies(id) ON DELETE SET NULL;
ALTER TABLE proxies ADD COLUMN expiry_warn_days INT NOT NULL DEFAULT 7;
CREATE INDEX IF NOT EXISTS proxies_expires_at_idx ON proxies (expires_at);
CREATE INDEX IF NOT EXISTS proxies_backup_proxy_id_idx ON proxies (backup_proxy_id);

-- accounts: fallback 来源（手动回切用）
ALTER TABLE accounts ADD COLUMN proxy_fallback_origin_id BIGINT;
CREATE INDEX IF NOT EXISTS accounts_proxy_fallback_origin_id_idx ON accounts (proxy_fallback_origin_id);
