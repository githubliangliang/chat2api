-- [sqlite-converted] from PostgreSQL migration: 135_allow_email_oauth_provider_types.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- [sqlite] DROP CONSTRAINT users_signup_source_check on users → try DROP INDEX
DROP INDEX IF EXISTS users_signup_source_check;

-- [sqlite] skipped ADD CONSTRAINT CHECK (not supported via ALTER TABLE)


-- [sqlite] DROP CONSTRAINT auth_identities_provider_type_check on auth_identities → try DROP INDEX
DROP INDEX IF EXISTS auth_identities_provider_type_check;

-- [sqlite] skipped ADD CONSTRAINT CHECK (not supported via ALTER TABLE)


-- [sqlite] DROP CONSTRAINT auth_identity_channels_provider_type_check on auth_identity_channels → try DROP INDEX
DROP INDEX IF EXISTS auth_identity_channels_provider_type_check;

-- [sqlite] skipped ADD CONSTRAINT CHECK (not supported via ALTER TABLE)


-- [sqlite] DROP CONSTRAINT pending_auth_sessions_provider_type_check on pending_auth_sessions → try DROP INDEX
DROP INDEX IF EXISTS pending_auth_sessions_provider_type_check;

-- [sqlite] skipped ADD CONSTRAINT CHECK (not supported via ALTER TABLE)

