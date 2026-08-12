-- [sqlite-converted] from PostgreSQL migration: 112_add_payment_order_provider_key_snapshot.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
ALTER TABLE payment_orders ADD COLUMN provider_key VARCHAR(30);

UPDATE payment_orders
SET provider_key = (
    SELECT provider_key
    FROM payment_provider_instances
    WHERE CAST(id AS TEXT) = payment_orders.provider_instance_id
)
WHERE provider_key IS NULL
  AND provider_instance_id IS NOT NULL;
