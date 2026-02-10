.PHONY: help build test test-unit test-integration test-e2e ci-preflight ci-preflight-lite

help:
	@echo "Available targets:"
	@awk 'BEGIN {FS=":"} /^[a-zA-Z0-9][a-zA-Z0-9_-]*:/ {if ($$1 != ".PHONY") print "  " $$1}' Makefile

build:
	go build -o bin/server ./cmd/server

test:
	go test ./...
	golangci-lint run ./...

test-unit:
	go test -tags=unit ./...

test-integration:
	go test -tags=integration ./...

test-e2e:
	go test -tags=e2e ./...

ci-preflight:
	go test -tags=unit ./...
	go test -tags=integration ./...
	@if [ -x ./bin/golangci-lint ]; then ./bin/golangci-lint run --timeout=5m; else golangci-lint run --timeout=5m; fi

ci-preflight-lite:
	go test -tags=unit ./...
	@if [ -x ./bin/golangci-lint ]; then ./bin/golangci-lint run --timeout=5m; else golangci-lint run --timeout=5m; fi
