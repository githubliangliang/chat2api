# SQLite Batch Usage Stats Fix

## Problem

SQLite production requests fail in two usage-statistics paths because SQL that
was originally written for PostgreSQL was only partially converted:

- Deployed builds predating commit `58a4cc1` still execute
  `percentile_cont(...) WITHIN GROUP` in the operations metrics collector.
- Batch API-key usage queries in older builds use `ANY($1)`, which SQLite does
  not provide.
- Current source expands IDs to an `IN` list, but its fixed `$2`, `$3`, and `$4`
  time placeholders overlap the variable-length ID placeholders.
- Current batch API-key and batch-user queries still call `LEAST`, which is not
  a built-in SQLite scalar function.

## Scope

Fix `GetBatchAPIKeyUsageStats` and `GetBatchUserUsageStats` in
`internal/repository/usage_log_repo_stats.go`. Do not change endpoint contracts,
result types, default date ranges, platform grouping, or success filtering.

The operations metrics latency code requires no source change because the
current implementation is already SQLite-compatible and covered by
`TestOpsMetricsCollectorQueryUsageLatencySQLite`. Deployment must rebuild from
the corrected source so that fix is included in the running binary.

## Design

For each batch query:

1. Normalize IDs and establish the default start/end times as today.
2. Generate the variable-length `IN` clause starting at `$1`.
3. Derive subsequent placeholder indexes from the number of generated ID
   arguments instead of embedding `$2`, `$3`, and `$4`.
4. Compute the earlier of `startTime` and `timezone.Today()` in Go and pass it
   as a normal lower-bound argument, removing `LEAST` from SQL.
5. Keep conditional aggregates in SQL. SQLite supports aggregate `FILTER`, so
   changing them would add unnecessary behavioral risk.

This retains numbered placeholders that work with both SQLite and PostgreSQL,
although SQLite is the configured production database.

## Verification

Add repository tests backed by an in-memory SQLite database and the real
migration schema. The tests will insert usage rows for multiple IDs across the
requested range and today's boundary, call both batch methods, and assert:

- multiple IDs bind to the intended placeholders;
- total cost respects `[startTime, endTime)`;
- today's cost respects the application day boundary;
- duplicate and invalid IDs remain normalized;
- IDs without usage remain present with zero values;
- batch-user platform grouping and success filtering remain intact.

Run the focused repository and service tests, then the broader backend test
suite if runtime permits. Verify the SQLite SQL audit also passes.

## Deployment Note

Rebuild and restart the production service from the fixed revision. Merely
restarting the existing artifact will leave both the old `ANY` query and the
old percentile query in place.
