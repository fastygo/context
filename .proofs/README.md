# Chunk 12+ proof artifacts

Neutral JSON produced by `context-dev proof-run` for Lab replay without live
services. Re-run:

```bash
./scripts/dev.sh up
export CONTEXT_PG_DSN='postgres://context:context@127.0.0.1:5432/context?sslmode=disable'
./scripts/proof-e2e.sh
# or: go run ./cmd/context-dev proof-run --root . --out .proofs
```

| File | Contents |
| --- | --- |
| `corpus/` | Ingested proof sources |
| `01-ingest.json` … `08-*.json` | Step artifacts |
| `SUMMARY.md` / `SUMMARY.json` | Hypothesis status |
| `eval/` | Golden catalog + report |
| `14-sparse-fts-limits.md` | FTS limits notes |

Docs: [docs/](../docs/README.md). Ops: [docs/operations/local-server.md](../docs/operations/local-server.md).
