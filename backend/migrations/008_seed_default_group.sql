-- [sqlite-converted] from PostgreSQL migration: 008_seed_default_group.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Seed a default group for fresh installs.
INSERT INTO groups (name, description, created_at, updated_at)
SELECT 'default', 'Default group', datetime('now'), datetime('now')
WHERE NOT EXISTS (SELECT 1 FROM groups);
