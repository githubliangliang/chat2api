-- [sqlite-converted] from PostgreSQL migration: 185_group_reasoning_effort_policy.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Add per-group controls for explicit OpenAI/Codex reasoning effort values.
ALTER TABLE groups ADD COLUMN max_reasoning_effort VARCHAR(20) NOT NULL DEFAULT '';
ALTER TABLE groups ADD COLUMN reasoning_effort_mappings TEXT NOT NULL DEFAULT '[]';
