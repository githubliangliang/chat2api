-- [sqlite-converted] from PostgreSQL migration: 201_channel_monitor_v2_refresh_5m.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Channel Monitor V2 is designed around fixed UI buckets; refresh passive
-- aggregation every 5 minutes and recompute the trailing bucket for each range.
-- Only migrate the untouched factory row.  A row with a newer version or an
-- updater identity reflects an operator choice and must retain its 60-second
-- interval (or any other valid setting).
UPDATE channel_monitor_v2_config
SET refresh_interval_seconds = 300,
    updated_at = datetime('now')
WHERE id = 1
  AND version = 1
  AND updated_by IS NULL
  AND refresh_interval_seconds = 60;
