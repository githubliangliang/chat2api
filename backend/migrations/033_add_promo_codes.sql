-- [sqlite-converted] from PostgreSQL migration: 033_add_promo_codes.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 创建注册优惠码表
CREATE TABLE IF NOT EXISTS promo_codes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code VARCHAR(32) NOT NULL UNIQUE,
    bonus_amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    max_uses INT NOT NULL DEFAULT 0,
    used_count INT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    expires_at DATETIME DEFAULT NULL,
    notes TEXT DEFAULT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- 创建优惠码使用记录表
CREATE TABLE IF NOT EXISTS promo_code_usages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    promo_code_id BIGINT NOT NULL REFERENCES promo_codes(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    bonus_amount DECIMAL(20,8) NOT NULL,
    used_at DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE(promo_code_id, user_id)
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_promo_codes_status ON promo_codes(status);
CREATE INDEX IF NOT EXISTS idx_promo_codes_expires_at ON promo_codes(expires_at);
CREATE INDEX IF NOT EXISTS idx_promo_code_usages_promo_code_id ON promo_code_usages(promo_code_id);
CREATE INDEX IF NOT EXISTS idx_promo_code_usages_user_id ON promo_code_usages(user_id);

-- [sqlite] skipped COMMENT ON table

-- [sqlite] skipped COMMENT ON table

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

