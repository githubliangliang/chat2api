-- [sqlite-converted] from PostgreSQL migration: 136_usage_log_image_size_metadata.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Add generated-image billing size audit metadata.
-- `image_size` remains the canonical billing tier used for cost calculation.

ALTER TABLE usage_logs ADD COLUMN image_input_size VARCHAR(32);

ALTER TABLE usage_logs ADD COLUMN image_output_size VARCHAR(32);

ALTER TABLE usage_logs ADD COLUMN image_size_source VARCHAR(16);

ALTER TABLE usage_logs ADD COLUMN image_size_breakdown TEXT;

-- [sqlite] skipped PostgreSQL DO $$ ... $$ block


-- [sqlite] skipped PostgreSQL DO $$ ... $$ block

