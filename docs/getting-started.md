# Getting started

Goal: run ingest → search → context-pack on a tiny corpus without Docker.

## Prerequisites

- Go 1.22+ (see `go.mod`)
- Optional: Docker for Postgres/pgvector ([operations/local-server.md](operations/local-server.md))

## Minimal offline loop

```bash
ROOT=$(pwd)
CORPUS=$ROOT/tmp/corpus
DATA=$ROOT/tmp/data
mkdir -p "$CORPUS"
printf '# Hello\n\nExact token ZEBRA42 lives here.\n' > "$CORPUS/notes.md"

go run ./cmd/context-dev init-project --root "$CORPUS" --data "$DATA" --project demo
go run ./cmd/context-dev ingest --data "$DATA" --project demo
go run ./cmd/context-dev search --data "$DATA" --project demo --query 'ZEBRA42' --mode hybrid
go run ./cmd/context-dev context-pack --data "$DATA" --project demo --query 'ZEBRA42'
go run ./cmd/context-dev inspect --data "$DATA" --project demo --query 'ZEBRA42'
```

Operator search (phrases / boolean / morphology) uses `--mode query`:

```bash
go run ./cmd/context-dev search --data "$DATA" --project demo \
  --mode query --lang en --query '"Exact token" AND ZEBRA42'
```

See [search-operators.md](search-operators.md). JSON goes to stdout. Host paths
never appear in HTTP/Lab JSON (`path_key` only).

## HTTP (same workspace)

```bash
go run ./cmd/context-serve --data "$DATA" --addr 127.0.0.1:8080
curl -s "http://127.0.0.1:8080/health"
curl -s -X POST http://127.0.0.1:8080/v1/search \
  -H 'Content-Type: application/json' \
  -d '{"project_id":"demo","query":"ZEBRA42","mode":"hybrid"}'
```

Go client: `github.com/fastygo/context/pkg/contextkit` ([api/v1.md](api/v1.md)).

## Next

- Completer swap: `CONTEXT_COMPLETER_KIND=localecho`
- Dense / Postgres: [operations/local-server.md](operations/local-server.md)
- Operator search: [search-operators.md](search-operators.md)
- Recipes: [scenarios/](scenarios/README.md)
- Lab + Stabilization freeze: [lab-gate.md](lab-gate.md)
