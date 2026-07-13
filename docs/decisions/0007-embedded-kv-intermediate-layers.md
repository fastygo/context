# ADR-0007: Embedded KV for Intermediate Layers

Status: Accepted  
Date: 2026-06-17  
Related: [0002](0002-metadata-store-progression.md), [0008](0008-hybrid-index-architecture.md), [0011](0011-merkle-manifest-and-snapshot-namespace.md)

## Context

Several subsystems need fast, local, read-heavy storage: morphology dictionaries,
embed cache, indexer checkpoints, policy/spec version pins, and optional Merkle
node cache. Archivarius-style linguistic assets are static blobs versioned by
hash. The team considered bbolt and BadgerDB instead of putting everything in
Postgres or inside search indexes.

## Decision

1. **Use embedded KV for intermediate/runtime layers, not for primary search:**
   | Data | Store | Notes |
   |------|-------|-------|
   | Active manifest pointer (offline) | bbolt | Small, single-writer |
   | Embed cache by `chunk_hash` | Badger | Larger values, write-heavy |
   | Morph/rules version pins | Immutable `.bin` + manifest ref | Optional Badger cache |
   | Merkle node cache (monorepo rescan) | Badger | Optional P1 |
   | Inverted index / postings | **Not KV** | Tantivy sidecar (ADR-0009) |
   | Dense vectors | **Not KV** | QDrant (ADR-0004) |

2. **bbolt** for manifest index and small structured KV (one process, predictable).
3. **Badger** for embed cache and larger blobs when local dedup matters.
4. Morphology dictionaries ship as **immutable versioned binaries** referenced
   from `IndexSnapshot.morph_version`; KV holds "which version is active", not
   the linguistic source of truth for search postings.

## Consequences

### Positive

- Cheap local dedup of embeddings; fast restarts without cloud embed API.
- Clear boundary: KV = cache/metadata acceleration; indexes = query-facing.

### Negative

- Badger/bbolt are single-process; not shared replicas — cloud authority remains
  Postgres + object store.

### Follow-ups

- Embed cache adapter interface; optional S3 backing for cloud warm cache (P1).
