# Scenario: ingest → search → pack

## CLI

```bash
go run ./cmd/context-dev ingest --data "$DATA" --project demo
go run ./cmd/context-dev search --data "$DATA" --project demo \
  --query 'ZEBRA42' --mode hybrid
go run ./cmd/context-dev context-pack --data "$DATA" --project demo \
  --query 'ZEBRA42'
go run ./cmd/context-dev inspect --data "$DATA" --project demo \
  --query 'ZEBRA42'
```

Modes: `exact` | `sparse` | `hybrid` | `dense` | `hybrid-dense` | `query`.

Operator example:

```bash
go run ./cmd/context-dev search --data "$DATA" --project demo \
  --mode query --lang en --query '"Exact token" AND ZEBRA42'
```

Details: [search-operators.md](../search-operators.md).

Optional focus:

```bash
go run ./cmd/context-dev focus-put --data "$DATA" --project demo --json '{...}'
go run ./cmd/context-dev search ... --focus focus_id
```

## HTTP

```bash
curl -s -X POST http://127.0.0.1:8080/v1/search \
  -H 'Content-Type: application/json' \
  -d '{"project_id":"demo","query":"ZEBRA42","mode":"hybrid"}'

curl -s -X POST http://127.0.0.1:8080/v1/context-pack \
  -H 'Content-Type: application/json' \
  -d '{"project_id":"demo","query":"ZEBRA42"}'

curl -s -X POST http://127.0.0.1:8080/v1/inspect \
  -H 'Content-Type: application/json' \
  -d '{"project_id":"demo","query":"ZEBRA42"}'
```

Ingest over HTTP uses **relative** `path_key` only:

```bash
curl -s -X POST http://127.0.0.1:8080/v1/ingest \
  -d '{"project_id":"demo","path_key":"notes.md"}'
```

## Notes

- Inspect shows budget, selected/rejected, scores; `surface_preview` is redacted.
- Dense requires live Postgres — see [operations/local-server.md](../operations/local-server.md).
