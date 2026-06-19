# ADR-0012: Index Snapshot Sync and Replication

Status: Accepted  
Date: 2026-06-17  
Related: [0008](0008-hybrid-index-architecture.md), [0010](0010-local-cloud-deployment-parity.md), [0011](0011-merkle-manifest-and-snapshot-namespace.md)

## Context

Archivarius 3000 uses portable proprietary index folders without native cloud
sync. Cursor replicates embedding index state via Merkle diff and server-side
namespace copy (simhash). Identical local/cloud search requires the **same index
snapshot** and **same query engine versions**, not independent reindex on each
side.

## Decision

1. **Sync unit = `IndexSnapshot`**, not live mutable indexes.
2. **Authority:** cloud holds canonical manifest chain + object store bundles;
   local pulls `active_snapshot_id` and associated artifacts.
3. **Replication flow:**

   ```text
   1. Client/source Merkle diff → changed files
   2. Chunk + embed + dual index write → new snapshot (building)
   3. Publish manifest when sparse + dense ready
   4. Replicate:
        - Tantivy bundle → S3/MinIO → local volume
        - QDrant → snapshot API or point export by snapshot_id
        - Morph blobs + manifest row → Postgres + local bbolt cache
   5. Local flips active_snapshot_id after verify (bundle hash, merkle_root)
   ```

4. **Do not** rely on "reindex from git on each machine" as the primary sync
   mechanism for production parity; use it only for disaster recovery or
   version migration.

5. **Source text authority** remains git/artifact store; indexes are derived
   and disposable given sources + manifest versions.

6. **Embed cache** (ADR-0007) accelerates sync but is not required for
   correctness — missing cache entries re-embed from `chunk_hash`.

## Consequences

### Positive

- Bit-identical ranking inputs when snapshot + engine versions match.
- Archivarius-like "portable index folder" story via Tantivy bundle export.

### Negative

- Bandwidth for full snapshot on first clone; incremental segment sync is P1.
- QDrant snapshot restore semantics must be validated per deployment mode.

### Follow-ups

- Bundle format hash in manifest (`sparse_index_ref`).
- rclone/S3 sync tooling or built-in `context sync pull` CLI command.
