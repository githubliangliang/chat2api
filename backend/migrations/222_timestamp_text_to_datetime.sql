-- Convert TEXT timestamp columns that Ent/database/sql scans into *time.Time.
--
-- modernc.org/sqlite only parses TEXT values as time.Time when the declared
-- type is DATE/DATETIME/TIMESTAMP (_time_format=sqlite only affects writes).
-- 005/020/064 added these as TEXT during the PG→SQLite conversion. A non-null
-- value then breaks admin list endpoints:
--   sql: Scan error ... temp_unschedulable_until: unsupported Scan,
--   storing driver.Value type string into type *time.Time
--
-- Do not edit 005/020/064: existing DBs have already recorded their checksums.
-- SQLite cannot ALTER COLUMN TYPE, so rename + add DATETIME + copy + drop.

DROP INDEX IF EXISTS idx_accounts_temp_unschedulable_until;
ALTER TABLE accounts RENAME COLUMN temp_unschedulable_until TO temp_unschedulable_until_old;
ALTER TABLE accounts ADD COLUMN temp_unschedulable_until DATETIME;
UPDATE accounts SET temp_unschedulable_until = temp_unschedulable_until_old;
ALTER TABLE accounts DROP COLUMN temp_unschedulable_until_old;
CREATE INDEX IF NOT EXISTS idx_accounts_temp_unschedulable_until ON accounts(temp_unschedulable_until) WHERE deleted_at IS NULL;

DROP INDEX IF EXISTS idx_accounts_overload_until;
ALTER TABLE accounts RENAME COLUMN overload_until TO overload_until_old;
ALTER TABLE accounts ADD COLUMN overload_until DATETIME;
UPDATE accounts SET overload_until = overload_until_old;
ALTER TABLE accounts DROP COLUMN overload_until_old;
CREATE INDEX IF NOT EXISTS idx_accounts_overload_until ON accounts(overload_until);

ALTER TABLE accounts RENAME COLUMN session_window_start TO session_window_start_old;
ALTER TABLE accounts ADD COLUMN session_window_start DATETIME;
UPDATE accounts SET session_window_start = session_window_start_old;
ALTER TABLE accounts DROP COLUMN session_window_start_old;

ALTER TABLE accounts RENAME COLUMN session_window_end TO session_window_end_old;
ALTER TABLE accounts ADD COLUMN session_window_end DATETIME;
UPDATE accounts SET session_window_end = session_window_end_old;
ALTER TABLE accounts DROP COLUMN session_window_end_old;

ALTER TABLE api_keys RENAME COLUMN window_5h_start TO window_5h_start_old;
ALTER TABLE api_keys ADD COLUMN window_5h_start DATETIME;
UPDATE api_keys SET window_5h_start = window_5h_start_old;
ALTER TABLE api_keys DROP COLUMN window_5h_start_old;

ALTER TABLE api_keys RENAME COLUMN window_1d_start TO window_1d_start_old;
ALTER TABLE api_keys ADD COLUMN window_1d_start DATETIME;
UPDATE api_keys SET window_1d_start = window_1d_start_old;
ALTER TABLE api_keys DROP COLUMN window_1d_start_old;

ALTER TABLE api_keys RENAME COLUMN window_7d_start TO window_7d_start_old;
ALTER TABLE api_keys ADD COLUMN window_7d_start DATETIME;
UPDATE api_keys SET window_7d_start = window_7d_start_old;
ALTER TABLE api_keys DROP COLUMN window_7d_start_old;
