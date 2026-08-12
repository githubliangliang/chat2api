-- [sqlite-converted] from PostgreSQL migration: 024_add_gemini_tier_id.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- -- +goose Up
-- -- +goose StatementBegin
-- 为 Gemini Code Assist OAuth 账号添加默认 tier_id
-- 包括显式标记为 code_assist 的账号，以及 legacy 账号（oauth_type 为空但 project_id 存在）
UPDATE accounts
SET credentials = json_set(
    credentials,
    '$.tier_id',
    '"LEGACY"')
WHERE platform = 'gemini'
  AND type = 'oauth'
  AND json_type(credentials) = 'object'
  AND json_extract(credentials, '$.tier_id') IS NULL
  AND (
    json_extract(credentials, '$.oauth_type') = 'code_assist'
    OR (json_extract(credentials, '$.oauth_type') IS NULL AND json_extract(credentials, '$.project_id') IS NOT NULL)
  );
-- -- +goose StatementEnd

-- -- [sqlite] skipped +goose Down section
