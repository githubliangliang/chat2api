-- [sqlite-converted] from PostgreSQL migration: 177_add_subscription_plan_currency.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Display-only ISO 4217 currency label for subscription plan prices; empty
-- keeps existing plans rendering without any label.
ALTER TABLE subscription_plans ADD COLUMN currency VARCHAR(3) NOT NULL DEFAULT '';
