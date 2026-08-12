-- [sqlite-converted] from PostgreSQL migration: 150_account_group_scheduler_indexes_notx.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
CREATE INDEX IF NOT EXISTS idx_account_groups_group_priority_account
    ON account_groups (group_id, priority, account_id);

CREATE INDEX IF NOT EXISTS idx_account_groups_account_priority_group
    ON account_groups (account_id, priority, group_id);
