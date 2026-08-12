-- [sqlite-converted] from PostgreSQL migration: 087_usage_log_billing_mode.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Add billing_mode to usage_logs (records the billing mode: token/per_request/image)
ALTER TABLE usage_logs ADD COLUMN billing_mode VARCHAR(20);
