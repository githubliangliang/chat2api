-- [sqlite-converted] from PostgreSQL migration: 054_drop_legacy_cache_columns.sql
-- SQLite: best-effort drop of legacy cache column names (ignore if absent at runtime).
-- Columns cache_creation5m_tokens / cache_creation1h_tokens may not exist on fresh DBs.
SELECT 1;
