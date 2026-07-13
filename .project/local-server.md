# Local Server Environment (Chunk 09)

PostgreSQL with the `pgvector` extension is the first live dense-vector backend
for PoC hypothesis validation ([ADR-0017](decisions/0017-poc-backend-order.md)).
This note covers bootstrap only: compose, env, health checks, and storage
config structs. Domain table DDL stays deferred to Chunk 11.

## Quick start

```bash
cp .env.example .env   # optional local overrides; .env is gitignored
./scripts/dev.sh up
./scripts/dev.sh health
# or, if GNU Make is installed: make dev-up && make dev-health
```

Equivalent without helpers:

```bash
docker compose --env-file .env.example up -d
docker compose --env-file .env.example exec -T postgres \
  psql -U context -d context -c "SELECT extname FROM pg_extension WHERE extname='vector';"
```

Stop / reset:

```bash
./scripts/dev.sh down     # keep volume
./scripts/dev.sh reset    # remove volume (wipes local DB)
./scripts/dev.sh logs
./scripts/dev.sh ps
```

## Health checks

| Check | Command / expectation |
| --- | --- |
| Container healthy | `./scripts/dev.sh ps` / `pg_isready` succeeds |
| SQL connection | `SELECT 1` via `./scripts/dev.sh health` |
| Extension present | `pg_extension.extname = 'vector'` (init script + health re-check) |
| Dimension smoke | temporary `vector(8)` insert; `vector_dims = 8` |

`./scripts/dev.sh health` runs all of the above. Override dimension with
`EMBED_DIM=8 ./scripts/dev.sh health` (must match `CONTEXT_EMBEDDING_DIMENSION`).

## PoC embedding dimension

Until a live embedding adapter is wired (still fake per ADR-0017), the locked
contract is:

| Field | Value |
| --- | --- |
| `embedding_version` | `fake-hash-v1` |
| `dimension` | `8` (matches `retrieval/fake.HashEmbed` default used in unit tests) |
| `metric` | `cosine` |
| `collection` | `context_dense_v1` (logical namespace name; table DDL later) |

Changing dimension later requires a new `embedding_version` and a new vector
column/index; do not silently rewrite existing rows.

## Storage configuration

Go package `internal/config` exposes replaceable endpoint structs without
hardcoding a vector vendor in domain code:

- `MetadataStoreConfig` — `memory` now; `postgres` in Chunk 11
- `VectorStoreConfig` — `postgres_pgvector` for live PoC; `qdrant` kind reserved
- `SparseStoreConfig` — `memory` now; `postgres_fts` / `context_sparse` later
- `ArtifactStoreConfig` — `localfs` now; `object_store` later

Load with `config.LoadStorageConfigFromEnv()` or start from
`config.DefaultStorageConfig()`. Unit tests must not require Docker.

## Secrets

- Commit `.env.example` only.
- Never commit `.env` (already gitignored).
- Default local password `context` is for disposable compose data only.

## Durable CLI metadata (Chunk 13)

Opt-in durable writes for ingest / agent-run / trace:

```bash
export CONTEXT_PG_DSN='postgres://context:context@127.0.0.1:5432/context?sslmode=disable'
export CONTEXT_METADATA_KIND=postgres

go run ./cmd/context-dev init-project --root <corpus> --data <data> --project demo
go run ./cmd/context-dev ingest --data <data> --project demo
# project/sources/chunks/snapshot also survive in Postgres; state.json remains a cache

go run ./cmd/context-dev agent-run --data <data> --project demo --query '...'
go run ./cmd/context-dev trace --data <data> --project demo --run <id>
# trace reads ListTrace from Postgres when CONTEXT_METADATA_KIND=postgres
```

Without `CONTEXT_METADATA_KIND=postgres`, CLI stays offline (`state.json` + memory).

Integration:

```bash
CONTEXT_PG_DSN=... go test ./internal/devcli/ -run Durable -count=1
```

## Dense search (Chunk 10)

With the stack up:

```bash
export CONTEXT_PG_DSN='postgres://context:context@127.0.0.1:5432/context?sslmode=disable'
go run ./cmd/context-dev search --data <dir> --project <id> --query '...' --mode dense
# or: --mode hybrid-dense
# or: CONTEXT_ENABLE_DENSE=1 ... --mode hybrid
```

Integration tests:

```bash
CONTEXT_PG_DSN='postgres://context:context@127.0.0.1:5432/context?sslmode=disable' \
  go test ./internal/retrieval/dense/postgresvector/ -count=1
```

Without `CONTEXT_PG_DSN`, those tests skip and `go test ./...` stays green.

## Sparse FTS (Chunk 14)

With the stack up:

```bash
export CONTEXT_PG_DSN='postgres://context:context@127.0.0.1:5432/context?sslmode=disable'
export CONTEXT_SPARSE_KIND=postgres_fts
go run ./cmd/context-dev ingest --data <dir> --project <id>
go run ./cmd/context-dev search --data <dir> --project <id> --query '...' --mode sparse
# hybrid / hybrid-dense also use FTS when CONTEXT_SPARSE_KIND=postgres_fts
```

Integration tests:

```bash
CONTEXT_PG_DSN='postgres://context:context@127.0.0.1:5432/context?sslmode=disable' \
  go test ./internal/retrieval/sparse/postgresfts/ ./internal/devcli/ -run FTS -count=1
```

Default `CONTEXT_SPARSE_KIND` (unset/memory) keeps offline fake term-overlap.
FTS uses PostgreSQL `simple` text config + `ts_rank_cd` (no morphology; gate for
`context-sparse` / lang adapters).

## Metadata store (Chunk 11)

Migrations live in `internal/storage/postgres/migrations/` and apply on
`postgres.Open`. Reset local DB with `./scripts/dev.sh reset` then `up`.

```bash
export CONTEXT_PG_DSN='postgres://context:context@127.0.0.1:5432/context?sslmode=disable'
export CONTEXT_METADATA_KIND=postgres
go run ./cmd/context-dev meta-check --backend postgres

CONTEXT_PG_DSN=... go test ./internal/storage/postgres/ -count=1
```

Durable rows cover projects, sources/chunks (with temporal bounds), artifact
meta (`artifact_type` / `schema_id`), lineage, snapshots, packs, runs, tool
calls, traces, and adapter-neutral `meta_documents` for linguistic/lexicographic
JSON without importing language/dictionary adapters.

## Non-goals (still deferred)

- No QDrant / Turbopuffer / `context-sparse` services in compose.
- No production language/dictionary adapters in the storage layer
  (`meta_documents` stays JSON-neutral).
- `go test ./...` remains fully offline unless `CONTEXT_PG_DSN` is set.
