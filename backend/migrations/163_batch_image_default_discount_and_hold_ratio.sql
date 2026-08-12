-- [sqlite-converted] from PostgreSQL migration: 163_batch_image_default_discount_and_hold_ratio.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- [sqlite] skipped ALTER COLUMN SET DEFAULT


UPDATE groups
SET batch_image_discount_multiplier = 0.5
WHERE batch_image_discount_multiplier = 1.0;

UPDATE groups
SET batch_image_hold_multiplier = 0.6
WHERE batch_image_hold_multiplier = 1.05;

-- [sqlite] skipped COMMENT ON column


-- [sqlite] skipped ALTER COLUMN SET DEFAULT


-- [sqlite] skipped COMMENT ON column

