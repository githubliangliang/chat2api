-- [sqlite-converted] from PostgreSQL migration: 103_add_allow_user_refund.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
ALTER TABLE payment_provider_instances ADD COLUMN allow_user_refund BOOLEAN NOT NULL DEFAULT false;
