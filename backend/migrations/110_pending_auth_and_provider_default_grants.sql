-- [sqlite-converted] from PostgreSQL migration: 110_pending_auth_and_provider_default_grants.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
CREATE TABLE IF NOT EXISTS user_provider_default_grants (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider_type VARCHAR(20) NOT NULL,
    grant_reason VARCHAR(20) NOT NULL DEFAULT 'first_bind',
    granted_at DATETIME NOT NULL DEFAULT (datetime('now')),
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    CONSTRAINT user_provider_default_grants_provider_type_check
        CHECK (provider_type IN ('email', 'linuxdo', 'wechat', 'oidc')),
    CONSTRAINT user_provider_default_grants_reason_check
        CHECK (grant_reason IN ('signup', 'first_bind'))
);

CREATE UNIQUE INDEX IF NOT EXISTS user_provider_default_grants_user_provider_reason_key
    ON user_provider_default_grants (user_id, provider_type, grant_reason);

CREATE INDEX IF NOT EXISTS user_provider_default_grants_user_id_idx
    ON user_provider_default_grants (user_id);

CREATE TABLE IF NOT EXISTS user_avatars (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    storage_provider VARCHAR(20) NOT NULL DEFAULT 'database',
    storage_key TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL DEFAULT '',
    content_type VARCHAR(100) NOT NULL DEFAULT '',
    byte_size INT NOT NULL DEFAULT 0,
    sha256 VARCHAR(64) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS user_avatars_user_id_key
    ON user_avatars (user_id);

INSERT INTO settings (key, value, updated_at)
VALUES
	('auth_source_default_email_balance', '0', datetime('now')),
	('auth_source_default_email_concurrency', '5', datetime('now')),
	('auth_source_default_email_subscriptions', '[]', datetime('now')),
	('auth_source_default_email_grant_on_signup', 'false', datetime('now')),
	('auth_source_default_email_grant_on_first_bind', 'false', datetime('now')),
	('auth_source_default_linuxdo_balance', '0', datetime('now')),
	('auth_source_default_linuxdo_concurrency', '5', datetime('now')),
	('auth_source_default_linuxdo_subscriptions', '[]', datetime('now')),
	('auth_source_default_linuxdo_grant_on_signup', 'false', datetime('now')),
	('auth_source_default_linuxdo_grant_on_first_bind', 'false', datetime('now')),
	('auth_source_default_oidc_balance', '0', datetime('now')),
	('auth_source_default_oidc_concurrency', '5', datetime('now')),
	('auth_source_default_oidc_subscriptions', '[]', datetime('now')),
	('auth_source_default_oidc_grant_on_signup', 'false', datetime('now')),
	('auth_source_default_oidc_grant_on_first_bind', 'false', datetime('now')),
	('auth_source_default_wechat_balance', '0', datetime('now')),
	('auth_source_default_wechat_concurrency', '5', datetime('now')),
	('auth_source_default_wechat_subscriptions', '[]', datetime('now')),
	('auth_source_default_wechat_grant_on_signup', 'false', datetime('now')),
	('auth_source_default_wechat_grant_on_first_bind', 'false', datetime('now')),
	('force_email_on_third_party_signup', 'false', datetime('now'))
ON CONFLICT (key) DO NOTHING;
