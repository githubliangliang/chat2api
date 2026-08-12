-- [sqlite-converted] from PostgreSQL migration: 067_add_account_load_factor.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
ALTER TABLE accounts ADD COLUMN load_factor INTEGER;
