-- [sqlite-converted] from PostgreSQL migration: 030_add_account_expires_at.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Add expires_at for account expiration configuration
ALTER TABLE accounts ADD COLUMN expires_at DATETIME;
-- Document expires_at meaning
-- [sqlite] skipped COMMENT ON column

-- Add auto_pause_on_expired for account expiration scheduling control
ALTER TABLE accounts ADD COLUMN auto_pause_on_expired boolean NOT NULL DEFAULT true;
-- Document auto_pause_on_expired meaning
-- [sqlite] skipped COMMENT ON column

-- Ensure existing accounts are enabled by default
UPDATE accounts SET auto_pause_on_expired = true;
