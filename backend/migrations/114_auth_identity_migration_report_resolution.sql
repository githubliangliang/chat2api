-- [sqlite-converted] from PostgreSQL migration: 114_auth_identity_migration_report_resolution.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
ALTER TABLE auth_identity_migration_reports ADD COLUMN resolved_at TEXT NULL;

ALTER TABLE auth_identity_migration_reports ADD COLUMN resolved_by_user_id BIGINT NULL;

ALTER TABLE auth_identity_migration_reports ADD COLUMN resolution_note TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_auth_identity_migration_reports_resolved_at
    ON auth_identity_migration_reports (resolved_at);
