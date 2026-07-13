# ADR-0021: Snapshot Commit Failure Semantics

Status: Accepted
Date: 2026-07-11
Related: [0008](0008-hybrid-index-architecture.md),
[0011](0011-merkle-manifest-and-snapshot-namespace.md),
[0012](0012-index-snapshot-sync-replication.md),
[0017](0017-poc-backend-order.md)

## Context

Hybrid indexing writes metadata, dense vectors, and sparse/exact structures.
Partial success must never become the active search generation. Chunk 04+ needs
minimal state and idempotency rules before commit code exists.

## Decision

### 1. Snapshot states

| State | Meaning | Searchable as active? |
|-------|---------|------------------------|
| `building` | Commit in progress; indexes may be partial | No |
| `ready` | All required writers succeeded; sealed | Yes (if selected active) |
| `failed` | Commit aborted; partial data must not be used | No |
| `superseded` | Was ready; replaced by a newer ready snapshot | No (except explicit pin/replay) |

ADR-0008's `building | ready | superseded` is extended with `failed`.

### 2. Active pointer rules

1. `active_snapshot_id` for a project points only at a snapshot in `ready`.
2. Flip is atomic in the metadata store (single transactional update).
3. Search defaults to `active_snapshot_id` unless a caller pins a `snapshot_id`
   for replay/eval; pinned `failed`/`building` snapshots are rejected.
4. A `ready` snapshot becomes `superseded` when a newer `ready` snapshot is
   activated (optional retain for rollback/eval).

### 3. Two-phase commit (minimum)

```text
1. Create snapshot row: status=building, parent_snapshot_id, versions, roots TBD
2. Write artifacts/chunks/metadata for this snapshot_id only
3. Write dense vectors for this snapshot_id only (VectorStore)
4. Write sparse/exact structures for this snapshot_id only
5. Verify counts/hashes against manifest expectations
6. Set status=ready (seal)
7. Flip active_snapshot_id -> this snapshot_id
```

If any step before seal fails: set `status=failed`, record `failure_reason`,
**do not** flip `active_snapshot_id`. Prior active remains searchable.

### 4. Partial write isolation

1. All dense/sparse upserts for a commit use the building `snapshot_id`.
2. Readers with `snapshot_id=active` never see building rows.
3. Failed snapshot data may be deleted by GC later; until GC, queries must still
   filter it out by status.
4. Never reuse a `failed` snapshot_id for a new attempt; allocate a new id.

### 5. Idempotency

1. Re-running commit for the same building id is allowed only while `building`,
   and writers must be idempotent on `(snapshot_id, chunk_id)` / chunk_hash keys.
2. After `ready` or `failed`, mutation of index payloads for that id is forbidden
   (read-only seal).
3. Retry after failure starts a **new** `building` snapshot with a new id,
   optionally copying unchanged chunk hashes from parent (incremental).
4. Manifest fields `source_merkle_root` and `chunk_set_hash` are written before
   seal and are immutable thereafter.

### 6. Required writers for PoC seal

Per ADR-0017, seal requires success of:

- metadata manifest + chunk records
- exact index path used by PoC (in-memory/exact structures acceptable in 02–08)
- dense writer when dense is enabled for the snapshot; if dense is disabled in a
  sparse-only test profile, snapshot must record `dense_enabled=false`
- sparse/FTS writer when sparse is enabled; else `sparse_enabled=false`

A snapshot cannot be `ready` if a declared-enabled writer failed.

### 7. Failure reasons (minimum codes)

`validation_error`, `artifact_missing`, `hash_mismatch`, `dense_write_failed`,
`sparse_write_failed`, `metadata_write_failed`, `verify_failed`, `cancelled`.

Traces record snapshot_id, transition, and failure code.

## Consequences

### Positive

- Active search stays consistent under crash mid-commit.
- Tests can assert failed snapshots never appear in default search.

### Negative

- Failed snapshots need later GC to reclaim vector/sparse space.
- True distributed two-phase commit across services is still best-effort in PoC;
  metadata status is the gate.

### Follow-ups

- Chunk 04 implements state machine + golden transition tests.
- Future-layer: retention/GC and cross-service snapshot export verification
  (ADR-0012).
