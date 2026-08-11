# Scenario: reusable evidence orchestrator

This scenario uses only HTTP v1 or `pkg/contextkit`. It shows two unrelated
consumers sharing the same Context contracts without importing `internal/`.

## Consumer A: documentation QA

1. Ingest reference documents as normal sources.
2. Create step-specific FocusProfiles such as `facts`, `external-voice`, and
   `rules`. These ids are consumer configuration, not core enums.
3. Build a pack for each step. Keep rules in `ContextPack.instructions`; do not
   disguise them as retrieved evidence.
4. Store a generated answer or evaluation as
   `artifact_type=structured`, with a consumer-owned `schema_id` and lineage to
   the source spans, ContextPack, run, and tool call that produced it.
5. Search the artifact on later steps; citations retain artifact checksum and
   byte spans.

## Consumer B: corpus curation

1. Store each raw utterance or transcript as immutable `body_base64`. This is
   the source of truth even when it contains ASR, keyboard, spelling, or
   segmentation noise.
2. Run an external language adapter. Store correction candidates as a separate
   structured artifact with candidate text, confidence, source spans, and
   adapter id/version. Attach lineage to the raw artifact. Never replace it.
3. Use focuses such as `facts`, `category-references`, `rules`, or
   `model-inference`. The runtime only sees strings plus generic trust,
   evidence-class, and budget policy.
4. Register an external labeler descriptor. Report `requested`, then obtain an
   approval for write/external side effects, execute it in the consumer's
   sandbox, and report `executed` with a result artifact. Finish with
   `verified` after checksum/schema checks.

## Focus and pack discipline

- Use one FocusProfile per consumer step instead of growing a product enum in
  core.
- `required_trust_level` filters evidence before selection.
- `evidence_class=instruction|policy` is rejected from `evidence_items`; pass
  authoritative rules through pack instructions/policy refs.
- `evidence_class=model_inference` identifies hypotheses and cannot justify a
  factual claim by itself.
- Every artifact/result gets a checksum. Derived outputs use lineage; raw bytes
  remain immutable.

## Deliberately outside Context

- Tool process sandbox, network/filesystem allowlists, credentials, retries,
  and compensation.
- Consumer schemas and names, source connectors, crawler logic, and UI approval
  prompts.
- Noisy-text correction or heavy language dictionaries. External adapters can
  use `pkg/langcontract` and `pkg/langtestkit` where applicable.

API shapes: [HTTP/contextkit v1](../api/v1.md). Boundary decision:
[ADR-0044](../decisions/0044-public-evidence-and-external-tool-lifecycle.md).

