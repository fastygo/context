# ADR-0011: Merkle Manifest and Snapshot Namespace

Status: Accepted  
Date: 2026-06-17  
Related: [0004](0004-vector-namespace-abstraction.md), [0008](0008-hybrid-index-architecture.md), [0013](0013-context-ref-and-path-alias.md)

## Context

Cursor uses client/server Merkle trees for incremental codebase indexing, plus
per-workspace vector namespaces. Smart code chunking means file-level checksums
alone are insufficient: a one-line edit can reshuffle many chunks. Namespaces
must not conflate tenant boundary, index version, and AI-facing aliases.

## Decision

### Separate namespace concepts

| Concept | Purpose |
|---------|---------|
| `ProjectID` | Tenant ACL, quota, policy |
| `SnapshotID` | Immutable index generation; version gate for search |
| `VectorNamespace` | QDrant partition (ADR-0004) |
| `SparseIndexRef` | Tantivy bundle URI + generation |
| `ContextRef` | Short alias for model prompts (ADR-0013) |

### Dual Merkle model

**Level A — Source tree (files)**

- Leaf: `SHA256(normalized_bytes ‖ path_key ‖ source_type)`
- `path_key` = stable logical key (hash of `project_id + relative_path`), not
  absolute filesystem path.
- Internal nodes: deterministic sorted-child hashing (Git-tree style).
- Root: `source_merkle_root` → cheap "what files changed" diff.

**Level B — Chunk set**

- After parse/chunk: each chunk → `chunk_hash = H(chunker_v, span, normalized_text, …)`
- `chunk_set_hash` = Merkle over sorted chunk hashes.
- Re-index: file diff → re-chunk changed files only → embed/index changed
  `chunk_hash`es.

### Indexing loop

```text
Scan → MerkleDiff → ChunkDiff → EmbedMissing → DualWrite(sparse, dense)
  → CommitSnapshot → flip active_snapshot_id
```

### Manifest versioning

Any change to `chunker_version`, `parser_version`, `embed_model_version`, or
`morph_version` requires a new snapshot (full re-chunk/re-embed as defined by
that version bump).

## Consequences

### Positive

- Incremental indexing at file and chunk granularity.
- Clone to a new directory does not break sync when `path_key` is stable.

### Negative

- Two Merkle layers to maintain and test.
- Chunker version bumps are expensive; must be explicit in ops runbooks.

### Follow-ups

- P2: simhash over chunk multiset for copy-on-write seed (Cursor-style).
- P2: Merkle proofs for safe index reuse across clones.
