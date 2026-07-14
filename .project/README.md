# `.project` — planning hub

This folder holds **planned / deferred** material only. It is not a second
user guide and not a completion dump.

| Need | Location |
| --- | --- |
| **Что умеет Runtime сейчас (RU, продукт)** | [context-runtime-seichas.ru.md](context-runtime-seichas.ru.md) |
| How to run / integrate shipped core | [`docs/`](../docs/README.md) |
| Lab/BFF freeze (passed) | [`docs/lab-gate.md`](../docs/lab-gate.md) |
| Stabilization Gate (passed) | [stabilization-roadmap.md](stabilization-roadmap.md), [ADR-0042](../docs/decisions/0042-stabilization-gate.md) |
| Why a boundary exists | [`docs/decisions/`](../docs/decisions/README.md) |
| Measured PoC / eval JSON | [`.proofs/`](../.proofs/) |
| **What to plan or defer next** | this folder (below) |

## Status snapshot

| Track | Status | Notes |
| --- | --- | --- |
| Phase 0–2 (baseline → MVP path) | **done** | Architecture + PoC/MVP scope in code; see ADRs |
| Phase 3 Lab-ready (Chunks 20–32) | **done** | Gate: [`docs/lab-gate.md`](../docs/lab-gate.md), [ADR-0027](../docs/decisions/0027-lab-gate-freeze.md) |
| Stabilization Gate S0–S5 | **passed** (2026-07-14) | [ADR-0042](../docs/decisions/0042-stabilization-gate.md) |
| Phase 4–5 | **planned after S5** | Commercial / ecosystem — reopen core only for measured blockers |
| External adapters (QDrant, Tantivy, …) | **frozen-deferred** | Measured blocker + ADR only |

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

1. **Integrate Lab/product on frozen core?** → [`docs/`](../docs/README.md),
   [lab-gate.md](../docs/lab-gate.md), [api/v1-changelog.md](../docs/api/v1-changelog.md).  
   Stabilization Gate passed — default is “don’t touch” core
   ([ADR-0042](../docs/decisions/0042-stabilization-gate.md)).
2. **Measured adapter / core reopen?** → [adapters-backlog.md](adapters-backlog.md)
   + superseding ADR. No casual reopen of section D / forever-defer items.
3. **Deferred production layer?** → [future-layer.md](future-layer.md)  
   Frozen-deferred registry at top; implement only if blocked.
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
