-- [sqlite-converted] from PostgreSQL migration: 111_payment_routing_and_scheduler_flags.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
INSERT INTO settings (key, value, updated_at)
VALUES
	('payment_visible_method_alipay_source', '', datetime('now')),
	('payment_visible_method_wxpay_source', '', datetime('now')),
	('payment_visible_method_alipay_enabled', 'false', datetime('now')),
	('payment_visible_method_wxpay_enabled', 'false', datetime('now')),
	('openai_advanced_scheduler_enabled', 'false', datetime('now'))
ON CONFLICT (key) DO NOTHING;
