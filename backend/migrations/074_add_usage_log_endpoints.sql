-- [sqlite-converted] from PostgreSQL migration: 074_add_usage_log_endpoints.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Add endpoint tracking fields to usage_logs.
-- inbound_endpoint: client-facing API route (e.g. /v1/chat/completions, /v1/messages, /v1/responses)
-- upstream_endpoint: normalized upstream route (e.g. /v1/responses)
ALTER TABLE usage_logs ADD COLUMN inbound_endpoint VARCHAR(128);
ALTER TABLE usage_logs ADD COLUMN upstream_endpoint VARCHAR(128);
