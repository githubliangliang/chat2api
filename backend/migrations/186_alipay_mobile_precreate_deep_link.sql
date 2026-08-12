-- [sqlite-converted] from PostgreSQL migration: 186_alipay_mobile_precreate_deep_link.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Mobile Alipay keeps the legacy WAP flow unless this opt-in is enabled.
INSERT INTO settings (key, value, updated_at)
VALUES ('ALIPAY_MOBILE_PRECREATE_DEEP_LINK', 'false', datetime('now'))
ON CONFLICT (key) DO NOTHING;
