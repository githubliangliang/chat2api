-- [sqlite-converted] from PostgreSQL migration: 162_add_group_batch_image_generation_gate.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
ALTER TABLE groups ADD COLUMN allow_batch_image_generation BOOLEAN NOT NULL DEFAULT false;

-- [sqlite] skipped COMMENT ON column

