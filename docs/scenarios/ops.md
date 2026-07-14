# Scenario: ops (metrics, quota, ready, repair)

## Readiness

```bash
go run ./cmd/context-dev ready
curl -s http://127.0.0.1:8080/health          # liveness + backends summary
curl -s "http://127.0.0.1:8080/v1/ready"      # 200 ready / 503 unavailable
```

Failure injection (offline): `CONTEXT_FAIL_METADATA|VECTOR|SPARSE|EMBEDDER|…=1`.

## Metrics and quotas

```bash
go run ./cmd/context-dev metrics --data "$DATA"
go run ./cmd/context-dev quota --data "$DATA"

export CONTEXT_QUOTA_MAX_PACKS=200
export CONTEXT_QUOTA_MAX_RUNS=100
# soft ask default 80%; hard deny at used >= max (HTTP 403)
```

## Index repair (ADR-0021)

```bash
go run ./cmd/context-dev repair --data "$DATA" --project demo \
  --mode rebuild|retry-failed --target all|dense|sparse

curl -s -X POST http://127.0.0.1:8080/v1/repair \
  -d '{"project_id":"demo","mode":"rebuild","target":"all"}'
```

## Snapshot move (ADR-0029 / stabilization C2)

```bash
go run ./cmd/context-dev snapshot-export --data "$DATA_A" --project demo \
  --out /tmp/demo.bundle.json
go run ./cmd/context-dev snapshot-import --data "$DATA_B" --project demo \
  --in /tmp/demo.bundle.json --activate

# HTTP: export returns bundle JSON; import verifies then optional activate
curl -s -X POST http://127.0.0.1:8080/v1/snapshot/export \
  -d '{"project_id":"demo"}'
```

Corrupt or partial bundles are refused; `active_snapshot_id` is never flipped
on verify failure. Rebuild dense/FTS after move when those backends are enabled.

## Project export / delete (ADR-0030 / stabilization C7)

```bash
go run ./cmd/context-dev project-export --data "$DATA" --project demo \
  --out /tmp/demo.archive.json
go run ./cmd/context-dev project-delete --data "$DATA" --project demo \
  --confirm demo
```

Delete tombstones sources first, then removes metadata (CASCADE), artifact
bytes, and workspace `state.json`. Always export before delete when retention
requires a copy.

## Eval history

```bash
go run ./cmd/context-dev eval --data "$DATA" --out .proofs/eval/report.json
go run ./cmd/context-dev eval-history --data "$DATA"
curl -s "http://127.0.0.1:8080/v1/metrics?project_id=demo"
curl -s "http://127.0.0.1:8080/v1/eval/history?limit=20"
```

Details: [operations/local-server.md](../operations/local-server.md).
