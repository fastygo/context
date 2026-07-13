# ADR-0003: Artifact Store Progression

Status: Accepted  
Date: 2026-06-17  
Related: [0014](0014-storage-role-separation.md)

## Context

Source text, tool spill output, terminal captures, and generated intermediates
must remain addressable by checksum and span. Vector and sparse indexes store
coordinates and embeddings, not authoritative plaintext (see ADR-0014).

## Decision

1. **PoC / local:** filesystem artifact store under a project data directory
   (`~/.context/projects/{project_id}/artifacts/` or configurable root).
2. **Cloud / team:** S3-compatible object storage with content-addressed keys
   (`sha256` prefix sharding).
3. Artifact records in metadata store hold: `artifact_id`, `checksum`, `size`,
   `mime`, `storage_uri`, `source_id`, provenance fields, plus `artifact_type`
   and optional `schema_id` for machine-readable documents ([0022](0022-structured-artifact-schema-id.md)).
4. Long tool outputs and terminal streams are **always** artifacts; context
   packs reference slices (offset/limit/grep), not full bodies.

## Consequences

### Positive

- Text authority stays outside QDrant and Tantivy.
- Same read path for local files and cloud objects via `ArtifactStore` interface.

### Negative

- Requires garbage collection policy for orphaned artifacts (deferred to P1).

### Follow-ups

- Spill artifact type in Plan Chunk 03; slice read API in retrieval layer.
