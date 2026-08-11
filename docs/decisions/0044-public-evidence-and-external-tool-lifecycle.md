# ADR-0044: Public Evidence Write And External Tool Lifecycle

Status: Accepted  
Date: 2026-08-11  
Related: [0022](0022-structured-artifact-schema-id.md),
[0023](0023-derived-artifact-lineage-temporal-source-metadata.md),
[0026](0026-public-api-v1-freeze.md),
[0034](0034-tool-side-effect-approval.md),
[0042](0042-stabilization-gate.md)

## Context

The domain and metadata adapters already persist structured artifact metadata,
`schema_id`, many-to-many lineage, typed tools, policy decisions, tool calls,
and trace events. Those contracts were not available through HTTP v1 or
`pkg/contextkit`. A downstream orchestrator therefore had to import
`internal/`, fork the runtime, or keep evidence and tool audit data elsewhere.

This is a measured integration blocker after Stabilization S5. It can be
closed without adding product schemas, arbitrary command execution, crawler
governance, or a new agent framework.

## Decision

1. Add artifact routes under `/v1/artifacts` and matching `contextkit` methods.
   A caller may store bytes (`body_base64`) or JSON (`json`) with generic
   `artifact_type`, optional `schema_id`, checksum assertion, trust,
   evidence class, and optional `ArtifactLineage`.
2. Artifact identifiers are immutable. Repeating an identical put is
   idempotent; different bytes or metadata under an existing id return
   conflict. Normalizers and language adapters never replace the stored bytes.
3. Text, JSON, and XML artifacts registered by this route participate in the
   local project-memory retrieval projection. Search and ContextPack citations
   carry a synthetic artifact source reference, byte span, and the stored
   checksum. Instruction/policy evidence classes remain rejected from
   `evidence_items`, and FocusProfile trust filtering still applies.
4. Add `/v1/tools` for neutral external tool descriptors and
   `/v1/tool-calls` for orchestrator-reported lifecycle transitions:
   `requested`, `approved`, `denied`, `executed`, `verified`.
5. Descriptors contain name, input/output JSON schemas and versions,
   permission, risk, side-effect class, timeout, and `needs_approval`.
   Product-specific names are opaque data; core defines no product tool enum.
6. A requested tool is evaluated outside the model. Explicit descriptor
   permission may allow or deny. Otherwise write/external side effects decide
   `ask`; `needs_approval` always decides `ask`. An `executed` transition is
   rejected until the call is approved and must include a checksummed result
   artifact. Approval/denial records require the reporting actor, and verified
   records require a verification description.
7. Context records policy/lifecycle trace events and result artifact identity.
   The consumer orchestrator performs the actual external execution and
   reports verification. Context does not launch arbitrary CLIs or provide a
   sandbox in this slice.
8. The existing `/v1/agent-run` request and response are unchanged. All routes,
   fields, contextkit types, and focus fields in this change are additive under
   ADR-0026.

## Consequences

### Positive

- Consumers can write schema-bound evidence and lineage without importing
  `internal/`.
- Tool approval, result checksums, and trace inspection work with an external
  orchestrator while execution stays behind the consumer's sandbox boundary.
- The same contracts serve document QA, corpus curation, reporting, code
  grounding, and other evidence workflows without core domain changes.

### Negative

- The local artifact retrieval projection is intentionally narrow: textual
  media only, whole-artifact spans, and existing retrieval algorithms. Rich
  parser/chunker re-indexing still uses normal ingest.
- External runs registered by the tool lifecycle route remain running because
  this contract does not add a general external AgentRun orchestration API.
- Descriptor persistence in the local workspace is single-node; distributed
  provider discovery remains deferred.

## Deferred To Consumers

- Sandboxed process/CLI execution, network allowlists, secrets, retries, and
  compensation.
- Product schemas, tool implementations, connectors, crawlers, UI approval
  prompts, and clarification flows.
- Noisy-input correction engines. They return candidate artifacts with spans,
  confidence, and adapter versions; raw bytes remain unchanged.
