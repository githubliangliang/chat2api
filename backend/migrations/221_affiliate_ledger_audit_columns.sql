-- Add the SQLite ledger columns required by affiliate rebate and transfer queries.
-- Migration 134 remains unchanged because existing databases may have recorded its checksum.
ALTER TABLE user_affiliate_ledger ADD COLUMN source_order_id BIGINT NULL REFERENCES payment_orders(id) ON DELETE SET NULL;
ALTER TABLE user_affiliate_ledger ADD COLUMN balance_after DECIMAL(20,8) NULL;
ALTER TABLE user_affiliate_ledger ADD COLUMN aff_quota_after DECIMAL(20,8) NULL;
ALTER TABLE user_affiliate_ledger ADD COLUMN aff_frozen_quota_after DECIMAL(20,8) NULL;
ALTER TABLE user_affiliate_ledger ADD COLUMN aff_history_quota_after DECIMAL(20,8) NULL;

CREATE INDEX IF NOT EXISTS idx_user_affiliate_ledger_source_order_id
    ON user_affiliate_ledger(source_order_id)
    WHERE source_order_id IS NOT NULL;
