-- [sqlite-converted] from PostgreSQL migration: 102_add_balance_notify_threshold_type.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Add threshold type support (fixed / percentage) to balance notification
ALTER TABLE users ADD COLUMN balance_notify_threshold_type VARCHAR(10) NOT NULL DEFAULT 'fixed';
-- Track cumulative recharge amount for percentage threshold calculation
ALTER TABLE users ADD COLUMN total_recharged DECIMAL(20,8) NOT NULL DEFAULT 0;
