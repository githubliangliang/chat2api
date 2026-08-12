-- [sqlite-converted] from PostgreSQL migration: 137_redeem_code_expires_at.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Add optional expiry time for redeem codes themselves.
-- `validity_days` remains the subscription duration granted after redeeming.

ALTER TABLE redeem_codes ADD COLUMN expires_at TEXT;

CREATE INDEX IF NOT EXISTS idx_redeem_codes_expires_at
    ON redeem_codes (expires_at);
