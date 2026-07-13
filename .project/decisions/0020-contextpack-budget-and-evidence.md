# ADR-0020: ContextPack Budget and Evidence Classes

Status: Accepted
Date: 2026-07-11
Related: [0006](0006-trace-event-append-only-replay.md),
[0016](0016-lexicographic-context-contracts.md),
[0019](0019-phase1-retrieval-scoring.md)

## Context

`ContextPack` is the central runtime handoff. Without explicit evidence classes,
trust labels, instruction/data separation, and deterministic trimming, packs
become opaque prompt blobs and cannot be verified or replayed safely.

## Decision

### 1. Evidence classes

Every `EvidenceItem` has exactly one `evidence_class`:

| Class | Meaning | May justify factual claims? |
|-------|---------|-----------------------------|
| `source_text` | Original retrieved source span | Yes, if trust allows |
| `lexical_analysis` | Lemma/wordform/morph metadata | No (annotation only) |
| `sense_claim` | Sense definition or sense link | Only with lexicon authority + trace |
| `concept_mapping` | Concept/thesaurus mapping | Only as mapping, not as witnessed quote |
| `attestation` | Witnessed quote with span | Yes |
| `tool_output` | Typed tool result excerpt | Yes if tool marked authoritative |
| `model_inference` | Model-produced text | Never as source truth |
| `instruction` | Task/system/developer instructions | Not evidence |
| `policy` | Frozen policy snapshot excerpts | Not evidence |

Factual verifier rules in phase 1: accept `source_text`, `attestation`, and
explicitly authoritative `tool_output` only. Flag or reject unsupported
`sense_claim` / `concept_mapping` / `model_inference` when used as facts.

### 2. Trust labels

Each evidence item carries `trust_level` derived from its source (future layer
may refine ACL):

`trusted` | `project` | `external` | `untrusted` | `quarantined`

Pack builder must enforce `FocusProfile.required_trust_level`. Quarantined items
never enter model-facing evidence; they may appear only in rejected lists.

### 3. Instruction / data separation

A `ContextPack` has distinct sections:

```text
instructions[]     # task, system, developer — not mixed into evidence_items
policy_refs[]      # PolicySnapshot ids
evidence_items[]   # data only
rejected_items[]   # debug / replay
verification_requirements[]
```

Retrieved source text must not be written into `instructions`. Pack consumers
must treat `instructions` as control plane and `evidence_items` as data plane.

### 4. Citation locking

When `FocusProfile.citation_strictness` is `strict` (default for factual tasks):

1. Every retained `source_text` / `attestation` item keeps `source_id`, spans,
   checksum, and optional `context_ref`.
2. Trimming must not remove the last citation supporting a locked claim marker.
3. Summaries are allowed only as optional `summary` fields alongside the span,
   never as a replacement that drops the span.

### 5. Budget model

Budget fields (phase 1):

```text
max_items
max_chars          # sum of evidence surface chars
max_tokens_estimate  # deterministic estimator version pinned on pack
reserve_for_instructions
```

Estimator version is recorded on the pack (`budget_estimator_version`). Phase 1
may use a simple `chars/4` token estimate; changing the estimator bumps the
version and invalidates golden budgets.

### 6. Deterministic trimming

Algorithm:

1. Partition candidates into `required` (citation-locked / FocusProfile must-include)
   and `optional` (ranked by ADR-0019 merged score).
2. Always attempt to include all `required` items; if they exceed budget, fail pack
   build with `budget_exhausted_required` (do not silently drop citations).
3. Fill remaining budget with `optional` in merge order.
4. Prefer complete spans over truncated spans; if a span does not fit, skip it
   rather than cutting mid-span unless `allow_span_truncate` is set (default off).
5. Deduplicate by ADR-0019 dedup key before packing.
6. Write skipped optional items with scores above `reject_score_floor` (default
   0.3) into `rejected_items` with reason `budget_trim` or `duplicate`.

### 7. Pack checksum and replay

```text
pack_checksum = SHA256(
  "context/context-pack/v1" || 0x00 ||
  canonical_json(pack_without_checksum_and_created_at)
)
```

Canonical JSON: UTF-8, sorted object keys, no insignificant whitespace, arrays
in stored order. Replay loads evidence by ids/spans/checksums from stores; it
must not trust inline text if checksum mismatches.

## Consequences

### Positive

- Verifiers can enforce source-backed claims without parsing freeform prompts.
- Lab/UI can render evidence vs instructions separately.

### Negative

- Strict citation mode can fail builds on tiny budgets; that is intentional.
- Token estimator is crude until a real tokenizer adapter is versioned.

### Follow-ups

- Prompt-injection labeling hooks (future-layer) reuse `trust_level` and
  instruction/data separation.
- Richer token estimators via language adapters without changing pack schema.
