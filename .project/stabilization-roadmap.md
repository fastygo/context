# Context Runtime Stabilization Roadmap

Status: **Stabilization Gate (S5) passed** (2026-07-14) —
[ADR-0042](../docs/decisions/0042-stabilization-gate.md)  
Scope: **core + adapters only** (not Lab UI, not branded apps, not domain UIs)  
Architecture baseline: [roadmap-context-core.md](roadmap-context-core.md)  
Deferral catalog: [future-layer.md](future-layer.md)  
Adapter matrix: [adapters-backlog.md](adapters-backlog.md)  
Product snapshot (RU): [context-runtime-seichas.ru.md](context-runtime-seichas.ru.md)

## Purpose

Lab Gate means: *safe to integrate*. It does **not** mean: *finished, never
return*.

This document defines what must land before Context Runtime can follow the
operational rule:

> works? don’t touch.

“Don’t touch” here means: no known core rewrite required for intended
downstream consumers; API v1 stays additive; adapters evolve behind harnesses;
ops can rebuild, degrade, and recover without archaeology.

## Honest status (2026-07-14, post-S5)

| Track | Reality |
| --- | --- |
| Phase 0–2 (baseline → MVP path) | **done** in code |
| Lab Gate (Chunks 20–32, ADR-0027) | **passed** — HTTP v1 + contextkit frozen for Lab |
| Stabilization S0–S5 | **passed** — [ADR-0042](../docs/decisions/0042-stabilization-gate.md) |
| Phase 4 commercial / Phase 5 ecosystem | **planned after S5** (unless deployment blocks) |
| Language / lexicon | thin `context-lang-en` + JSON lexicon + public langtestkit; rich engines external |
| Graph store / traversal | **forever-defer** — [ADR-0040](../docs/decisions/0040-graph-consumer-projection.md) |
| Binary parsers | HTML + lossy PDF shipped; DOCX/OCR deferred |
| Fuzzy / Query AST | Fuzzy: outside core (ADR-0039). Query operators: **minimal subset shipped post-S5** via documented reopen — [ADR-0043](../docs/decisions/0043-ru-adapter-operator-query-layer.md); general DSL still deferred ([ADR-0041](../docs/decisions/0041-query-ast-defer-fts-filters.md)) |
| Scheduled jobs | durable schedule port + file adapter (C8); distributed workers deferred |
| QDrant / Turbopuffer / Tantivy | **frozen-deferred** on measured blocker |

Shipped runtime loop today:

```text
ingest → hybrid search (+ snippets) → ContextPack → agent-run → trace
```

with quotas, redaction, readiness, repair, inspect, tombstones, snapshot
move, schedules, and index lifecycle explain.

Do not claim: full morphology, in-core graph, boolean Query AST, or
production multi-tenant auth.

## What “complete core + adapters” means

### In scope

- Neutral domain invariants stay stable and tested.
- Live first adapters for every **required** port cover the intended PoC/MVP
  deployment (Postgres metadata, pgvector, Postgres FTS, localfs artifacts,
  HTTP Completer/Embedder).
- External adapter SDKs/harnesses prove `context-lang-*` and lexicon
  `ResourceAdapter` without core forks.
- Phase 3 leftovers that would force API breaks or silent corruption are closed
  or explicitly frozen-as-out.
- Failure, rebuild, and degraded modes are boring and documented.

### Out of scope (forever for this freeze)

- Chat/Lab UI, brand, companion identity, domain workflow UIs
- Broad web crawling marketplace
- Training custom embedding models inside core
- Claiming every future-layer item is implemented

Downstream apps may still grow; the **engine** should stop churning.

## Required capability areas

Gaps below are framed as **engine capabilities**, not as product names.

| Capability area | Needed for |
| --- | --- |
| Durable project corpus | long-lived indexes across restart, move, rebuild |
| Evidence presentation | stable citations / snippets for consumers |
| Safety & policy | untrusted sources + non-read tools |
| Linguistic / lexicon adapters | morphology and sense/attestation beyond FTS stemming |
| Document / event ingress | HTML, PDF, time-windowed sources |
| Graph & query semantics | decide once: ship minimal ports or freeze-defer by ADR |
| Eval gates | prove quality instead of asserting it |

## Gap matrix (core vs adapters)

### A. Already enough for Lab integration (do not reopen)

| Capability | Evidence |
| --- | --- |
| API v1 + contextkit | ADR-0026/0027, `docs/lab-gate.md` |
| Hybrid search modes | exact / sparse / dense / hybrid |
| ContextPack + inspect | pack builder + inspector |
| FocusProfile persistence | CLI + Postgres |
| AgentRun + in-process jobs | foreground + background owner/cancel |
| Soft quotas / redaction / ready | Chunks 28–30 |
| Index repair | Chunk 23 |
| Linguistic/lexicon contracts + harness | Chunk 18 |

### B. Must close before Stabilization Gate (core)

These are the items that, if left open, force returns to the engine.

| ID | Gap | Why “don’t touch” fails without it | Source |
| --- | --- | --- | --- |
| C1 | Index lifecycle states + tombstones | Corpora go stale; results lie | **closed** — tombstones [ADR-0028](../docs/decisions/0028-source-tombstones.md); explain [ADR-0032](../docs/decisions/0032-index-lifecycle-explain.md) |
| C2 | Snapshot export/import + active flip safety | Cannot move projects without rewrite | **closed** — [ADR-0029](../docs/decisions/0029-snapshot-bundle-export-import.md); verify refuses corrupt/partial before activate |
| C3 | Lineage + temporal filters durable in Postgres | ~~ADR debt~~ | **closed** — Postgres `artifact_lineage` + temporal cols; client-side overlap filters; integration + proof `08-events-lineage-temporal` |
| C4 | Snippet / highlight contract (offset-stable) | Consumers need citations, not only chunk blobs | **closed** — [ADR-0033](../docs/decisions/0033-offset-stable-snippets.md); `retrieval.Snippet` + `snippet.Attach` |
| C5 | Threat-model + prompt-injection fixtures | Untrusted sources poison agents | **closed** — [ADR-0035](../docs/decisions/0035-prompt-injection-fixtures.md); `evals/adversarial` |
| C6 | Tool side-effect / approval baseline | Any write/network tool is unsafe otherwise | **closed** — [ADR-0034](../docs/decisions/0034-tool-side-effect-approval.md); write/external → `ask` / `needs_approval` |
| C7 | Retention / project delete-export hooks | Long-lived corpora need a governance boundary | **closed** (minimal) — [ADR-0030](../docs/decisions/0030-project-export-delete.md) |
| C8 | Scheduled + event-triggered job **ports** (adapter may be local cron first) | In-process-only dies with the process | **closed** (port + file adapter) — [ADR-0031](../docs/decisions/0031-durable-schedule-port.md) |
| C9 | Graph **port** + one store adapter (even Postgres edges) | Otherwise consumers fork edge schemas | **forever-defer** — consumer projection; stubs only; [ADR-0040](../docs/decisions/0040-graph-consumer-projection.md) |
| C10 | Query AST subset (phrase / AND-OR-NOT / field filters) **or** explicit forever-defer ADR | Power-search keeps reopening without a decision | **forever-defer** — FTS + `RetrievalFilters`; [ADR-0041](../docs/decisions/0041-query-ast-defer-fts-filters.md) |
| C11 | Reranker wiring behind interface (even no-op/weighted only documented) | Interface exists; path must be intentional | **closed** — [ADR-0036](../docs/decisions/0036-intentional-reranker-path.md); Identity on CLI search |
| C12 | Golden eval suites for morph expansion + sense/attestation + event-window | Without metrics, “stable” is opinion | **closed** (baseline) — `eval-golden-v2` morph/sense-attestation/event-window cases |

### C. Must close before Stabilization Gate (adapters)

| ID | Gap | Port | Notes |
| --- | --- | --- | --- |
| A1 | At least one real `context-lang-*` (prefer `ru` or `en`) passing harness | language | **closed** — `context-lang-en` (`pkg/langtestkit/refen` + `internal/linguistic/en`); [ADR-0037](../docs/decisions/0037-public-langtestkit.md)/[0038](../docs/decisions/0038-s3-thin-adapters.md) |
| A2 | `context-lang-testkit` published / usable from adapter repos | language | **closed** — `pkg/langcontract` + `pkg/langtestkit`; [ADR-0037](../docs/decisions/0037-public-langtestkit.md) |
| A3 | At least one lexicon `ResourceAdapter` (TEI **or** curated JSON) passing harness | lexicon | **closed** — `internal/lexicon/jsonres`; [ADR-0038](../docs/decisions/0038-s3-thin-adapters.md) |
| A4 | HTML parser adapter | parse | **closed** — `parse.HTML`; [ADR-0038](../docs/decisions/0038-s3-thin-adapters.md) |
| A5 | PDF text-extraction adapter (confidence flagged) | parse | **closed** — `parse.PDF` (`LowConfidence`); [ADR-0038](../docs/decisions/0038-s3-thin-adapters.md) |
| A6 | DOCX adapter (optional if PDF ships first) | parse | **freeze-defer** — [ADR-0039](../docs/decisions/0039-s3-adapter-freeze-defer.md) |
| A7 | One observation/event source adapter (message export **or** JSONL log) | source + temporal | **closed** — `source.NDJSONFiles`; [ADR-0038](../docs/decisions/0038-s3-thin-adapters.md) |
| A8 | Object-store `ArtifactStore` **or** ADR “localfs-only until measured” | artifacts | **freeze-defer localfs** — [ADR-0039](../docs/decisions/0039-s3-adapter-freeze-defer.md) |
| A9 | Provider Completer/Embedder beyond generic HTTP **or** document HTTP as the supported production adapter | models | **closed (HTTP documented)** — [ADR-0039](../docs/decisions/0039-s3-adapter-freeze-defer.md) |
| A10 | Fuzzy/trigram **or** ADR defer-with-Postgres-`pg_trgm` recipe outside core | sparse | **freeze-defer pg_trgm outside core** — [ADR-0039](../docs/decisions/0039-s3-adapter-freeze-defer.md) |

### D. Explicitly deferred past Stabilization Gate

Keep interfaces; do not implement unless a measured blocker appears.

- QDrant / Turbopuffer / Tantivy `context-sparse`
- Distributed workers / leases / DLQ (beyond single-node scheduler)
- Multi-tenant OIDC + fine-grained ACL (design exists; auth later)
- OpenAPI codegen / gRPC
- Claim/contradiction graph, simhash COW, cross-user Merkle proofs
- Broad crawler governance
- OCR / spreadsheet / mailbox pipelines
- Full DSL workbench inside core
- Billing / cost accounting

Each deferral must have a one-line freeze note in the Stabilization Gate
checklist so the team does not reopen them casually.

## Capability → gap coverage

| Required capability | Primary gaps |
| --- | --- |
| Durable project corpus across restart / move / rebuild | C1–C3, C7–C8 |
| Time-windowed / event corpora | C3, C8, C12, A7 |
| Stable citations and evidence UX for any consumer | C4, C11, C12 |
| Safe ingest of untrusted text + non-read tools | C5–C6 |
| Morphology beyond FTS stemming | A1–A2, C12 |
| Sense / concept / attestation retrieval path | A3, C12 |
| Binary / HTML document ingest | A4–A6 |
| Private-search power features (boolean / fuzzy) | C10, A10 |
| Citation / co-occurrence / reply edges | C9 |

If Stabilization Gate closes **C\*** and **A1–A7**, downstream consumers can
build on API v1 + adapters without core surgery.

## Sequencing (do this order)

### Gate S0 — Freeze discipline (now)

1. Treat API v1 as additive-only (already ADR-0026).
2. Refuse product/brand names in `internal/`.
3. Promote adapters only with harness + ADR when ports change.
4. Keep `.project/` planning-only; ship how-to in `docs/`.

**Exit:** this document accepted; no parallel “mega progress dump.” ✅ (2026-07-14)

### Gate S1 — Durability of memory (core)

Close: **C1, C2, C7 (minimal), C8 (port + one local adapter)** — **C3 already closed**.

Status: **S1 complete** (2026-07-14).

Goal: a project can live across process restarts, machine moves, and index
rebuilds without silent drift.

**Exit tests:**

- ~~Rebuild/reindex leaves search available or explicitly degraded.~~ ✅ ADR-0032
- ~~Snapshot import refuses corrupt/partial bundles.~~ ✅ ADR-0029
- ~~Lineage + temporal filter round-trip on Postgres.~~ ✅ C3
- ~~Project export/delete tombstones expected rows.~~ ✅ ADR-0030
- ~~Background job survives process model documented (file/cron/queue adapter).~~ ✅ ADR-0031

### Gate S2 — Evidence presentation & safety (core) ✅

Close: **C4, C5, C6, C11, C12 (baseline golden sets)**. ✅

Goal: citations and tools are trustworthy enough that consumers stop patching
core for “show me why.”

**Exit tests:**

- ~~Snippet offsets stable across re-index of unchanged bytes.~~ ✅ ADR-0033
- ~~Adversarial fixture cannot grant tools or override policy.~~ ✅ ADR-0035
- ~~Approval required for non-read tool classes in tests.~~ ✅ ADR-0034
- ~~Golden suite gates CI for exact / sparse / morph-fake / sense-filter /
  event-window cases.~~ ✅ `eval-golden-v2`
- ~~Intentional reranker path (even Identity).~~ ✅ ADR-0036

### Gate S3 — Adapter completeness (external + thin in-repo adapters) ✅

Close: **A1–A7**; decide **A8–A10** via ADR (implement or freeze-defer). ✅

Goal: linguistic, lexicon, binary, and event paths exist as replaceable
adapters, not wishful interfaces.

**Exit tests:**

- ~~`context-lang-*` passes harness.~~ ✅ `langtestkit` + `linguistic/en`
- ~~Lexicon importer passes `lexicon/harness.RunContract`.~~ ✅ `lexicon/jsonres`
- ~~PDF/HTML ingest preserves provenance + extraction confidence.~~ ✅
- ~~One event adapter proves idempotent re-ingest + temporal filter explain.~~ ✅ NDJSON
- ~~No adapter imports product brand into core.~~ ✅
- ~~A6/A8/A9/A10 decided.~~ ✅ [ADR-0039](../docs/decisions/0039-s3-adapter-freeze-defer.md)

### Gate S4 — Graph + search semantics decision (core ports) ✅

Close: **C9** and **C10** (implement minimal **or** write forever-defer ADR
with recommended consumer-side pattern). ✅

Goal: stop re-litigating “do we have a graph?” and “do we have boolean query?”

**Exit (chosen: forever-defer ADRs):**

- ~~ADR: graph remains a consumer projection; core keeps only `NodeRef` /
  `EdgeRef` stubs.~~ ✅ [ADR-0040](../docs/decisions/0040-graph-consumer-projection.md)
- ~~ADR: Postgres FTS + API filters are the supported power-user path.~~ ✅
  [ADR-0041](../docs/decisions/0041-query-ast-defer-fts-filters.md);
  how-to [docs/search-power-user.md](../docs/search-power-user.md)

### Gate S5 — Stabilization Gate (“don’t touch”) ✅

Checklist (all must be true):

- [x] S1–S4 exit criteria green
- [x] `go test ./...` + Lab smoke + golden suites green offline where required
- [x] `docs/lab-gate.md` still accurate; additive API changelog clean
      ([api/v1-changelog.md](../docs/api/v1-changelog.md))
- [x] `docs/` runbooks: ingest, rebuild, restore snapshot, degraded modes
      ([operations/runbook.md](../docs/operations/runbook.md))
- [x] Adapters-backlog updated: every port has first-live **or** explicit defer
- [x] future-layer items not in S1–S4 marked **frozen-deferred** with owner
      (registry in [future-layer.md](future-layer.md); owner: core steward)
- [x] No open “Immediate Next Steps” implying unfinished foundation without ADR
- [x] Downstream consumers integrate via API/adapters only (no `internal/`
      imports) — enforced by `pkg/contextkit` import test + Lab gate docs

**Section D freeze notes** (reopen only with measured blocker + ADR):

| Item | Owner |
| --- | --- |
| QDrant / Turbopuffer / Tantivy | core steward |
| Distributed workers / leases / DLQ | core steward |
| Multi-tenant OIDC + fine-grained ACL | core steward |
| OpenAPI codegen / gRPC | core steward |
| Claim graph, simhash COW, cross-user Merkle | core steward |
| Broad crawler governance | core steward |
| OCR / spreadsheet / mailbox | core steward |
| Full DSL workbench in core | core steward |
| Billing / cost accounting | product (after S5) |

**After S5:** only measured blockers + ADRs reopen core. Default answer to
feature requests that expand domain language: *adapter or downstream consumer.*
([ADR-0042](../docs/decisions/0042-stabilization-gate.md))

## Mapping to existing phase language

| Stabilization gate | Rough roadmap phase |
| --- | --- |
| S0 | Post Lab Gate discipline |
| S1–S2 | Phase 3 leftovers (reliability / evidence / jobs) |
| S3 | Adapter backlog + plugins language/lexicon/event |
| S4 | Selected future-layer ports (graph, query) |
| S5 | Entry to Phase 4 calm — not full commercial checklist |

Phase 4 items (billing, team governance OIDC, COW index reuse) remain
**after** S5 unless a deployment blocks without them.

## Non-goals reminder

Do not use this roadmap to:

- build Lab UI inside `fastygo/context`
- embed dictionaries or morphology engines in core
- implement all of `future-layer.md`
- name core packages after downstream use cases

## Reading order

```text
docs/lab-gate.md
  → this file
  → adapters-backlog.md / plugins/*
  → future-layer.md (only for deferred acceptance gates)
  → roadmap-context-core.md (baseline invariants)
```

## Immediate next actions

1. ~~Accept S0 (this file).~~ ✅
2. ~~S1–S4.~~ ✅
3. ~~S5 Stabilization Gate.~~ ✅ [ADR-0042](../docs/decisions/0042-stabilization-gate.md)
4. **Default:** ship Lab/products against API v1; reopen core only for measured
   blockers + ADR.
5. Phase 4 commercial items (billing, team OIDC, COW) remain **after** S5 unless
   a deployment blocks without them.
