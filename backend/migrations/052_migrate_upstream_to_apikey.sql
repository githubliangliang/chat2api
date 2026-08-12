-- [sqlite-converted] from PostgreSQL migration: 052_migrate_upstream_to_apikey.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Migrate upstream accounts to apikey type
-- Background: upstream type is no longer needed. Antigravity platform APIKey accounts
-- with base_url pointing to an upstream sub2api instance can reuse the standard
-- APIKey forwarding path. GetBaseURL()/GetGeminiBaseURL() automatically appends
-- /antigravity for Antigravity platform APIKey accounts.

UPDATE accounts
SET type = 'apikey'
WHERE type = 'upstream'
  AND platform = 'antigravity'
  AND deleted_at IS NULL;
