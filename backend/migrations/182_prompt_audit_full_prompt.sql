-- [sqlite-converted] from PostgreSQL migration: 182_prompt_audit_full_prompt.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Retain the full (unredacted) prompt text on audit events so admins can
-- review the exact content that triggered a finding. Scoped to events only:
-- transient processing jobs keep storing redacted metadata.
ALTER TABLE prompt_audit_events ADD COLUMN full_prompt TEXT NOT NULL DEFAULT '';
