# SQLite-Only SQL Native Compatibility Design

## Context

This fork uses SQLite as the personal-deployment database, but its runtime still
contains PostgreSQL-oriented connection branches and raw SQL. Recent fixes have
addressed individual failures in usage logging, OAuth refresh selection, and ops
queries. A repository scan still finds PostgreSQL constructs across many production
files, including `ANY` with `pq.Array`, `ILIKE`, PostgreSQL casts, `FOR UPDATE`,
array operators, and PostgreSQL-specific DML forms.

The product decision for this work is explicit: SQLite is the only supported
database target. PostgreSQL compatibility is not a requirement.

## Goals

1. Make fresh SQLite initialization and all embedded migrations fail-fast and
   deterministic.
2. Make production raw SQL executable by the configured modernc SQLite driver.
3. Remove untested PostgreSQL runtime paths and PostgreSQL-only dependencies where
   they are no longer used.
4. Add executable SQLite coverage for high-risk repository operations.
5. Add a static regression gate for PostgreSQL-only SQL constructs that SQLite
   cannot execute.
6. Preserve existing migration checksums; introduce new migrations for any schema
   correction required by the runtime fixes.

## Non-Goals

- Retaining PostgreSQL deployment support.
- Rewriting repositories wholesale with Ent.
- Refactoring business behavior unrelated to database compatibility.
- Changing existing migration contents merely to normalize formatting or syntax
  that already executes successfully on SQLite.
- Claiming that every occurrence of `$1`, `RETURNING`, `ON CONFLICT`, booleans, or
  aggregate `FILTER` is invalid. The bundled SQLite version supports these forms.

## Chosen Approach

Use SQLite-native SQL throughout the production path and enforce it through both
real execution tests and a narrowly defined static audit.

Extending the current PostgreSQL-compatibility function layer is rejected because
scalar functions can emulate names such as `NOW` or `GREATEST`, but cannot emulate
array binding, `ANY`, row locks, PostgreSQL casts, array operators, or complex DML
grammar. Migrating every raw query to Ent is also rejected because it would combine
a large architectural rewrite with the compatibility repair.

## Architecture

### Database configuration and startup

- SQLite becomes the sole accepted database driver and the effective default.
- Repository initialization opens only the modernc SQLite driver using the existing
  DSN pragmas, time format, foreign keys, WAL, and connection-pool policy.
- Setup CLI, setup HTTP endpoints, environment-driven setup, documentation, and
  deployment templates no longer advertise or validate PostgreSQL options.
- PostgreSQL connection construction and server-timing connector wiring are removed
  when no longer referenced. Server timing remains supported at the HTTP/service
  layer; only the PostgreSQL driver wrapper is out of scope for preservation.
- `prepareSchema` always follows the SQLite path.

### Migration behavior

- Existing migration files remain immutable so already initialized databases keep
  valid checksums.
- Fresh-database migration tests execute the complete embedded migration set against
  a temporary SQLite database and verify the critical schema objects.
- Upgrade tests start from representative existing SQLite states and run the same
  migration entry point.
- Required schema changes are introduced as new, ordered SQLite migrations.
- Migration syntax or checksum errors are fatal. Startup must not suppress a failed
  migration merely because a few core tables already exist.
- SQLite-specific idempotency handling remains limited to explicitly understood
  legacy bootstrap cases. It must not classify generic syntax errors as ignorable.

### Runtime SQL conversion

Queries are converted by behavior, not by blind text replacement:

- `column = ANY($n)` and `pq.Array` become expanded `IN` clauses and scalar
  arguments using the existing `sqlSliceIn`/`sqlInt64In` helpers.
- Empty slices use an explicit false predicate and never produce `IN ()`.
- `ILIKE` becomes `LIKE ... COLLATE NOCASE` where case-insensitive behavior is
  required. Escaping semantics remain unchanged.
- JSONB operators and casts become SQLite JSON1 expressions such as
  `json_extract`, `json_type`, and `json_each`.
- `FOR UPDATE` and advisory locks are removed or replaced by transaction designs
  valid under SQLite's single-writer semantics. Atomic conditional updates are
  preferred over read-then-write sequences.
- PostgreSQL `UPDATE ... FROM` or returning-old-row patterns become SQLite-safe
  transactions or conditional updates. Existing business invariants and returned
  values must remain covered by tests.
- Timestamp expressions use `CURRENT_TIMESTAMP` or bind Go `time.Time` values.
  PostgreSQL-shaped scalar compatibility functions are removed after their callers
  are converted.
- PostgreSQL array columns represented as SQLite `TEXT` store JSON. Reads and writes
  use structured JSON encoding and SQLite JSON1 table functions rather than lib/pq
  array encoders/scanners.
- Unique/constraint error translation uses modernc SQLite error codes or stable
  SQLite constraint classification rather than `pq.Error`.

Work proceeds by risk clusters: connection/migrations, shared SQL helpers and error
translation, identity/users/subscriptions/quotas, accounts/groups/channels, usage and
billing, ops/monitoring, payments/security-audit, and maintenance commands.

## SQL Audit Gate

A Go test scans production Go and SQL sources while excluding generated Ent code,
tests, documentation, and explicit audit fixtures. It rejects constructs known to be
PostgreSQL-only in this codebase:

- imports or uses of `github.com/lib/pq` in production database code;
- `ANY(...)`, `ALL(...)`, PostgreSQL `::type` casts, `ILIKE`, `FOR UPDATE`,
  `SKIP LOCKED`, `DISTINCT ON`, `NULLS FIRST/LAST`, `DATE_TRUNC`, `EXTRACT`,
  interval literals, PostgreSQL JSONB operators/functions, and advisory-lock SQL;
- PostgreSQL-only schema/type keywords in active migrations.

The audit does not reject SQLite-supported `$n` placeholders, `RETURNING`,
`ON CONFLICT`, `CURRENT_TIMESTAMP`, `TRUE/FALSE`, CTEs, window functions, or
aggregate `FILTER`. Rules must identify the file and matched token on failure and
support a small documented allowlist only when a construct is proven SQLite-safe or
appears outside executable SQL.

This gate is a regression detector, not the sole correctness proof. Dynamic SQLite
tests remain mandatory because syntax scanning cannot validate schema assumptions,
parameter binding, scan types, transaction behavior, or query results.

## Testing Strategy

### Test-first workflow

For each risk cluster, add the smallest real SQLite test that executes the failing
query and asserts its business result. Verify that it fails for the expected SQL or
binding reason before changing production code, then make the minimal production
change and rerun the cluster plus affected package tests.

Mocks may remain for unrelated dependencies, but SQL syntax tests must use a real
modernc in-memory or temporary-file SQLite database.

### Coverage layers

1. Static dialect audit: no prohibited PostgreSQL constructs in executable sources.
2. Migration integration: complete fresh initialization and representative upgrade.
3. Repository cluster tests: execute high-risk reads, writes, bulk operations,
   conflicts, empty lists, JSON fields, and transactional balance/quota paths.
4. Service tests: cover raw SQL issued outside repositories, especially payment and
   ops flows.
5. Full backend regression: run all normal, unit-tagged, and integration-tagged tests
   that do not require an external service.
6. Build verification: build the server and SQLite maintenance commands.

Because the current workspace has no `go` executable on `PATH`, implementation must
locate or provision the repository's required Go 1.26.5 toolchain before verification.
No completion claim is valid without recording the actual test and build results.

## Error Handling and Compatibility

- SQL errors retain operation context and wrap the underlying SQLite error.
- Constraint conflicts that are part of normal control flow are mapped explicitly;
  unexpected syntax, missing-object, and binding errors propagate.
- Existing SQLite data is preserved. Destructive database recreation is not part of
  the automated repair or test process.
- Any intentional behavior change discovered during conversion is stopped and
  reviewed separately rather than hidden inside a dialect patch.

## Documentation and Cleanup

After runtime conversion is green:

- update root and deployment documentation to describe SQLite as the only database;
- remove the PostgreSQL migration converter if it has no remaining supported use;
- remove unused lib/pq and PostgreSQL driver configuration only after production and
  test references are migrated;
- update misleading comments that claim dual-database portability.

Generated files are regenerated only when their source configuration or dependency
wiring actually changes.

## Acceptance Criteria

The work is complete when all of the following are demonstrated:

1. A new SQLite database applies every embedded migration without ignored syntax
   errors and contains the expected core and auxiliary schema.
2. A representative existing SQLite database upgrades without data loss.
3. The static SQL audit reports no prohibited PostgreSQL dialect in production paths.
4. High-risk raw SQL clusters have real SQLite execution tests, including empty-list,
   conflict, JSON, bulk update, and transaction cases.
5. The normal backend test suite and relevant tagged suites pass.
6. The server and SQLite maintenance commands build successfully.
7. Configuration, setup UI/API, templates, and documentation expose SQLite only.
8. No migration checksum was changed; schema fixes, if any, use new migrations.

## Implementation Sequencing

Implementation should be split into small test-driven batches. Each batch must leave
its affected packages green before the next cluster begins. The static audit is added
early with a baseline inventory, then tightened as each cluster is converted until
the allowlist is empty or contains only documented non-executable false positives.
