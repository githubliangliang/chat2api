-- [sqlite-converted] from PostgreSQL migration: 161_batch_image_pricing_snapshot.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
ALTER TABLE groups ADD COLUMN batch_image_discount_multiplier DECIMAL(10,4) NOT NULL DEFAULT 0.5;
ALTER TABLE groups ADD COLUMN batch_image_hold_multiplier DECIMAL(10,4) NOT NULL DEFAULT 0.6;

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column


ALTER TABLE batch_image_jobs ADD COLUMN base_unit_price DECIMAL(20,10) NOT NULL DEFAULT 0;
ALTER TABLE batch_image_jobs ADD COLUMN group_rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0;
ALTER TABLE batch_image_jobs ADD COLUMN account_rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0;
ALTER TABLE batch_image_jobs ADD COLUMN batch_discount_multiplier DECIMAL(10,4) NOT NULL DEFAULT 0.5;
ALTER TABLE batch_image_jobs ADD COLUMN hold_multiplier DECIMAL(10,4) NOT NULL DEFAULT 0.6;
ALTER TABLE batch_image_jobs ADD COLUMN billable_unit_price DECIMAL(20,10) NOT NULL DEFAULT 0;
ALTER TABLE batch_image_jobs ADD COLUMN hold_unit_price DECIMAL(20,10) NOT NULL DEFAULT 0;
ALTER TABLE batch_image_jobs ADD COLUMN pricing_snapshot_version INTEGER NOT NULL DEFAULT 0;

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

