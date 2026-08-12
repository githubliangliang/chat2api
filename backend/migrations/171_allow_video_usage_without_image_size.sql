-- [sqlite-converted] from PostgreSQL migration: 171_allow_video_usage_without_image_size.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Grok video generation stores billing_mode='video' and keeps image_count=1
-- only as a legacy media-unit counter. It must not be forced to carry an
-- image_size, because video pricing uses video_resolution/request metadata.

-- [sqlite] DROP CONSTRAINT usage_logs_image_billing_size_check on usage_logs → try DROP INDEX
DROP INDEX IF EXISTS usage_logs_image_billing_size_check;

-- [sqlite] skipped ADD CONSTRAINT

