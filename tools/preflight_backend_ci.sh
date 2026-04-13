#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
LINT_CMD=(go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.9.0 run --timeout=30m)

echo "==> verify go version"
(
  cd "$BACKEND_DIR"
  REQUIRED_GO=$(sed -n 's/^go //p' go.mod | head -1)
  go version | grep -q "go${REQUIRED_GO}" || {
    echo "go ${REQUIRED_GO} required, found $(go version)" >&2
    exit 1
  }
)

echo "==> golangci-lint"
(
  cd "$BACKEND_DIR"
  "${LINT_CMD[@]}"
)

echo "==> unit tests"
(
  cd "$BACKEND_DIR"
  make test-unit
)

echo "==> docker availability"
if ! docker info >/dev/null 2>&1; then
  echo "docker is required for backend integration tests" >&2
  echo "start Docker Desktop or another local Docker daemon, then rerun make preflight-backend-ci" >&2
  exit 1
fi

echo "==> integration tests"
(
  cd "$BACKEND_DIR"
  make test-integration
)

echo "==> backend build"
(
  cd "$BACKEND_DIR"
  go build ./...
)

echo "preflight passed"
