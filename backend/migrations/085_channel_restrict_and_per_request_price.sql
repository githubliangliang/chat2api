-- [sqlite-converted] from PostgreSQL migration: 085_channel_restrict_and_per_request_price.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Add model restriction switch to channels
ALTER TABLE channels ADD COLUMN restrict_models BOOLEAN DEFAULT false;

-- Add default per_request_price to channel_model_pricing (fallback when no tier matches)
ALTER TABLE channel_model_pricing ADD COLUMN per_request_price NUMERIC(20,10);
