#!/bin/sh
set -e

SPORED_DIR="$(cd "$(dirname "$0")/../spored" && pwd)"

echo "==> Cleaning build cache..."
go clean -cache -testcache

echo "==> Building..."
cd "$SPORED_DIR" && go build ./...

echo "==> Vetting..."
go vet ./...

echo "==> Testing..."
go test ./...

echo "==> All checks passed."
