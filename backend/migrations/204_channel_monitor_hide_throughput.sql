-- [sqlite-converted] from PostgreSQL migration: 204_channel_monitor_hide_throughput.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Soft switch: hide RPM/TPM throughput rates on user-facing Channel Monitor V2.
-- Default false (rates visible). Admins always see full metrics.
INSERT INTO settings (key, value, updated_at)
VALUES ('channel_monitor_hide_throughput', 'false', datetime('now'))
ON CONFLICT (key) DO NOTHING;
