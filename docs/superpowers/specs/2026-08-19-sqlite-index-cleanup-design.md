# SQLite Index Cleanup Design

## Context

The local SQLite database contains historical Ent-generated indexes alongside
the canonical SQL-migration indexes. Schema inspection found 22 indexes whose
uniqueness, partial predicate, and indexed columns exactly duplicate another
index:

- 10 on `usage_logs`;
- 10 on `accounts`;
- 2 on `account_groups`.

This is primarily write amplification. Every insert or update must maintain all
duplicate B-trees, increasing WAL volume and the time for which SQLite holds its
single-writer lock. The database also has no `sqlite_stat1` table, so the query
planner has not persisted statistics.

## Goal

Reduce SQLite write amplification without changing query results, durability,
connection-pool behavior, scheduler outbox behavior, or application APIs.

## Design

Add one ordered SQLite migration after migration 223. It will:

1. Drop only the 22 indexes proven to be exact duplicates by live
   `PRAGMA index_list` and `PRAGMA index_info` inspection.
2. Retain the canonical `idx_*` migration indexes used by current query plans.
3. Run `PRAGMA optimize=0x10002` after the drops so SQLite considers every table
   when refreshing planner statistics on this first optimization pass.

The migration uses `DROP INDEX IF EXISTS`, making it safe for clean databases
that were never bootstrapped through the historical Ent schema path.

The following indexes are removed:

- `usagelog_created_at`
- `usagelog_api_key_id_created_at`
- `usagelog_subscription_id`
- `usagelog_group_id`
- `usagelog_user_id_created_at`
- `usagelog_model`
- `usagelog_account_id`
- `usagelog_api_key_id`
- `usagelog_user_id`
- `idx_usage_logs_billing_dedup_created_at`
- `account_rate_limit_reset_at`
- `account_rate_limited_at`
- `account_schedulable`
- `account_deleted_at`
- `account_last_used_at`
- `account_priority`
- `account_proxy_id`
- `account_status`
- `account_type`
- `account_platform`
- `accountgroup_priority`
- `accountgroup_group_id`

## Out Of Scope

- Removing overlapping but non-identical indexes.
- Dropping the legacy `account_overload_until` index, which targets the old
  `overload_until_old` column and is not an exact duplicate.
- Changing `synchronous`, WAL checkpoint settings, cache size, mmap size, busy
  timeout, or connection-pool limits.
- Changing cleanup batch sizes or enabling the quota flusher.
- Running `VACUUM`; the inspected database currently has no free-list pages.

## Testing

A real SQLite migration test will create canonical and historical duplicate
indexes, seed enough representative rows for analysis, apply the migration, and
verify:

- every listed historical index is absent;
- representative canonical indexes remain available;
- `sqlite_stat1` exists after optimization and contains analyzed table data;
- reapplying the migration statements is harmless.

The repository and complete backend test suites must remain green.

## Success Criteria

After migration, the 22 exact duplicate indexes no longer exist, canonical
indexes and query behavior remain intact, and SQLite has planner statistics.
The migration is idempotent and introduces no runtime configuration changes.
