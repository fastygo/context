# ADR-0028: Source Tombstones For Soft Delete

Status: Accepted  
Date: 2026-07-14  
Related: [0021](0021-snapshot-commit-failure-semantics.md),
[0023](0023-derived-artifact-lineage-temporal-source-metadata.md),
[0026](0026-public-api-v1-freeze.md),
stabilization gap **C1**

## Context

Deleted or withdrawn sources must stop appearing in search and new
`ContextPack`s without rewriting historical snapshots or destroying provenance.
Hard-delete would break lineage and repair archaeology. Lab Gate had repair but
no soft-delete path.

## Decision

1. `Source.TombstonedAt` is optional. Nil means live; non-nil means soft-deleted.
2. `MetadataStore.TombstoneSource` is idempotent (keeps the first timestamp).
3. `PutSource` with `TombstonedAt == nil` revives a source (re-ingest path).
4. Local workspace persists `tombstoned_source_ids` in `state.json`; search and
   pack loaders mark matching chunks `Tombstoned` and exclude them from
   `index.Memory.List` / `MatchesFilters`.
5. Additive API: `POST /v1/sources/tombstone` and CLI `tombstone-source`.
6. Chunk rows and artifact bytes are retained until a later retention/GC gate
   (stabilization C7 / future-layer L13).

## Consequences

### Positive

- Consumers can withdraw evidence without core archaeology.
- Provenance and lineage remain inspectable for tombstoned sources.

### Negative

- Sparse/dense backends may still hold vectors until rebuild/GC; client-side
  filters hide them from packs. Ops should rebuild after mass tombstones when
  backends claim server-side filters.

### Follow-ups

- Snapshot export/import (C2), project delete/export (C7).
- Explicit index health states beyond `SnapshotStatus` (rest of C1).
