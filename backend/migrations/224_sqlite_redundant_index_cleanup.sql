-- Remove exact duplicates left by historical Ent schema creation.
DROP INDEX IF EXISTS usagelog_created_at;
DROP INDEX IF EXISTS usagelog_api_key_id_created_at;
DROP INDEX IF EXISTS usagelog_subscription_id;
DROP INDEX IF EXISTS usagelog_group_id;
DROP INDEX IF EXISTS usagelog_user_id_created_at;
DROP INDEX IF EXISTS usagelog_model;
DROP INDEX IF EXISTS usagelog_account_id;
DROP INDEX IF EXISTS usagelog_api_key_id;
DROP INDEX IF EXISTS usagelog_user_id;
DROP INDEX IF EXISTS idx_usage_logs_billing_dedup_created_at;

DROP INDEX IF EXISTS account_rate_limit_reset_at;
DROP INDEX IF EXISTS account_rate_limited_at;
DROP INDEX IF EXISTS account_schedulable;
DROP INDEX IF EXISTS account_deleted_at;
DROP INDEX IF EXISTS account_last_used_at;
DROP INDEX IF EXISTS account_priority;
DROP INDEX IF EXISTS account_proxy_id;
DROP INDEX IF EXISTS account_status;
DROP INDEX IF EXISTS account_type;
DROP INDEX IF EXISTS account_platform;

DROP INDEX IF EXISTS accountgroup_priority;
DROP INDEX IF EXISTS accountgroup_group_id;

PRAGMA optimize=0x10002;
