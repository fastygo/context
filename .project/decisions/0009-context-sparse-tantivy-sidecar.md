# ADR-0009: Sparse Search via `context-sparse` Sidecar

Status: Accepted  
Date: 2026-06-17  
Supersedes: Plan Chunk 01 item "in-memory sparse first"  
Related: [0008](0008-hybrid-index-architecture.md), [0010](0010-local-cloud-deployment-parity.md)

## Context

Tantivy is Rust. Options included: `anyproto/tantivy-go` (CGO + static `.a`),
Bleve (pure Go), Quickwit as a service, or a dedicated sidecar. The team decided
**not** to use `tantivy-go` and to run Tantivy in Docker from the start for
local/cloud parity and simpler Go CI (no Cargo in the core module).

## Decision

1. **`context-sparse`:** separate container/service wrapping Tantivy with a
   stable HTTP or gRPC API.
2. **Go core:** HTTP/gRPC client only; no CGO, no Rust toolchain in
   `github.com/fastygo/context`.
3. **Index layout on disk:**

   ```text
   /data/indexes/{project_id}/{snapshot_id}/   # immutable after commit
   /data/active/{project_id} → current snapshot_id
   ```

4. **Sidecar API (minimum):**
   - `POST /v1/index/{project_id}/ingest` — batch docs for a building snapshot
   - `POST /v1/index/{project_id}/commit` — seal snapshot, return bundle hash
   - `GET  /v1/search/{project_id}` — query + required `snapshot_id`
   - `POST /v1/snapshot/{project_id}/export` — bundle for replication

5. Document fields: `chunk_id`, `context_ref`, `text` (or stored field policy),
   tokenized fields for morphology-enriched terms; **no absolute host paths**.

6. Bleve remains a possible **test double** behind the same interface, not the
   production sparse engine.

## Consequences

### Positive

- Identical sparse behavior in docker-compose (laptop) and k8s (cloud).
- Process isolation: Tantivy merge crashes do not take down Go orchestrator.

### Negative

- Requires Docker (or a native `context-sparse` binary) even for local dev.
- Network hop latency; mitigate with localhost sidecar and connection pooling.

### Follow-ups

- OpenAPI/proto spec in `context-sparse` repo or `internal/indexing/sparse/` stub.
- Morphology token filters configured in sidecar from manifest `morph_version`.
