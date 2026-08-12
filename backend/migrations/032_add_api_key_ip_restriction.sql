-- [sqlite-converted] from PostgreSQL migration: 032_add_api_key_ip_restriction.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Add IP restriction fields to api_keys table
-- ip_whitelist: JSON array of allowed IPs/CIDRs (if set, only these IPs can use the key)
-- ip_blacklist: JSON array of blocked IPs/CIDRs (these IPs are always blocked)

ALTER TABLE api_keys ADD COLUMN ip_whitelist TEXT DEFAULT NULL;
ALTER TABLE api_keys ADD COLUMN ip_blacklist TEXT DEFAULT NULL;

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

