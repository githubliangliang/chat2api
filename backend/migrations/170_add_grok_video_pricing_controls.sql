-- [sqlite-converted] from PostgreSQL migration: 170_add_grok_video_pricing_controls.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Add independent group pricing controls for Grok video generation.
-- Video prices intentionally do not backfill from image prices: image and video
-- generation must be priced separately.

ALTER TABLE groups ADD COLUMN video_rate_independent BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE groups ADD COLUMN video_rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0;
ALTER TABLE groups ADD COLUMN video_price_480p DECIMAL(20,8);
ALTER TABLE groups ADD COLUMN video_price_720p DECIMAL(20,8);
ALTER TABLE groups ADD COLUMN video_price_1080p DECIMAL(20,8);

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

