-- [sqlite-converted] from PostgreSQL migration: 127_add_user_group_rpm_override.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 在已有的"用户专属分组倍率表"上扩展 rpm_override 列；同时放宽 rate_multiplier 为可空，
-- 使一行记录可以只覆盖 rate、只覆盖 rpm，或同时覆盖两者。
-- 语义：
--   - rate_multiplier NULL  → 该用户在此分组使用 groups.rate_multiplier 默认值
--   - rate_multiplier 非 NULL → 覆盖分组默认计费倍率
--   - rpm_override NULL     → 该用户在此分组使用 groups.rpm_limit 默认值
--   - rpm_override 非 NULL  → 覆盖分组默认 RPM（0 = 不限制）
-- 用户级 users.rpm_limit 仍独立生效（跨分组总配额）。
ALTER TABLE user_group_rate_multipliers ADD COLUMN rpm_override integer NULL;

-- [sqlite] skipped ALTER COLUMN DROP NOT NULL


-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

