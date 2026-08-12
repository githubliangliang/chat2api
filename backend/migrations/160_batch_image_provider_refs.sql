-- [sqlite-converted] from PostgreSQL migration: 160_batch_image_provider_refs.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
ALTER TABLE batch_image_jobs ADD COLUMN provider_input_ref VARCHAR(1024);
ALTER TABLE batch_image_jobs ADD COLUMN provider_output_ref VARCHAR(1024);
