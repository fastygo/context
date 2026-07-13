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

## Non-goals (this chunk)

- No `VectorStore` / `MetadataStore` Postgres adapters (Chunks 10–11).
- No domain migrations, lineage tables, or sparse FTS schema.
- No QDrant / `context-sparse` services in compose.
- `go test ./...` remains fully offline.
