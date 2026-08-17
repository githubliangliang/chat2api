#!/usr/bin/env bash
set -euo pipefail

: "${RELEASE_TAG:?RELEASE_TAG is required}"
: "${GITHUB_OUTPUT:?GITHUB_OUTPUT is required}"

TAG_REF="refs/tags/$RELEASE_TAG"
if ! git show-ref --verify --quiet "$TAG_REF"; then
  echo "Release tag does not exist: $RELEASE_TAG" >&2
  exit 1
fi

if ! RELEASE_SHA="$(git rev-parse --verify "${TAG_REF}^{commit}")"; then
  echo "Release tag does not resolve to a commit: $RELEASE_TAG" >&2
  exit 1
fi

echo "release_sha=$RELEASE_SHA" >> "$GITHUB_OUTPUT"
echo "Resolved $RELEASE_TAG to $RELEASE_SHA"
