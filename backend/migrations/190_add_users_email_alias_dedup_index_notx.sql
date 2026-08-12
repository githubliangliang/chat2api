-- [sqlite-converted] from PostgreSQL migration: 190_add_users_email_alias_dedup_index_notx.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Registration alias dedup (repository.existsByEmailAliasWithClient) probes users
-- by the dot-stripped email form, on the public register / send-verify-code paths.
-- Index that exact expression so the probes stay index lookups instead of a
-- sequential scan. serves both the equality probe and the
-- "local+%@domain" prefix probe regardless of database collation.
CREATE INDEX IF NOT EXISTS idx_users_email_dot_stripped
    ON users ((REPLACE(LOWER(TRIM(email)), '.', '')))
    WHERE deleted_at IS NULL;
