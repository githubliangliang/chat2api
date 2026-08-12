-- [sqlite-converted] from PostgreSQL migration: 005_schema_parity.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Align SQL migrations with current GORM persistence models.
-- This file is designed to be safe on both fresh installs and existing databases.

-- users: add fields added after initial migration
ALTER TABLE users ADD COLUMN username VARCHAR(100) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN wechat VARCHAR(100) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN notes TEXT NOT NULL DEFAULT '';

-- api_keys: allow longer keys (GORM model uses size:128)
-- [sqlite] skipped ALTER COLUMN TYPE on api_keys.key (unsupported)


-- accounts: scheduling and rate-limit fields used by repository queries
ALTER TABLE accounts ADD COLUMN schedulable BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE accounts ADD COLUMN rate_limited_at DATETIME;
ALTER TABLE accounts ADD COLUMN rate_limit_reset_at DATETIME;
ALTER TABLE accounts ADD COLUMN overload_until TEXT;
ALTER TABLE accounts ADD COLUMN session_window_start TEXT;
ALTER TABLE accounts ADD COLUMN session_window_end TEXT;
ALTER TABLE accounts ADD COLUMN session_window_status VARCHAR(20);

CREATE INDEX IF NOT EXISTS idx_accounts_schedulable ON accounts(schedulable);
CREATE INDEX IF NOT EXISTS idx_accounts_rate_limited_at ON accounts(rate_limited_at);
CREATE INDEX IF NOT EXISTS idx_accounts_rate_limit_reset_at ON accounts(rate_limit_reset_at);
CREATE INDEX IF NOT EXISTS idx_accounts_overload_until ON accounts(overload_until);

-- redeem_codes: subscription redeem fields
ALTER TABLE redeem_codes ADD COLUMN group_id BIGINT REFERENCES groups(id) ON DELETE SET NULL;
ALTER TABLE redeem_codes ADD COLUMN validity_days INT NOT NULL DEFAULT 30;
CREATE INDEX IF NOT EXISTS idx_redeem_codes_group_id ON redeem_codes(group_id);

-- usage_logs: billing type used by filters and stats
ALTER TABLE usage_logs ADD COLUMN billing_type SMALLINT NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_usage_logs_billing_type ON usage_logs(billing_type);

-- settings: key-value store
CREATE TABLE IF NOT EXISTS settings (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    key         VARCHAR(100) NOT NULL UNIQUE,
    value       TEXT NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

