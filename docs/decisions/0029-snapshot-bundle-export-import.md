# ADR-0029: Snapshot Bundle Export And Import

Status: Accepted  
Date: 2026-07-14  
Related: [0012](0012-index-snapshot-sync-replication.md),
[0021](0021-snapshot-commit-failure-semantics.md),
[0026](0026-public-api-v1-freeze.md),
[0028](0028-source-tombstones.md),
stabilization gap **C2**

## Context

Moving a project between machines or restoring an index must not require
re-deriving the whole corpus blindly, and must never activate a corrupt or
partial snapshot. ADR-0012 defines `IndexSnapshot` as the sync unit; Lab Gate
had no portable bundle path.

## Decision

1. Portable unit is `snapshot-bundle-v1`: project identity, sealed ready
   `IndexSnapshot`, chunks for that snapshot, optional tombstoned source ids,
   and `bundle_checksum` (SHA-256 of the canonical payload excluding the
   checksum field).
2. Host paths (`CorpusRoot`, absolute artifact URIs) are **not** in the bundle.
3. `Verify` refuses import/activate when:
   - format version mismatches;
   - snapshot is not `ready`;
   - `bundle_checksum` mismatches;
   - chunks are empty/incomplete/duplicated;
   - recomputed `chunk_set_hash` ≠ sealed snapshot `ChunkSetHash`.
4. Active flip happens only after verify succeeds (`activate=true`). Verify-only
   mode never writes workspace state.
5. Activating into a workspace with a different `project_id` is refused.
6. Additive surfaces: CLI `snapshot-export` / `snapshot-import`, HTTP
   `POST /v1/snapshot/export|import`, `contextkit` helpers.

## Consequences

### Positive

- Projects can move index state without core rewrite.
- Corrupt/partial bundles cannot become `active_snapshot_id`.

### Negative

- Dense/sparse backend payloads are not in the v1 bundle; after move, rebuild
  dense/FTS when those backends are enabled (exact/in-memory path works from
  chunk text immediately).
- `source_merkle_root` is carried sealed; leaf recomputation needs corpus bytes
  (out of band).

### Follow-ups

- Include sparse/dense refs when measured (future-layer L03A).
- Project delete/export (C7).
