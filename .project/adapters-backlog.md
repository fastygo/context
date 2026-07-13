# Adapter Backlog (Post–Chunk 10)

Ports are stable in `internal/retrieval` and `internal/models`. Live adapters
beyond PostgreSQL/pgvector wait for Chunk 12 proof or a measured blocker
([ADR-0017](decisions/0017-poc-backend-order.md)).

| Port | First live | Later adapters | Notes |
| --- | --- | --- | --- |
| `VectorStore` | `internal/retrieval/dense/postgresvector` | QDrant, Turbopuffer | Same `BackendCapabilities`; shared-collection + payload filters first |
| `SparseSearchClient` | fake / exact overlap | PostgreSQL FTS (Chunk 11+), then `context-sparse` (Tantivy) | Do not add Tantivy sidecar until lexical limits of Postgres FTS are measured |
| `Embedder` | `models/fake` (`fake-hash-v1`, dim 8) | Provider embedding adapters | Dimension change requires new `embedding_version` |
| Language | `linguistic/simple` | `context-lang-*` repositories | Carry language, token spans, analyzer version, expansions without changing vector adapters |
| `MetadataStore` | `internal/storage/postgres` | SQLite (optional), bbolt cache later | Migrations on Open; DocumentStore for lex/ling JSON |
| `ArtifactStore` | localfs | Object store | Unchanged by dense path |

## Capability contract

Adapters should implement `retrieval.CapabilityReporter` when they can declare:

- project/snapshot filter enforcement
- temporal / metadata filter enforcement (pgvector PoC: **false** — client index filters)
- dimension, metrics, namespace model, managed-service flag

## Trigger to add QDrant / Turbopuffer / context-sparse

1. Chunk 12 CLI proof with pgvector completed and limits recorded.
2. Superseding ADR updates PoC backend order if needed.
3. Contract tests pass against pgvector + the candidate adapter.
