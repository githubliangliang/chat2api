#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
RESOLVER="$ROOT_DIR/deploy/resolve-release-tag.sh"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

if [[ ! -x "$RESOLVER" ]]; then
  fail "release tag resolver is missing or not executable: $RESOLVER"
fi

TEST_REPO="$(mktemp -d)"
trap 'rm -rf "$TEST_REPO"' EXIT

git -C "$TEST_REPO" init -q
git -C "$TEST_REPO" config user.name "Release Test"
git -C "$TEST_REPO" config user.email "release-test@example.com"
git -C "$TEST_REPO" commit --allow-empty -q -m "initial"
COMMIT_SHA="$(git -C "$TEST_REPO" rev-parse HEAD)"
OUTPUT_FILE="$TEST_REPO/github-output"

expect_rejected() {
  local input="$1"
  : > "$OUTPUT_FILE"
  if (cd "$TEST_REPO" && RELEASE_TAG="$input" GITHUB_OUTPUT="$OUTPUT_FILE" "$RESOLVER") >/dev/null 2>&1; then
    fail "expected release ref to be rejected: $input"
  fi
  if [[ -s "$OUTPUT_FILE" ]]; then
    fail "rejected release ref wrote a workflow output: $input"
  fi
}

expect_resolved() {
  local tag="$1"
  : > "$OUTPUT_FILE"
  (cd "$TEST_REPO" && RELEASE_TAG="$tag" GITHUB_OUTPUT="$OUTPUT_FILE" "$RESOLVER") >/dev/null
  if ! grep -Fxq "release_sha=$COMMIT_SHA" "$OUTPUT_FILE"; then
    fail "tag $tag did not resolve to its peeled commit"
  fi
}

git -C "$TEST_REPO" branch v1.2.2
expect_rejected "v1.2.2"
expect_rejected "$COMMIT_SHA"
expect_rejected "refs/heads/v1.2.2"

git -C "$TEST_REPO" tag v1.2.3
expect_resolved "v1.2.3"

git -C "$TEST_REPO" tag -a v1.2.4 -m "annotated release"
expect_resolved "v1.2.4"

echo "release tag preflight tests passed"
