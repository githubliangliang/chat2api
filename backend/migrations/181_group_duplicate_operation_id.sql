-- [sqlite-converted] from PostgreSQL migration: 181_group_duplicate_operation_id.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Persist an internal operation identity for safely recovering an already
-- committed group duplicate when the idempotency response write is ambiguous.
ALTER TABLE groups ADD COLUMN duplicate_operation_id VARCHAR(64);

CREATE UNIQUE INDEX IF NOT EXISTS idx_groups_duplicate_operation_id_active
    ON groups (duplicate_operation_id)
    WHERE duplicate_operation_id IS NOT NULL AND deleted_at IS NULL;
