-- [sqlite-converted] from PostgreSQL migration: 102_add_out_trade_no_to_payment_orders.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 100_add_out_trade_no_to_payment_orders.sql
-- Adds out_trade_no column for external order ID used with payment providers.
-- Allows webhook handlers to look up orders by external ID instead of embedding DB ID.

ALTER TABLE payment_orders ADD COLUMN out_trade_no VARCHAR(64) NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS paymentorder_out_trade_no ON payment_orders (out_trade_no);
