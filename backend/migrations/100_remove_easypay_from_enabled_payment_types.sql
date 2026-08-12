-- [sqlite-converted] from PostgreSQL migration: 100_remove_easypay_from_enabled_payment_types.sql
UPDATE settings
SET value = trim(REPLACE(REPLACE(REPLACE(value, 'easypay,', ''), ',easypay', ''), 'easypay', ''), ','),
    updated_at = datetime('now')
WHERE key = 'ENABLED_PAYMENT_TYPES'
  AND value LIKE '%easypay%';
