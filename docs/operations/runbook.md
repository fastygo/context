# Ops runbook (stabilization)

Hub for day-2 operations. Detailed env and flags:
[local-server.md](local-server.md). Scenario copy-paste: [scenarios/ops.md](../scenarios/ops.md).

## Ingest

```bash
go run ./cmd/context-dev init-project --data "$DATA" --project demo
# corpus under $DATA/corpus (or --path)
go run ./cmd/context-dev ingest --data "$DATA" --project demo
```

HTTP ingest uses relative `path_key` only (`POST /v1/ingest`).  
Optional dense/FTS: `CONTEXT_ENABLE_DENSE=1`, `CONTEXT_SPARSE_KIND=postgres_fts`.

After mass tombstones or parser upgrades, plan a rebuild (below).

## Rebuild / repair

```bash
go run ./cmd/context-dev repair --data "$DATA" --project demo \
  --mode rebuild --target all
# or: --mode retry-failed
curl -s -X POST http://127.0.0.1:8080/v1/repair \
  -d '{"project_id":"demo","mode":"rebuild","target":"all"}'
```

Rebuild does **not** clear `active_snapshot_id`. Search stays available on the
prior ready snapshot while `phase=rebuilding` ([ADR-0032](../decisions/0032-index-lifecycle-explain.md)).

## Restore snapshot (move project)

```bash
go run ./cmd/context-dev snapshot-export --data "$DATA_A" --project demo \
  --out /tmp/demo.bundle.json
go run ./cmd/context-dev snapshot-import --data "$DATA_B" --project demo \
  --in /tmp/demo.bundle.json --activate
```

Verify-before-activate: corrupt/partial bundles never flip active
([ADR-0029](../decisions/0029-snapshot-bundle-export-import.md)).  
Rebuild dense/FTS on the destination when those backends are enabled.

## Degraded modes

| Signal | Meaning | Action |
| --- | --- | --- |
| `GET /v1/index` → `degraded` | `last_failed` and/or tombstones | Inspect reasons; `retry-failed` or tombstone cleanup |
| `GET /v1/index` → `rebuilding` | Repair in progress | Search may still be available |
| `GET /v1/ready` → 503 | Backend unavailable | Check Postgres / env; failure injection off |
| Search `degraded:true` | Hybrid dense failed soft | Exact/sparse still used; fix vector path |
| Hard `dense` / `hybrid-dense` error | Required dense path down | Fix DSN/pgvector or change mode |

```bash
go run ./cmd/context-dev index-status --data "$DATA" --project demo
go run ./cmd/context-dev ready
curl -s "http://127.0.0.1:8080/v1/index?project_id=demo"
curl -s "http://127.0.0.1:8080/v1/ready"
```

## Related

- Tombstones: CLI `tombstone-source` / `POST /v1/sources/tombstone`
- Project archive/delete: [scenarios/ops.md](../scenarios/ops.md)
- Schedules: [scenarios/background-jobs.md](../scenarios/background-jobs.md)
- Power-user search (no Query AST): [search-power-user.md](../search-power-user.md)
