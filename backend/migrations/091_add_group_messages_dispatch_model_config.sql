-- [sqlite-converted] from PostgreSQL migration: 091_add_group_messages_dispatch_model_config.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
ALTER TABLE groups ADD COLUMN messages_dispatch_model_config TEXT NOT NULL DEFAULT '{}';
