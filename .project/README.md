# `.project` — planning hub

This folder holds **planned / deferred** material only. It is not a second
user guide and not a completion dump.

| Need | Location |
| --- | --- |
| **Что умеет Runtime сейчас (RU, продукт)** | [context-runtime-seichas.ru.md](context-runtime-seichas.ru.md) |
| How to run / integrate shipped core | [`docs/`](../docs/README.md) |
| Lab/BFF freeze (passed) | [`docs/lab-gate.md`](../docs/lab-gate.md) |
| Why a boundary exists | [`docs/decisions/`](../docs/decisions/README.md) |
| Measured PoC / eval JSON | [`.proofs/`](../.proofs/) |
| **What to plan or defer next** | this folder (below) |

## Status snapshot

| Track | Status | Notes |
| --- | --- | --- |
| Phase 0–2 (baseline → MVP path) | **done** | Architecture + PoC/MVP scope in code; see ADRs |
| Phase 3 Lab-ready (Chunks 20–32) | **done** | Gate: [`docs/lab-gate.md`](../docs/lab-gate.md), [ADR-0027](../docs/decisions/0027-lab-gate-freeze.md) |
| Phase 3 leftovers → Stabilization Gate | **S0 accepted; S1 in progress** | [stabilization-roadmap.md](stabilization-roadmap.md) (C3/C2 closed; C1 tombstones done; C7/C8 next) |
| Phase 4–5 | **planned** | Commercial / ecosystem — roadmap + future-layer |
| External adapters (QDrant, Tantivy, …) | **backlog** | Only on measured blocker + ADR; language/lexicon first via stabilization S3 |

Do **not** re-add a chunk-by-chunk progress file here. Archaeology: git history +
ADRs under `docs/decisions/`.

## When to open which file

Keep these **planning documents separate** — they answer different questions.

| File | Open when… | Do not use for… |
| --- | --- | --- |
| [stabilization-roadmap.md](stabilization-roadmap.md) | You need the post–Lab path to “works — don’t touch” for core + adapters | Building product UI; implementing all of future-layer |
| [roadmap-context-core.md](roadmap-context-core.md) | You need the architecture baseline, domain model, package layout, or phased scope (Phase 0–5) | Day-to-day how-to; adapter pick list; “is X deferred?” detail |
| [future-layer.md](future-layer.md) | You are about to add production hardening and must check deferral gates / acceptance | Implementing without a blocker; treating layers as the current sprint backlog |
| [adapters-backlog.md](adapters-backlog.md) | You need the port → first-live → later-adapter matrix and promotion triggers | Redesigning core ports; plugin sketches |

Also in this folder:

| Path | Role |
| --- | --- |
| [stabilization-roadmap.md](stabilization-roadmap.md) | Post–Lab Gate path to durable core + adapters freeze |
| [plugins/](plugins/) | Downstream plugin sketches (not core packages) |
| [.draft/](.draft/) | Scratch notes — not normative |

## Forward work (start here)

1. **Stabilize core + adapters (default next)?** → [stabilization-roadmap.md](stabilization-roadmap.md)  
   Lab Gate ≠ finished. Close gates S1–S5 before treating the runtime as
   long-lived “don’t touch.” Gaps are framed as required engine capabilities
   (durability, evidence, safety, language/lexicon, document/event ingress).
2. **Measured adapter need?** → [adapters-backlog.md](adapters-backlog.md)  
   Promote QDrant / Turbopuffer / `context-sparse` / provider embedders only after
   limits are recorded and an ADR updates backend order
   ([ADR-0017](../docs/decisions/0017-poc-backend-order.md)).
   Prefer language/lexicon/event adapters from stabilization S3 before new
   vector backends.
3. **Production gate / deferred layer?** → [future-layer.md](future-layer.md)  
   Define the contract early; implement only if a proof or Lab path is blocked.
4. **Phase scope / invariants?** → [roadmap-context-core.md](roadmap-context-core.md)  
   Use Phase 3 leftovers and Phase 4–5 as orientation; do not treat the whole
   baseline as unfinished work.
5. **New durable boundary?** → write an ADR under [`docs/decisions/`](../docs/decisions/README.md)
   before code lands.

## Reading order for new planning

```text
docs/lab-gate.md          (what is already frozen for Lab)
  → this README           (where to look next)
  → stabilization-roadmap.md  (default path to “don’t touch”)
  → adapters-backlog.md   OR  future-layer.md  (concrete next choice)
  → roadmap-context-core.md  (only if baseline / phase wording is unclear)
  → docs/decisions/       (ADR before a new boundary)
```

## Explicit non-goals for this folder

- No merge of roadmap + future-layer + adapters into one mega-doc.
- No resurrected `progress.md` completion dump.
- No product/brand identity in core planning (plugins stay sketches).
- No “implement all future layers” without a blocker, owner, budget, and ADR.
