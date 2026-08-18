# Scheduler Outbox Atomic Mutations Design

## Context

Several account and group repository methods commit scheduler-relevant database
changes before enqueueing the corresponding scheduler outbox event. When SQLite
rejects the outbox insert, these methods log the error and still return success.
The database then contains the new state while scheduler account metadata or
bucket membership remains stale until a later full rebuild.

`BindGroups` already uses the desired pattern: the business mutation and outbox
insert share one transaction, and an outbox failure rolls back the mutation.

## Goal

Apply that atomicity guarantee to the approved high-risk create, group update,
and account recovery paths without redesigning the scheduler or adding a
general SQLite retry layer.

## Scope

The following repository methods are in scope:

- `accountRepository.Create`
- `groupRepository.Create`
- `groupRepository.Update`
- `accountRepository.ClearError`
- `accountRepository.SetSchedulable`
- `accountRepository.ClearRateLimitIfObserved`
- `accountRepository.ClearRateLimit`
- `accountRepository.ClearTempUnschedulable`
- `accountRepository.ClearModelRateLimits`
- `accountRepository.ClearAntigravityQuotaScopes`
- `accountRepository.ResetQuotaUsed`

The following remain out of scope:

- `AddToGroup` and `RemoveFromGroup`
- Group bulk binding and clearing
- Making account creation plus later group binding one service-level transaction
- Making the multi-step admin clear-error workflow one transaction
- A generic cross-repository transaction framework, SQLite retry behavior, or
  scheduler redesign

## Design

Each in-scope method opens an ent transaction when it does not already run in a
caller-owned transaction. It performs the existing business mutation using the
transaction client, enqueues the existing scheduler event through that same
client, and commits only after both operations succeed.

If a caller-owned transaction is present, the method reuses its client and
leaves commit ownership with the caller. Existing persistence error translation
and return values remain unchanged.

Any direct scheduler account-cache synchronization remains a latency
optimization. It runs only after a locally owned transaction commits and never
substitutes for a durable outbox event. When a caller owns the transaction, the
method does not publish uncommitted state to the cache.

A small repository-internal helper owns the repeated begin/reuse/commit/rollback
mechanics and reports whether it committed locally. It is intentionally limited
to this transaction boundary and does not abstract business mutations, outbox
events, retries, or cache behavior. Each mutation site still explicitly performs
its business write and outbox enqueue in the helper callback.

## Error Handling

- A business mutation failure returns the existing error and rolls back.
- An outbox insertion failure returns that error and rolls back the business
  mutation.
- A commit failure is returned and no cache synchronization is attempted.
- Cache synchronization failures retain their current best-effort logging
  behavior because the durable database and outbox state is already committed.

## Testing

SQLite repository tests create a real `scheduler_outbox` table and an insert
trigger that aborts outbox writes. Tests exercise representative mutation
shapes and assert observable database behavior:

- account and group creation return an error and leave no created record;
- group update returns an error and preserves the prior value;
- account recovery mutations return an error and preserve the prior status,
  schedulable, transient-limit, model-limit, or quota state as applicable.

Existing success-path integration tests continue to verify normal create,
update, and recovery behavior. The complete repository package must pass.

## Success Criteria

No in-scope method can report success after committing scheduler-relevant state
without also committing its scheduler outbox event. Forced outbox insertion
failures leave the corresponding account or group data unchanged, and no cache
write exposes uncommitted state.
