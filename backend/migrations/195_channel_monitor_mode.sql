-- [sqlite-converted] from PostgreSQL migration: 195_channel_monitor_mode.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Channel monitor exclusive mode: v1 (active probes) or v2 (passive aggregation).
-- Default v1 preserves existing active-probe behavior; operators opt in to v2.
INSERT INTO settings (key, value, updated_at)
VALUES ('channel_monitor_mode', 'v1', datetime('now'))
ON CONFLICT (key) DO NOTHING;
