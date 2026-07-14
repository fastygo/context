# Context Runtime Stabilization Roadmap

Status: **S0 accepted** (2026-07-14) — post–Lab Gate freeze path in progress  
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

## Honest status (2026-07-14)

| Track | Reality |
| --- | --- |
| Phase 0–2 (baseline → MVP path) | **done** in code |
| Lab Gate (Chunks 20–32, ADR-0027) | **passed** — HTTP v1 + contextkit frozen for Lab |
| Phase 3 leftovers beyond Lab | **open** |
| Phase 4 commercial / Phase 5 ecosystem | **planned** |
| Language / lexicon live adapters | **contracts + harness only** |
| Graph store / traversal | **stubs only** (`NodeRef` / `EdgeRef`) |
| Binary parsers (PDF/DOCX/…) | **not shipped** (plaintext + markdown only) |
| Fuzzy / trigram / query language | **deferred** |
| Distributed / scheduled job control | **in-process jobs only** |
| QDrant / Turbopuffer / Tantivy | **backlog on measured blocker** |

Shipped runtime loop today:

```text
ingest → hybrid search → ContextPack → agent-run → trace
```

with quotas, redaction, readiness, repair, and inspect.

Do not claim: full morphology, full graph, complete private-search parity, or
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
| C9 | Graph **port** + one store adapter (even Postgres edges) | Otherwise consumers fork edge schemas | graph stub today |
| C10 | Query AST subset (phrase / AND-OR-NOT / field filters) **or** explicit forever-defer ADR | Power-search keeps reopening without a decision | future-layer L04 |
| C11 | Reranker wiring behind interface (even no-op/weighted only documented) | Interface exists; path must be intentional | **closed** — [ADR-0036](../docs/decisions/0036-intentional-reranker-path.md); Identity on CLI search |
| C12 | Golden eval suites for morph expansion + sense/attestation + event-window | Without metrics, “stable” is opinion | **closed** (baseline) — `eval-golden-v2` morph/sense-attestation/event-window cases |

### C. Must close before Stabilization Gate (adapters)

| ID | Gap | Port | Notes |
| --- | --- | --- | --- |
| A1 | At least one real `context-lang-*` (prefer `ru` or `en`) passing harness | language | Postgres FTS is fallback, not the adapter story |
| A2 | `context-lang-testkit` published / usable from adapter repos | language | plugins/language-adapters.md |
| A3 | At least one lexicon `ResourceAdapter` (TEI **or** curated JSON) passing harness | lexicon | proves Sense/Attestation ingress |
| A4 | HTML parser adapter | parse | web captures / HTML corpora |
| A5 | PDF text-extraction adapter (confidence flagged) | parse | binary document corpora |
| A6 | DOCX adapter (optional if PDF ships first) | parse | same |
| A7 | One observation/event source adapter (message export **or** JSONL log) | source + temporal | time-windowed corpora |
| A8 | Object-store `ArtifactStore` **or** ADR “localfs-only until measured” | artifacts | ADR-0003 / 0017 |
| A9 | Provider Completer/Embedder beyond generic HTTP **or** document HTTP as the supported production adapter | models | avoid vendor lock in core |
| A10 | Fuzzy/trigram **or** ADR defer-with-Postgres-`pg_trgm` recipe outside core | sparse | typo-tolerant search |

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

### Gate S3 — Adapter completeness (external + thin in-repo adapters)

Close: **A1–A7**; decide **A8–A10** via ADR (implement or freeze-defer).

Goal: linguistic, lexicon, binary, and event paths exist as replaceable
adapters, not wishful interfaces.

**Exit tests:**

- `context-lang-*` passes `linguistic/harness.RunContract`.
- Lexicon importer passes `lexicon/harness.RunContract`.
- PDF/HTML ingest preserves provenance + extraction confidence.
- One event adapter proves idempotent re-ingest + temporal filter explain.
- No adapter imports product brand into core.

### Gate S4 — Graph + search semantics decision (core ports)

Close: **C9** and **C10** (implement minimal **or** write forever-defer ADR
with recommended consumer-side pattern).

Goal: stop re-litigating “do we have a graph?” and “do we have boolean query?”

**Exit:** either:

- Postgres edge store + bounded traversal used by retrieval planner, **or**
- ADR: graph remains a consumer projection; core keeps only `NodeRef` /
  `EdgeRef` filters; and
- Query AST subset shipped **or** ADR: Postgres FTS + API filters are the
  supported power-user path until measured need.

### Gate S5 — Stabilization Gate (“don’t touch”)

Checklist (all must be true):

- [ ] S1–S4 exit criteria green
- [ ] `go test ./...` + Lab smoke + golden suites green offline where required
- [ ] `docs/lab-gate.md` still accurate; additive API changelog clean
- [ ] `docs/` runbooks: ingest, rebuild, restore snapshot, degraded modes
- [ ] Adapters-backlog updated: every port has first-live **or** explicit defer
- [ ] future-layer items not in S1–S4 marked **frozen-deferred** with owner
- [ ] No open “Immediate Next Steps” in roadmap that imply unfinished foundation
      without a dated follow-up ADR
- [ ] Downstream consumers integrate via API/adapters only (no `internal/` imports)

**After S5:** only measured blockers + ADRs reopen core. Default answer to
feature requests that expand domain language: *adapter or downstream consumer.*

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
2. ~~Confirm C3 (Postgres lineage/temporal).~~ ✅ closed — see gap matrix.
3. ~~C1 tombstones + lifecycle explain.~~ ✅ ADR-0028 + ADR-0032
4. ~~C2 snapshot export/import.~~ ✅ ADR-0029
5. ~~C7 project export/delete.~~ ✅ ADR-0030
6. ~~C8 scheduler port + file adapter.~~ ✅ ADR-0031
7. ~~S1 complete.~~ ✅
8. ~~S2 complete.~~ ✅ (C4–C6, C11, C12) → start **S3** (A1–A7; A8–A10 ADR).
9. Parallel: **A2** (`context-lang-testkit`) — unblocks A1.
10. Write defer ADRs for anything in section D that keeps getting reopened.
