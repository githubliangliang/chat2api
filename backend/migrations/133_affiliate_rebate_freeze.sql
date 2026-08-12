-- [sqlite-converted] from PostgreSQL migration: 133_affiliate_rebate_freeze.sql
-- 1) Add frozen quota column to user_affiliates for rebate freeze period.
ALTER TABLE user_affiliates ADD COLUMN aff_frozen_quota DECIMAL(20,8) NOT NULL DEFAULT 0;

-- 2) Add frozen_until column to user_affiliate_ledger for per-entry freeze tracking.
-- NULL = no freeze (or already thawed); non-NULL = frozen until this timestamp text.
ALTER TABLE user_affiliate_ledger ADD COLUMN frozen_until TEXT NULL;

-- 3) Partial index for efficient thaw queries (only rows still frozen).
CREATE INDEX IF NOT EXISTS idx_ual_frozen_thaw
    ON user_affiliate_ledger (user_id, frozen_until)
    WHERE frozen_until IS NOT NULL;
