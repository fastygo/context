#!/usr/bin/env bash
# Chunk 12 end-to-end proof helper.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

export CONTEXT_PG_DSN="${CONTEXT_PG_DSN:-postgres://context:context@127.0.0.1:5432/context?sslmode=disable}"

./scripts/dev.sh wait
go run ./cmd/context-dev proof-run --root "$ROOT" --out "$ROOT/.proofs"
