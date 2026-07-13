# ADR-0023: Derived Artifact Lineage And Temporal Source Metadata

Status: Accepted  
Date: 2026-07-13  
Related: [0003](0003-artifact-store-progression.md),
[0006](0006-trace-event-append-only-replay.md),
[0014](0014-storage-role-separation.md),
[0020](0020-contextpack-budget-and-evidence.md),
[0022](0022-structured-artifact-schema-id.md)

## Context

Derived structured artifacts may combine several artifacts and source spans.
`Artifact.SourceID` can identify one immediate origin, but cannot represent this
many-to-many provenance. Time-oriented corpora also need source/chunk event-time
bounds without importing adapter-owned event schemas into the core.

Runtime `tracing.Event` records engine execution for operational replay. Source
events are corpus bytes and must not be placed in that runtime event log.

## Decision

1. `ArtifactLineage` is immutable metadata keyed by project and output artifact.
   It records input artifact ids, source refs, optional `ContextPack`,
   `AgentRun`, and `ToolCall` ids, generator id/version, transformation kind,
   and creation time. It requires at least one input artifact or source ref;
   duplicate input artifact ids and duplicate source refs are invalid.
2. `Artifact.SourceID` remains an optional immediate origin. It is not a
   derivation list and is not overloaded with many-to-many lineage.
3. `TemporalRange` is a half-open interval `[start, end)` with a time basis.
   Standard bases are `occurred`, `observed`, and `effective`; adapters may use
   an explicit `adapter:*` extension. `TemporalMetadata` adds `ingested_at`.
4. `Source` and `Chunk` may carry `TemporalMetadata`. Individual event payloads,
   subject references, and domain fields remain adapter-owned bytes.
5. Generic retrieval temporal filters use same-basis half-open overlap:
   `source.start < filter.end && filter.start < source.end`. Adjacent ranges do
   not overlap; missing temporal metadata does not match an explicit filter.
   Lexicographic `TimePeriod` remains a separate sense/attestation concept.
6. Event-capable source adapters declare stable event identity, adapter/schema/
   producer versions, occurred and ingested times, trust assignment, ordering,
   idempotent ingest, deterministic batch/window checksums, clock precision, and
   late/out-of-order handling. Duplicate stable ids with identical checksums
   collapse; conflicting payload checksums are rejected.
7. Raw event batches are stored as normal artifact bytes and registered as
   sources. Late events create a deterministic new source/snapshot version or
   are rejected according to the adapter descriptor; they do not rewrite
   historical source bytes.
8. The in-memory metadata adapter stores lineage now so derivation replay can be
   proven before PostgreSQL. Lineage records are immutable and list in
   deterministic output-artifact order.

## Consequences

### Positive

- Derived outputs retain provenance without loading or parsing an agent trace.
- Event-window retrieval has deterministic, backend-neutral overlap semantics.
- Observation, log, message, and telemetry adapters can share compatibility
  tests while retaining their own schemas.

### Negative

- PostgreSQL needs separate lineage and temporal columns/tables in Chunk 11.
- Adapters with point timestamps must choose a documented precision interval.

### Boundaries

- No device, person, room, session, reaction, capability, clinical, or product
  entity is added to the core.
- `tracing.Event` remains an append-only runtime record and is not a source
  event store.
- Streaming, checkpoints, retention, redaction, and production adapter
  lifecycle remain deferred to the event/observation plugin roadmap.
