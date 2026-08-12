-- [sqlite-converted] from PostgreSQL migration: 117_add_payment_order_provider_snapshot.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
ALTER TABLE payment_orders ADD COLUMN provider_snapshot TEXT;
