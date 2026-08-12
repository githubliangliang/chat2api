-- [sqlite-converted] from PostgreSQL migration: 132_affiliate_custom_settings.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 邀请返利：用户专属配置增强
-- 1) aff_rebate_rate_percent: 用户作为邀请人时的专属返利比例（百分比，NULL 表示沿用全局比例）
-- 2) aff_code_custom: 标记当前 aff_code 是否被管理员手动改写过（用于"专属用户"列表筛选）

ALTER TABLE user_affiliates ADD COLUMN aff_rebate_rate_percent DECIMAL(5,2);

ALTER TABLE user_affiliates ADD COLUMN aff_code_custom BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_user_affiliates_admin_settings
    ON user_affiliates (updated_at)
    WHERE aff_code_custom = true OR aff_rebate_rate_percent IS NOT NULL;

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

