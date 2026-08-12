-- [sqlite-converted] from PostgreSQL migration: 154_account_spark_shadow.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 154_account_spark_shadow.sql
ALTER TABLE accounts ADD COLUMN parent_account_id BIGINT;
ALTER TABLE accounts ADD COLUMN quota_dimension VARCHAR(20) NOT NULL DEFAULT 'global';

-- 幂等加约束:维度合法 + 禁自指 + parent⟺非global 维度一致(评审 P1-d)
-- [sqlite] skipped PostgreSQL DO $$ ... $$ block


-- [sqlite] skipped VALIDATE CONSTRAINT

-- [sqlite] skipped VALIDATE CONSTRAINT

-- [sqlite] skipped VALIDATE CONSTRAINT

-- [sqlite] skipped VALIDATE CONSTRAINT

