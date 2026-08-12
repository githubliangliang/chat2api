-- [sqlite-converted] from PostgreSQL migration: 144_add_opus48_to_model_mapping.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 为已持久化的 Antigravity model_mapping 添加 claude-opus-4-8。
--
-- 未持久化 model_mapping 的账号会直接使用 DefaultAntigravityModelMapping，
-- 因此这里只需要回填已有映射对象。

UPDATE accounts
SET credentials = json_set(
    credentials,
    '$.model_mapping.claude-opus-4-8',
    '"claude-opus-4-8"')
WHERE platform = 'antigravity'
  AND deleted_at IS NULL
  AND json_type(json_extract(credentials, '$.model_mapping')) = 'object'
  AND json_extract(credentials, '$.model_mapping')->>'claude-opus-4-8' IS NULL;
