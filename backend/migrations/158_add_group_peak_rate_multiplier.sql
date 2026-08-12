-- [sqlite-converted] from PostgreSQL migration: 158_add_group_peak_rate_multiplier.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
ALTER TABLE groups ADD COLUMN peak_rate_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE groups ADD COLUMN peak_start VARCHAR(5) NOT NULL DEFAULT '';
ALTER TABLE groups ADD COLUMN peak_end VARCHAR(5) NOT NULL DEFAULT '';
ALTER TABLE groups ADD COLUMN peak_rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0;
