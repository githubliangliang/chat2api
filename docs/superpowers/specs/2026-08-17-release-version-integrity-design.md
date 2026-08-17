# Release Version Integrity

## Problem

The release workflow has two competing version sources: the Git tag and
`backend/cmd/server/VERSION`. The workflow temporarily overwrites the version
file through an artifact, while both GoReleaser configurations omit an
`-X main.Version=...` linker assignment. This makes the release binary depend on
the temporary file reaching the exact build context.

The workflow later writes the released version back to the default branch. That
write cannot change the already-created tag, can roll the default branch back
when an older tag is dispatched manually, and can race with another release.
The `v1.1.3` tag demonstrates the inconsistency: its embedded version file is
still `1.1.2`.

Manual dispatch also accepts an arbitrary checkout ref. A branch or commit can
therefore pass checkout even though GoReleaser requires release-tag semantics.

## Goals

1. Make the Git tag the sole authoritative version for release artifacts.
2. Accept only an existing tag for manual releases.
3. Build frontend, backend verification, and release artifacts from one
   immutable commit SHA.
4. Inject the GoReleaser version directly into `main.Version`.
5. Fail before publishing if a release-version smoke test does not report the
   requested version.
6. Remove default-branch version writes, including rollback and race hazards.
7. Keep these invariants covered by CI.

## Non-Goals

- Changing image names, architectures, registries, or release notes.
- Changing `backend/cmd/server/VERSION` for non-tag local builds.
- Changing the local SQLite Compose build process.
- Retagging or rewriting the existing `v1.1.3` tag.
- Redesigning backend or frontend test suites.

## Considered Approaches

### Tag-authoritative release with linker injection

Resolve the requested tag to a commit SHA in a preflight job, use that SHA in
all source checkouts, inject GoReleaser's `.Version` into `main.Version`, and
remove all release-time version-file mutation. This is the chosen approach
because it has one version source and no shared branch write.

### Keep the VERSION artifact and add validation

The workflow could retain the temporary artifact and only add a binary smoke
test. This would detect some failures but preserve two version sources and a
fragile cross-job file handoff.

### Require a VERSION bump before tagging

The workflow could reject tags whose committed version file differs from the
tag. This keeps tagged source archives self-consistent, but adds a manual release
step and makes existing stale tags impossible to re-run. It is not required for
correct release binaries once the tag is injected directly.

## Chosen Design

### Tag preflight

The Release workflow gains a `preflight` job. It checks out the repository with
full history, then resolves only `refs/tags/$RELEASE_TAG^{commit}`. Missing tags,
branch names, raw commit IDs, and malformed inputs fail in this job before any
build or registry login.

The resolved commit SHA is exposed as a job output. `build-frontend`,
`verify-backend`, and `release` depend on `preflight` and checkout that SHA. The
release checkout retains full history so GoReleaser can find the tag pointing at
HEAD and read annotated tag metadata.

Tag input enters shell only through the existing `RELEASE_TAG` environment
variable and remains quoted. No event expression containing tag data is inserted
directly into shell source.

### Version injection

Both `.goreleaser.yaml` and `.goreleaser.simple.yaml` add:

```text
- -X main.Version={{.Version}}
```

GoReleaser derives `.Version` from the validated tag. The existing commit, date,
and build-type linker assignments remain unchanged. The embedded VERSION file
continues to provide a fallback for builds that do not inject a version.

### Pre-publish smoke test

After downloading the frontend artifact and setting up Go, the release job
builds a temporary embedded server binary with `main.Version` set from the
validated release tag. It runs the binary with `--version` and requires the
reported version to match exactly. A mismatch stops the job before registry
login and GoReleaser publication.

The smoke binary is written under `RUNNER_TEMP` and is not uploaded.

### Remove version-file mutation

The `update-version` job, version-file artifact upload/download, and
`sync-version-file` job are removed. The release job no longer writes to the
default branch. This removes the possibility of an old manual release rolling
the branch version backward and eliminates cross-release push races.

### Regression coverage

A shell regression test under `deploy/tests` checks the release contract:

- both GoReleaser files inject `main.Version` from `.Version`;
- the workflow resolves an exact `refs/tags/` reference;
- source-consuming jobs checkout the preflight commit SHA;
- a pre-publish version smoke test is present;
- version artifact and default-branch synchronization jobs are absent.

The existing CI shell job runs this test. It must fail against the current
workflow before implementation and pass after the configuration changes.

## Error Handling

- A missing or non-tag manual input fails in `preflight`.
- A checkout that does not match the resolved SHA fails normally.
- A version mismatch fails before registry login and publication.
- Backend or frontend failures retain their existing release gates.
- Notification behavior remains best-effort and unchanged.

## Verification

1. Add the release-contract test and confirm it fails because exact-tag
   preflight, linker version injection, and smoke verification are missing.
2. Implement the workflow and GoReleaser changes and confirm the test passes.
3. Parse every workflow YAML file and run `bash -n` over every shell block.
4. Confirm all referenced Action tags exist remotely.
5. Build an embedded backend binary with `main.Version=1.1.3` and confirm
   `--version` reports `1.1.3`.
6. Run the existing deployment shell-test suite.
7. Confirm the final diff contains only the release contract, workflow,
   GoReleaser configurations, and this design document.

## Acceptance Criteria

- Manual input must resolve through `refs/tags/`, never as a branch or raw SHA.
- Frontend, backend tests, and release publication use the same immutable commit.
- Full and simple GoReleaser builds inject the tag-derived version.
- Publication cannot begin after a version smoke-test mismatch.
- The workflow does not upload, download, or synchronize a VERSION artifact.
- The release workflow never writes the default branch.
- CI prevents these invariants from regressing.
