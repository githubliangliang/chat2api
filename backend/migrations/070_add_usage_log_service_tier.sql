-- [sqlite-converted] from PostgreSQL migration: 070_add_usage_log_service_tier.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
ALTER TABLE usage_logs ADD COLUMN service_tier VARCHAR(16);

CREATE INDEX IF NOT EXISTS idx_usage_logs_service_tier_created_at
    ON usage_logs (service_tier, created_at);
