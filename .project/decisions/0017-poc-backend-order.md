# ADR-0017: PoC Backend Order

Status: Accepted
Date: 2026-07-11
Supersedes: PoC first-stack ordering implied by ADR-0004, ADR-0008, ADR-0009,
and ADR-0010 (ports and long-term parity remain in force)
Related: [0002](0002-metadata-store-progression.md),
[0003](0003-artifact-store-progression.md),
[0004](0004-vector-namespace-abstraction.md),
[0007](0007-embedded-kv-intermediate-layers.md),
[0008](0008-hybrid-index-architecture.md),
[0009](0009-context-sparse-tantivy-sidecar.md),
[0010](0010-local-cloud-deployment-parity.md)

## Context

Earlier ADRs name QDrant and `context-sparse` (Tantivy) as production-shaped
backends for local/cloud parity. The 2026-06 planning correction narrowed the
**first live proof** to a single PostgreSQL service so Chunks 09–12 stay
operable without a multi-sidecar stack. Ports must stay replaceable; only the
default PoC adapter order changes.

## Decision

### 1. Ports stay normative; first live adapters are ordered

| Port | Chunks 02–08 | First live (09–12) | Later adapters |
|------|--------------|--------------------|----------------|
| `ArtifactStore` | local filesystem | local filesystem | S3-compatible object store |
| `MetadataStore` | in-memory (optional file-backed) | PostgreSQL | SQLite for single-node only if needed |
| `VectorStore` | fake / in-memory | **PostgreSQL + pgvector** | QDrant, Turbopuffer |
| `SparseSearchClient` | fake / exact lookup | **PostgreSQL full-text** or continue fake | `context-sparse` (Tantivy) |
| Models / embeddings | fake deterministic | fake until measured need | real provider adapters |
| Embedded KV (bbolt/Badger) | none | none | cache-only after measurement (ADR-0007) |

### 2. Normative rules

1. Domain packages never import pgvector, QDrant, Tantivy, bbolt, or provider SDKs.
2. Every dense/sparse query requires `project_id` and `snapshot_id`.
3. ADR-0010 parity means **same ports and algorithms**; compose topology for PoC
   may be one PostgreSQL container instead of QDrant + `context-sparse`.
4. ADR-0008 hybrid shape remains: dense + sparse/exact + `IndexSnapshot` manifest.
   For PoC, "sparse" may be PostgreSQL FTS or a fake client behind the same port.
5. QDrant / Turbopuffer / `context-sparse` may be added only after Chunk 12 proof
   (or a measured blocker) and a superseding ADR that records the trigger.
6. bbolt/Badger remain optional acceleration layers, not PoC requirements.

### 3. What this does **not** change

- `VectorNamespace` abstraction (ADR-0004).
- Dual Merkle + snapshot commit gate (ADR-0011, ADR-0021).
- Long-term option for Tantivy sidecar API shape (ADR-0009) as a **later**
  sparse engine.
- Local/cloud endpoint-style configuration (ADR-0010, ADR-0012).

### 4. Supersedes notes (ordering only)

| Prior wording | PoC replacement |
|---------------|-----------------|
| ADR-0008 "Dense: QDrant" | Dense: pgvector adapter first |
| ADR-0008 "Sparse: Tantivy" | Sparse: Postgres FTS or fake first |
| ADR-0009 "Tantivy from the start" | Tantivy after PoC unless measured need |
| ADR-0010 compose of QDrant + context-sparse | Compose PostgreSQL/pgvector for first proof |
| ADR-0004 "QDrant is one adapter" / PoC shared collection | Still true as adapter; first live adapter is pgvector |

## Consequences

### Positive

- One container for live PoC; lower ops cost for hypothesis validation.
- Contract tests can require two adapters later without rewriting domain models.

### Negative

- Postgres FTS is not a full BM25/morphology sparse engine; lexical limits must
  be recorded in Chunk 12 notes before promoting `context-sparse`.
- ADR-0008/0009 text still describes target production shape; readers must check
  this ADR for PoC order.

### Follow-ups

- Chunk 09 compose + health checks for PostgreSQL/pgvector only.
- After Chunk 12, decide whether to keep pgvector, add QDrant, and/or introduce
  `context-sparse` based on measured gaps.
