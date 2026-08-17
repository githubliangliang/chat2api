# SQLite Initialization and Release Safety

## Problem

The SQLite setup path creates the latest Ent schema, while normal application
startup subsequently replays every historical SQL migration. Migration
`008_seed_default_group.sql` inserts only the columns that existed when it was
written. Against the latest `groups` table, that insert violates the `NOT NULL`
constraints on four JSON columns that have Ent-level defaults but no database
defaults:

- `supported_model_scopes`
- `messages_dispatch_model_config`
- `models_list_config`
- `reasoning_effort_mappings`

The existing fresh-migration test does not reproduce this because it starts with
an empty database and applies SQL migrations in order. The four columns do not
exist when migration 008 runs in that path.

The release workflow can therefore publish an artifact that builds successfully
but cannot complete a fresh installation. Manual releases can also combine a
frontend from the workflow branch with a backend from the requested tag. Several
tag values and the annotated tag body are interpolated directly into shell
scripts, which can break shell parsing and provide a command-injection surface.

## Goals

1. Use one authoritative SQLite schema initialization path for setup and startup.
2. Reproduce the real setup-to-startup transition in an automated regression test.
3. Prevent a release when backend verification fails.
4. Build every component of a manual release from the requested tag.
5. Pass tag data to shell steps as data rather than executable shell source.

## Non-Goals

- Changing historical migration 008 or its checksum.
- Adding database defaults solely to accommodate the conflicting bootstrap path.
- Changing existing SQLite data or migration compatibility rules.
- Redesigning GoReleaser image and archive outputs.
- Refactoring unrelated CI, deployment, or notification behavior.

## Chosen Approach

SQL migrations remain the SQLite schema authority. Setup will open the configured
SQLite database, run `repository.ApplyMigrations`, and then run
`repository.EnsureSQLiteAuxTables`. It will no longer call Ent `Schema.Create`.
Normal startup will see the recorded migration checksums and safely skip the
already-applied migrations.

This is preferred over adding database defaults because it removes the conflicting
schema authorities and prevents the same class of failure when future required
fields are added. Special-casing migration 008 is rejected because it couples the
runner to one historical seed and leaves the underlying ordering problem intact.

## Components

### Setup initialization

`initializeSQLiteSchema` keeps the current directory creation, DSN, connection
limit, timeout, and error context. Its schema operation changes from Ent schema
creation to the same embedded migration runner used at application startup. The
auxiliary-table repair remains after migrations.

The obsolete Ent client and dialect imports are removed from the setup package.

### Regression coverage

An untagged setup-package test uses a temporary SQLite file and the real embedded
migrations. It performs setup initialization, opens the resulting database, and
runs the startup migration entry point again. It then verifies that the default
group exists and that all four required JSON values are valid and non-null.

Before the production change, the second migration pass must fail at migration
008 with the `supported_model_scopes` constraint error. After the change, both
passes and the field assertions must succeed.

### Release verification

The Release workflow gains a backend verification job that checks out the exact
release ref, installs the Go version declared by `backend/go.mod`, and runs the
backend unit-tagged and integration-tagged test targets. The publishing job depends
on this verification in addition to the version and frontend jobs, so GoReleaser
cannot push images or create a GitHub Release after a test failure.

### Release ref consistency

A workflow-level `RELEASE_TAG` selects the manual input for `workflow_dispatch`
and `github.ref_name` for tag pushes. Every checkout that consumes repository
source uses the same selected ref. Version extraction, tag-message lookup,
notifications, and version-file synchronization read `RELEASE_TAG` from the
environment rather than embedding the expression in shell source.

### Shell-safe tag handling

GitHub expressions that contain tag or tag-message data are assigned through step
environment variables. Shell code references those variables with normal quoting.
The multiline tag-message output uses a randomly generated delimiter so a tag body
cannot terminate the output block with a fixed `EOF` line. The Telegram step reads
the tag message from its environment instead of placing the message inside a
single-quoted shell assignment.

## Error Handling

- Setup continues to wrap database open, migration, and auxiliary-table failures
  with operation-specific context.
- Migration errors remain fatal; no new ignored-error category is introduced.
- Checkout fails clearly when a manually supplied release tag does not exist.
- A backend verification failure stops the release before registry login and
  publishing.
- Telegram remains best-effort and retains `continue-on-error` behavior.

## Verification

1. Run the new regression test before implementation and confirm the migration 008
   constraint failure.
2. Run it after implementation and confirm the default group plus all four JSON
   fields.
3. Run the affected setup and repository packages, then both backend CI targets.
4. Build the frontend production artifact and the embedded backend binary.
5. Run the built binary with `AUTO_SETUP=true`, a temporary data directory, and
   embedded Redis disabled; confirm it passes initialization and reaches its health
   endpoint.
6. Validate workflow YAML and inspect the release dependency/ref expressions.
7. Confirm the worktree contains only the intended changes plus the user's existing
   unrelated documentation edit.

## Acceptance Criteria

- A fresh SQLite auto-setup no longer fails when normal application startup applies
  migrations.
- Existing migration files and checksums are unchanged.
- The regression test fails on the old implementation and passes on the new one.
- Release publishing cannot start until backend verification succeeds.
- Manual releases build frontend and backend from the same requested tag.
- Tag names and annotated tag bodies are never interpolated directly into shell
  source.
