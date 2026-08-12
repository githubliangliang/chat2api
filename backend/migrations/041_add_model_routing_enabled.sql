-- [sqlite-converted] from PostgreSQL migration: 041_add_model_routing_enabled.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Add model_routing_enabled field to groups table
ALTER TABLE groups ADD COLUMN model_routing_enabled BOOLEAN NOT NULL DEFAULT false;
