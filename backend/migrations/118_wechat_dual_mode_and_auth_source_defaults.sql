-- [sqlite-converted] from PostgreSQL migration: 118_wechat_dual_mode_and_auth_source_defaults.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
INSERT INTO settings (key, value, updated_at)
VALUES
    (
        'wechat_connect_open_enabled',
        CASE
            WHEN NOT EXISTS (SELECT 1 FROM settings WHERE key = 'wechat_connect_enabled') THEN ''
            WHEN COALESCE((SELECT value FROM settings WHERE key = 'wechat_connect_enabled'), 'false') <> 'true' THEN 'false'
            WHEN LOWER(TRIM(COALESCE((SELECT value FROM settings WHERE key = 'wechat_connect_mode'), 'open'))) = 'mp' THEN 'false'
            ELSE 'true'
		END,
		datetime('now')
    ),
    (
        'wechat_connect_mp_enabled',
        CASE
            WHEN NOT EXISTS (SELECT 1 FROM settings WHERE key = 'wechat_connect_enabled') THEN ''
            WHEN COALESCE((SELECT value FROM settings WHERE key = 'wechat_connect_enabled'), 'false') <> 'true' THEN 'false'
            WHEN LOWER(TRIM(COALESCE((SELECT value FROM settings WHERE key = 'wechat_connect_mode'), 'open'))) = 'mp' THEN 'true'
            ELSE 'false'
		END,
		datetime('now')
	),
	('auth_source_default_email_grant_on_signup', 'false', datetime('now')),
	('auth_source_default_linuxdo_grant_on_signup', 'false', datetime('now')),
	('auth_source_default_oidc_grant_on_signup', 'false', datetime('now')),
	('auth_source_default_wechat_grant_on_signup', 'false', datetime('now'))
ON CONFLICT (key) DO NOTHING;
