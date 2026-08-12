-- [sqlite-converted] from PostgreSQL migration: 028_add_usage_logs_user_agent.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Add user_agent column to usage_logs table
-- Records the User-Agent header from API requests for analytics and debugging

ALTER TABLE usage_logs ADD COLUMN user_agent VARCHAR(512);

-- Optional: Add index for user_agent queries (uncomment if needed for analytics)
-- CREATE INDEX IF NOT EXISTS idx_usage_logs_user_agent ON usage_logs(user_agent);

-- [sqlite] skipped COMMENT ON column

