-- [sqlite-converted] from PostgreSQL migration: 192_group_profit_control.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Per-group profit control for scheduling admission.
-- Admission rule at request time: an account qualifies iff its cost multiplier
-- U (accounts.rate_multiplier) satisfies U <= D * (1 - margin - buffer), where
-- D is the requester's effective downstream multiplier at the request's
-- pricing instant.
ALTER TABLE groups ADD COLUMN profit_control_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE groups ADD COLUMN profit_min_margin DECIMAL(10,4) NOT NULL DEFAULT 0;
ALTER TABLE groups ADD COLUMN profit_safety_buffer DECIMAL(10,4) NOT NULL DEFAULT 0;
