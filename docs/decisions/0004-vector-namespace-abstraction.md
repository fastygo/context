# ADR-0004: Vector Namespace Abstraction

Status: Accepted  
Date: 2026-06-17  
Related: [0008](0008-hybrid-index-architecture.md), [0011](0011-merkle-manifest-and-snapshot-namespace.md)

## Context

QDrant can isolate data by collection, by payload filter, or by collection alias
per snapshot. Cursor uses one namespace per workspace (~80M namespaces at scale).
Operational cost and query latency depend on the chosen layout. Domain code must
not hardcode collection names.

## Decision

1. Introduce a `VectorNamespace` (or `VectorStore`) interface in the indexing
   layer; QDrant is one adapter.
2. **Initial mapping (PoC):** shared collection (e.g. `context_dense_v1`) with
   strict payload filters: `project_id`, `snapshot_id` (or active snapshot only).
3. **Revisit trigger:** benchmark collection-per-project when project count,
   isolation, or delete semantics require it.
4. Point ID = stable `chunk_id`. Payload includes provenance fields from roadmap
   but **not** absolute filesystem paths (see ADR-0013).
5. Embed cache key = `chunk_hash`; namespace upsert skips unchanged hashes.

## Consequences

### Positive

- Collection strategy can change without rewriting retrieval planner.
- Aligns with incremental Merkle/chunk diff (ADR-0011).

### Negative

- Shared collection demands rigorous payload filters to prevent cross-project
  leakage; integration tests must assert boundary enforcement.

### Follow-ups

- QDrant adapter in Plan Chunk 05; namespace benchmark before production
  multi-tenant load.
