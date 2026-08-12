-- [sqlite-converted] from PostgreSQL migration: 088_channel_billing_model_source_channel_mapped.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Change default billing_model_source for new channels to 'channel_mapped'
-- Existing channels keep their current setting (no UPDATE on existing rows)
-- [sqlite] skipped ALTER COLUMN SET DEFAULT

