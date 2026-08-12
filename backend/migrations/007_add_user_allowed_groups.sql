-- [sqlite-converted] from PostgreSQL migration: 007_add_user_allowed_groups.sql
-- Create join table that replaces legacy users.allowed_groups array column.
-- (Previous SQLite conversion incorrectly turned this into a no-op.)

CREATE TABLE IF NOT EXISTS user_allowed_groups (
    user_id BIGINT NOT NULL,
    group_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (user_id, group_id)
);

CREATE INDEX IF NOT EXISTS idx_user_allowed_groups_group_id
    ON user_allowed_groups (group_id);
