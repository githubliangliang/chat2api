-- [sqlite-converted] from PostgreSQL migration: 031_add_ip_address.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Add IP address field to usage_logs table for request tracking (admin-only visibility)
ALTER TABLE usage_logs ADD COLUMN ip_address VARCHAR(45);

-- Create index for IP address queries
CREATE INDEX IF NOT EXISTS idx_usage_logs_ip_address ON usage_logs(ip_address);
