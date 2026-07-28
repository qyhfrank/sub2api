#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
BACKEND_DIR="$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)"
REPO_DIR="$(CDPATH= cd -- "$BACKEND_DIR/.." && pwd)"
VERSION_FILE="$BACKEND_DIR/cmd/server/VERSION"

# Prefer the exact release tag when building from a tagged checkout so
# source builds from vX.Y.Z don't inherit the previous VERSION file value.
if command -v git >/dev/null 2>&1; then
  TAG="$(
    git -C "$REPO_DIR" describe --tags --exact-match --match 'v[0-9]*' 2>/dev/null || \
    git -C "$REPO_DIR" describe --tags --exact-match --match '[0-9]*' 2>/dev/null || \
    true
  )"
  if [ -n "$TAG" ]; then
    printf '%s\n' "${TAG#v}"
    exit 0
  fi
fi

BASE_VERSION="$(tr -d '\r\n' < "$VERSION_FILE")"
if [ -z "$BASE_VERSION" ]; then
  BASE_VERSION="0.0.0"
fi

COMMIT_SUFFIX=""
if command -v git >/dev/null 2>&1; then
  SHORT_COMMIT="$(git -C "$REPO_DIR" rev-parse --short=12 HEAD 2>/dev/null || true)"
  if [ -n "$SHORT_COMMIT" ]; then
    COMMIT_SUFFIX="+$SHORT_COMMIT"
  fi
fi

printf '%s-fork.dev%s\n' "$BASE_VERSION" "$COMMIT_SUFFIX"
