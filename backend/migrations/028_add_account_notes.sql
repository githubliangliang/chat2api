-- [sqlite-converted] from PostgreSQL migration: 028_add_account_notes.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 028_add_account_notes.sql
-- Add optional admin notes for accounts.

ALTER TABLE accounts ADD COLUMN notes TEXT;

-- [sqlite] skipped COMMENT ON column

