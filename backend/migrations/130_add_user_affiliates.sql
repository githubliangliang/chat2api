-- [sqlite-converted] from PostgreSQL migration: 130_add_user_affiliates.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
CREATE TABLE IF NOT EXISTS user_affiliates (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    aff_code VARCHAR(32) NOT NULL UNIQUE,
    inviter_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    aff_count INTEGER NOT NULL DEFAULT 0,
    aff_quota DECIMAL(20,8) NOT NULL DEFAULT 0,
    aff_history_quota DECIMAL(20,8) NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_user_affiliates_inviter_id ON user_affiliates(inviter_id);
CREATE INDEX IF NOT EXISTS idx_user_affiliates_aff_quota ON user_affiliates(aff_quota);

-- [sqlite] skipped COMMENT ON table

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

