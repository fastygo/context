# ADR-0008: Hybrid Index Architecture

Status: Accepted  
Date: 2026-06-17  
Related: [0004](0004-vector-namespace-abstraction.md), [0009](0009-context-sparse-tantivy-sidecar.md), [0011](0011-merkle-manifest-and-snapshot-namespace.md), [0012](0012-index-snapshot-sync-replication.md)

## Context

Plain vector RAG misses exact symbols, citations, morphology, and legal/scientific
spans. Cursor combines semantic index, grep/trigram, and Merkle sync. Archivarius
3000 is sparse/morphology-first with proprietary segment files. The Context core
targets hybrid retrieval with identical behavior locally and in cloud.

## Decision

1. **Three coordinated artifacts per logical index state:**
   - **Dense:** QDrant (semantic recall).
   - **Sparse:** Tantivy index bundle via `context-sparse` sidecar (lexical
     precision, BM25-style).
   - **Manifest:** `IndexSnapshot` record tying versions, Merkle roots, and
     index refs together.

2. **`IndexSnapshot` is the atomic unit of index consistency.** Search and ingest
   always bind to a `snapshot_id` (active or explicit).

3. **Commit protocol (two-phase):**
   - Build changed chunks → upsert QDrant + ingest Tantivy.
   - Publish snapshot row → flip `active_snapshot_id` only when both succeed.

4. **Go orchestrator (`context-core`)** owns manifest, Merkle diff, chunk pipeline,
   embed cache, and retrieval merge. It does **not** embed Tantivy natively.

5. Snapshot fields (minimum):

   ```text
   snapshot_id, project_id, parent_snapshot_id
   source_merkle_root, chunk_set_hash
   chunker_version, parser_version, embed_model_version, morph_version
   sparse_index_ref, vector_revision
   status: building | ready | superseded
   ```

## Consequences

### Positive

- Production-proven components (QDrant + Tantivy) with a clear version gate.
- Matches roadmap hybrid retrieval planner.

### Negative

- Operational complexity: two index backends must stay in sync per snapshot.
- Index rebuild required when chunker or embed model version changes (explicit in
  manifest).

### Follow-ups

- Retrieval planner parallel calls sparse + dense; merge by `chunk_id` (roadmap).
