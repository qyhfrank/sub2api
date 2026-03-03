.PHONY: help build generate test test-unit test-integration test-e2e ci-preflight ci-preflight-lite

VERSION ?= $(shell tr -d '\r\n' < ./cmd/server/VERSION)
LDFLAGS ?= -s -w -X main.Version=$(VERSION)

help:
	@echo "Available targets:"
	@awk 'BEGIN {FS=":"} /^[a-zA-Z0-9][a-zA-Z0-9_-]*:/ {if ($$1 != ".PHONY") print "  " $$1}' Makefile

build:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -trimpath -o bin/server ./cmd/server

generate:
	go generate ./ent
	go generate ./cmd/server

test:
	go test ./...
	golangci-lint run ./...

test-unit:
	go test -tags=unit ./...

test-integration:
	go test -tags=integration ./...

test-e2e:
	./scripts/e2e-test.sh

test-e2e-local:
	go test -tags=e2e -v -timeout=300s ./internal/integration/...
