-- [sqlite-converted] from PostgreSQL migration: 160_add_user_frozen_balance.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
ALTER TABLE users ADD COLUMN frozen_balance DECIMAL(20,8) NOT NULL DEFAULT 0;
