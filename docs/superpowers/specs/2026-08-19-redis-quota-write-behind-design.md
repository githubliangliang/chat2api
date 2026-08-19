# Redis Quota Write-Behind Design

## Context

The SQLite deployment updates several quota counters after every successful
usage billing operation. User-platform quota usage already has an optional
Redis-first flusher, but it is disabled by default and can silently skip the
SQLite fallback on a Redis miss or error. Its absolute snapshot writer can also
race an administrator reset and restore stale usage.

API-key and Bedrock account quota usage is still read from and written to
`accounts.extra` inside the core usage billing transaction. That adds a JSON
read, decode, encode, and account update while SQLite is serializing writers.
Scheduler snapshots then carry a stale copy until the quota crosses a limit and
the account snapshot is refreshed.

The target deployment is a single application instance using SQLite and local
or embedded Redis. Account and user-platform quotas are operational limits, not
the financial ledger. The operator accepts losing at most one default flush
interval of these counters after an abnormal process or in-memory Redis loss.

## Goals

- Enable the existing user-platform quota write-behind path by default after
  making Redis failure and administrator mutation behavior safe.
- Move API-key and Bedrock account quota usage out of the successful core
  SQLite billing transaction when account quota write-behind is enabled.
- Make Redis the real-time enforcement source for both quota types while
  keeping SQLite as the recovery and reporting mirror.
- Keep account selection correct by overlaying current Redis quota state on
  scheduler snapshot candidates before quota filtering.
- Fall back to the existing SQLite paths when Redis is unavailable, stale, or
  malformed.
- Preserve the current behavior when either flusher is explicitly disabled.

## Non-Goals

- Moving balances, frozen balances, subscription billing, API-key total quota,
  usage deduplication, payments, or scheduler outbox state out of SQLite.
- Providing zero-loss durability for quota counters.
- Adding a durable queue, WAL, or another database.
- Rewriting scheduler snapshots or removing the final selected-account database
  recheck.
- Supporting multiple application processes against one SQLite database.

## Durability Boundary

The existing SQLite usage billing transaction remains authoritative for every
financial effect and for request deduplication. Redis quota increments happen
only after that transaction returns `Applied=true`.

With write-behind enabled, an abnormal exit can lose Redis quota increments
that have not reached SQLite. The expected bound is the configured flush
interval, 2 seconds by default. A smaller post-commit window also exists between
the SQLite commit and the Redis increment. Losing operations in either window
is an accepted operational-quota undercount. A duplicate billing request that
returns `Applied=false` does not increment Redis again.

## Architecture

### User-Platform Quota

The existing `BillingCache` user-platform quota operations and
`UserPlatformQuotaUsageFlusher` remain the owners of this path. Their contracts
are hardened rather than replaced.

- The Redis increment reports whether it was applied. A missing key, an old
  cache schema, or an epoch mismatch is not reported as success.
- The cache schema gains an administrator epoch separate from its existing
  per-increment revision counter.
- The SQLite row gains the same usage epoch. Limit changes and administrator
  resets advance the epoch and replace the Redis snapshot.
- Flusher snapshots include the epoch. An absolute usage snapshot can update a
  row only when its epoch still matches.
- Old Redis schema entries are treated as misses and rebuilt from SQLite.

This prevents an in-flight old snapshot from permanently undoing a reset or
limit mutation. The existing absolute snapshot and dirty-set model remains
idempotent.

### Account Quota

A dedicated `AccountQuotaCache` owns account quota hashes and a dirty account
set. It is separate from scheduler account JSON so hot quota writes cannot
overwrite credentials, model configuration, group state, or transient account
status.

Each account quota hash contains:

- cache schema version and quota usage epoch;
- total, daily, and weekly usage and limits;
- daily and weekly window start timestamps;
- fixed-window reset timestamps and reset modes needed by atomic increment;
- the account ID implicit in the key.

The quota usage epoch is stored in `accounts.extra`. Missing epochs from older
rows are interpreted as the initial epoch. Administrator changes to quota
limits, reset configuration, window state, or usage advance the epoch.

An atomic Redis script initializes a missing hash from the selected account
state, validates the epoch, uses Redis server time, applies rolling or fixed
window resets, increments the enabled dimensions, refreshes the TTL, adds the
account to the dirty set, and returns the complete post-increment quota state.
It rejects an older caller epoch instead of mutating a newer administrator
state.

`AccountQuotaUsageFlusher` follows the existing narrow-interface flusher
pattern. It pops a bounded dirty batch, reads current absolute states, and asks
an account quota snapshot writer to conditionally update only the quota usage
and window keys in `accounts.extra`. It never replaces the whole JSON object.
Failed batches are re-added for retry. Repeating a successful absolute snapshot
is harmless.

The snapshot writer enqueues the existing account-changed scheduler event when
a flushed state is over any configured limit. The event is committed with the
SQLite quota update. A best-effort account snapshot refresh may run after the
commit, matching the current quota-crossing latency optimization.

## Data Flow

### Successful Billing

When account quota write-behind is enabled, command construction omits the
account quota effect from the core SQLite usage billing transaction. All other
effects remain unchanged.

After an `Applied=true` result, one quota coordinator performs the following:

1. Increment account quota with the account snapshot, usage epoch, and
   quantized account cost through `AccountQuotaCache`.
2. Store the returned post-increment state in the billing result for threshold
   notification checks.
3. If Redis returns an error or rejects the state, call the existing SQLite
   account quota increment path, advancing the usage epoch.
4. Immediately refresh the scheduler account snapshot after a SQLite fallback
   and attempt to replace Redis with the returned database state.

The same coordinator is used by the legacy billing fallback so the two paths do
not drift. When account quota write-behind is disabled, account quota remains in
the current core transaction.

User-platform quota increments similarly distinguish an applied Redis update
from a miss or failure. A non-applied update uses the existing SQLite increment
path. With its flusher disabled, behavior stays on the current SQLite path.

Absolute snapshots make an ambiguous Redis timeout safe to reconcile. If Redis
actually applied the increment and SQLite fallback also ran, replacing SQLite
with the matching absolute state does not permanently double the amount.

### Account Selection

After `ListSchedulableAccounts` obtains candidates, the gateway performs one
chunked batch read of their account quota hashes. A pure overlay function copies
matching Redis usage and window values into candidate account copies. It never
mutates cached scheduler objects.

Only entries with a supported schema, valid fields, and an epoch equal to the
account snapshot are applied. Missing, corrupt, or mismatched entries leave the
snapshot unchanged. Existing `Account.IsQuotaExceeded` logic then handles total
limits and expired daily or weekly windows.

The final selected-account database recheck remains. Its fresh database account
is overlaid with one fresh Redis quota read before the final quota decision.
This is one batch operation plus at most one selected-account operation, never
one Redis request per candidate.

On a Redis read failure, selection logs a sampled warning and continues with
the scheduler snapshot and final database recheck. While Redis is unavailable,
new increments use SQLite and the existing immediate scheduler refresh path.

### Administrator Mutation

Administrator quota updates and resets perform the database mutation, epoch
advance, and scheduler outbox insert in one SQLite transaction. After commit,
they replace the Redis quota state with the new epoch and trigger the existing
best-effort scheduler synchronization.

If the post-commit Redis write fails, the database and scheduler event remain
authoritative. Old Redis data cannot overlay a fresh account because the epochs
differ. An old flusher snapshot cannot update the newer database epoch.

Account deletion removes its account quota hash and dirty-set member on a
best-effort basis. User-platform quota administration follows the same epoch
rule for its own cache and row.

## Failure Handling

- Redis increment error, miss, schema mismatch, or epoch mismatch: increment
  SQLite and reconcile Redis from the resulting database state.
- Redis selection read error: use snapshot plus final database recheck.
- Corrupt cache fields: ignore the entire entry; do not apply partial values.
- SQLite fallback failure after financial billing committed: record a
  high-priority error and metric, but do not roll back or duplicate the
  financial transaction.
- Flusher database failure: re-add the popped dirty members for the next tick.
- Dirty re-add failure: emit an alert and count the lost member. A later active
  increment re-adds it; low-activity SQLite mirrors may remain stale.
- Normal shutdown: stop recurring quota timers and run a final flush.
- Abnormal shutdown: accept the documented bounded operational-quota loss.

Repeated Redis errors are sampled in logs. Counters remain unsampled so an
outage is observable without log amplification.

## Configuration

The existing settings remain and change to enabled by default:

- `database.user_platform_quota_flusher_enabled: true`
- `database.user_platform_quota_flush_interval_ms: 2000`
- `database.user_platform_quota_flush_batch_size: 1000`

Account quota receives independent rollback controls:

- `database.account_quota_flusher_enabled: true`
- `database.account_quota_flush_interval_ms: 2000`
- `database.account_quota_flush_batch_size: 1000`

Account quota cache TTL defaults to one day and is refreshed on successful
increments. Configuration validation rejects a non-positive enabled interval or
batch size. Runtime code retains defensive 2-second and bounded-batch fallbacks.

Each flusher limits the number of batches consumed by one tick. This bounds a
SQLite transaction episode and yields to other work when the dirty set is
backlogged.

## Observability

The implementation exposes counters or gauges for:

- Redis quota increment success and error;
- non-applied increments and their reason;
- SQLite fallback attempts and failures;
- selection overlay hits, misses, version mismatches, and read failures;
- flush success, failure, batch size, maximum latency, and dirty re-add failure;
- dirty-set backlog.

The existing operations/health surface should expose both flusher metric sets.
Metric names follow current repository conventions; the design does not require
a new metrics framework.

## Implementation Stages

### Stage 1: User-Platform Quota

1. Add the usage epoch and cache schema upgrade.
2. Make Redis increment return applied versus non-applied.
3. Add SQLite fallback for Redis miss and error.
4. Make administrator mutation and flusher snapshots epoch-aware.
5. Enable the flusher by default and retain the disable switch.
6. Verify the change under SQLite and test Redis before starting Stage 2.

### Stage 2: Account Quota

1. Add the account quota cache, atomic increment, and repository tests.
2. Add conditional absolute snapshot persistence and the account flusher.
3. Route post-commit quota accounting through the coordinator.
4. Add batch and final-account selection overlays.
5. Make administrator mutation, reset, and deletion epoch-aware.
6. Enable the account flusher by default and run the full regression suite.

Stage 1 can be reverted independently by its existing flag. Stage 2 has a
separate flag and preserves the old transaction path when disabled.

## Testing

All production behavior is developed test-first.

Stage 1 tests cover:

- enabled defaults and invalid configuration fallback;
- Redis hit avoiding per-request SQLite quota persistence;
- Redis miss, old schema, epoch mismatch, and error taking one SQLite fallback;
- administrator reset interleaved with an old flusher snapshot;
- failed flush re-add and final shutdown flush;
- automatic rebuild from old cache schema.

Stage 2 tests cover:

- concurrent atomic total, daily, and weekly increments;
- rolling and fixed window reset behavior using Redis server time;
- `Applied=true` incrementing once and duplicate billing skipping increment;
- successful Redis operation leaving SQLite account quota unchanged until flush;
- Redis failure falling back to SQLite and reconciling the new epoch;
- idempotent absolute flush that preserves unrelated `accounts.extra` keys;
- old snapshots rejected after reset or quota configuration changes;
- scheduler outbox behavior when a flushed state exceeds a limit;
- batch selection overlay filtering an exhausted account;
- miss, corruption, epoch mismatch, and Redis outage selection behavior;
- one batch cache call for a candidate set, with no candidate N+1;
- deletion cleanup and normal shutdown flush.

SQLite plus test-Redis integration tests cover end-to-end increment, flush,
restart recovery, threshold crossing, and administrator races. Targeted race
tests exercise concurrent increment and flush. A write-pressure regression
asserts that a Redis-successful account increment does not update
`accounts.extra` in the core billing transaction.

Verification includes the affected service and repository packages, their
SQLite/Redis integration tests, and the complete backend test suite where the
environment supports it.

## Success Criteria

- With Redis healthy and both defaults enabled, user-platform quota and account
  quota usage do not cause per-request SQLite quota writes.
- Core financial billing and deduplication remain transactionally unchanged.
- Account selection filters an exhausted account from real-time Redis state
  without per-candidate Redis calls.
- Redis can be fully unavailable while requests continue through the existing
  SQLite paths.
- Administrator reset or configuration changes cannot be permanently undone by
  an older flusher snapshot.
- Flusher retries are idempotent and never replace unrelated account JSON.
- Abnormal process loss is limited to the documented non-financial quota
  window; normal shutdown attempts a final flush.
