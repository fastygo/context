# ADR-0022: Structured Artifact Schema Identity

Status: Accepted  
Date: 2026-07-13  
Related: [0003](0003-artifact-store-progression.md),
[0014](0014-storage-role-separation.md),
[0020](0020-contextpack-budget-and-evidence.md)

## Context

Downstream builders need machine-readable drafts (screen/UX specs, validated
JSON IR, tool results bound to a published schema). `ContextPack` stays the
evidence handoff into a model/tool step. The **output** of that step is often a
structured document that must be stored, checksummed, re-ingested, and cited on
later turns.

`Artifact` already had `media_type` and a free-form `artifact_type`, but no
stable **schema identity**. Without `schema_id`, consumers cannot tell which
contract a JSON blob claims to satisfy, and verifiers cannot gate
schema-bound drafts separately from opaque spill files.

UI brick/DOM IR and multi-runtime codegen remain **outside** this core
(consumer/toolkit concern). The core only stores bytes + neutral metadata.

## Decision

1. **`Artifact.SchemaID`** is an optional string identifying the consumer
   contract of the payload (example shape: `uxspec.screen.v1`). It is not a
   MIME type and not a file extension.
2. **`Artifact.ArtifactType`** uses a small closed vocabulary for store/routing
   behavior (see code constants). Values:
   - `blob` — opaque or text bytes (default when empty on write)
   - `spill` — long runtime spill
   - `tool_output` — tool result body
   - `structured` — machine-readable document bound to a schema
3. **Invariant:** if `artifact_type` is `structured`, `schema_id` is **required**
   and must be non-empty after trim. If `schema_id` is set on a non-structured
   type, writers should set `artifact_type` to `structured` (adapters may
   normalize on Put).
4. **`ArtifactStore.Put`** accepts optional `PutOptions` for
   `artifact_type`, `schema_id`, and `source_id` without changing the blob
   authority rules in ADR-0003.
5. Tool registry `InputSchemaVer` / `OutputSchemaVer` remain the contract for
   **tool calls**. `Artifact.SchemaID` is the contract for **stored documents**
   (including tool outputs persisted as artifacts when they are schema-bound).

## Consequences

### Positive

- Generative UX and other builders can store versioned specs without forking
  core into UI types.
- Replay and retrieval can filter `artifact_type=structured` + `schema_id`.
- Brand-neutral: no brick/DOM types in `fastygo/context`.

### Negative

- Localfs meta format grows two optional keys; old meta without them still
  loads as `blob`.
- Call sites that emit structured JSON must pass `PutOptions`.

### Follow-ups

- Metadata/Postgres columns for `schema_id` / `artifact_type` in Chunk 11.
- Downstream tools (`draft_*_spec`, validators) register schemas in the
  consumer; core only persists ids.
- Optional: index structured artifacts as Sources with `source_type=spec`.
