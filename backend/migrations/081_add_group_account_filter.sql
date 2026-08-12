-- [sqlite-converted] from PostgreSQL migration: 081_add_group_account_filter.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
ALTER TABLE groups ADD COLUMN require_oauth_only BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE groups ADD COLUMN require_privacy_set BOOLEAN NOT NULL DEFAULT false;
