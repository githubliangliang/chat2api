-- [sqlite-converted] from PostgreSQL migration: 069_add_group_messages_dispatch.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
ALTER TABLE groups ADD COLUMN allow_messages_dispatch BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE groups ADD COLUMN default_mapped_model VARCHAR(100) NOT NULL DEFAULT '';
