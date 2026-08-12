-- [sqlite-converted] from PostgreSQL migration: 151_account_autopause_expiry_index_notx.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
CREATE INDEX IF NOT EXISTS idx_accounts_autopause_expiry_due
    ON accounts (expires_at)
    WHERE deleted_at IS NULL
      AND schedulable = TRUE
      AND auto_pause_on_expired = TRUE
      AND expires_at IS NOT NULL;
