# Generative UX From Context Core

Status: draft  
Purpose: product goals for Generative UX, what existing Context / UI toolkit
factura already covers, and what (if anything) must still enter the **brand-neutral
core** so agents can form trustworthy UX specs.

Related:

- `../roadmap-context-core.md` (Lab, FocusProfile, ContextPack, tools)
- `../progress.md` (Chunks 02–08 done; 09–12 live stack)
- [cursor-storage-inventory.md](./cursor-storage-inventory.md)
- [lex-lego.md](./lex-lego.md)
- External factura: UI8Kit Codegen (`brick` def → N runtimes + DOM parity)

This draft is brand-neutral on purpose. Downstream names (builders, personal
handles) stay outside `fastygo/context`.

---

## 1. Product goal (one page)

**User need:** refine a real-world task in small steps and see the **interface**
update — not only another chat paragraph.

Example loop:

```text
“carpenters in my city”
  → catalog + ratings + filters
“shipping via carrier N”
  → same catalog, constrained to providers that work with carrier N
next constraint (price / slot / district)
  → screen updates again
```

**System need:** each screen must be produced from a **machine-readable UX spec**
that can be validated, previewed, emitted to a runtime, and replayed — with
**evidence** for any data-bound claim (who appears in the list, which filter
exists).

**Split of ownership:**

| Layer | Owns | Does not own |
|-------|------|--------------|
| **Context Runtime** | Memory, retrieval, ContextPack, tools, AgentRun, verify, trace | Brick DOM IR, multi-runtime emitters, product chrome |
| **UI toolkit / codegen** | Brick definitions, canonical render, emitters, parity | Project memory, hybrid retrieval, agent policy |
| **Product builder / Lab / BFF** | AppBuilder UX, preview host, domain schemas, Generative UI chat shell | Becoming a second context engine |

```text
intent + refinement
    → Context: Focus + retrieve + ContextPack
    → model/tool: draft or patch UX spec (structured)
    → verify: schema + evidence links
    → artifact: versioned UX spec
    → UI toolkit / Templ/BFF: preview or emit
    → next refinement reuses prior spec as source
```

---

## 2. What “UX spec” means here

Not a ContextPack. Not free-form markdown alone.

A UX spec is a **schema-versioned JSON (or IR) document**, for example:

| Level | Example | Producer |
|-------|---------|----------|
| **Screen / page model** | layout regions, bound queries, actions | Builder agent |
| **Composition tree** | nodes referencing brick ids + props | Builder agent |
| **Brick definition** | `brick({ id, parts, props, render })` | UI toolkit authors (usually human/CI) |
| **Variants / tokens** | `*.variants.json` | Design system |

UI8Kit-style brick shape (external factura, illustrative):

```ts
brick({
  id: "ui.card",
  parts: [{
    name: "Card",
    props: [/* Variant, Class, Attrs, Children */],
    render: el("div", [attrClass(), attrRest()], [slot()]),
  }],
})
```

Parity rule from that toolkit: **identity across runtimes is tested**, not
reviewed by eye. Generative UX should target the same discipline: emit into a
**closed IR / schema**, then validate — do not ship one-off HTML as source of truth.

---

## 3. Factura already available

### Context Runtime (in-repo direction)

| Capability | Status (progress) | Generative UX use |
|------------|-------------------|-------------------|
| Source / Artifact / Chunk / Manifest | PoC path | Catalog data, prior specs, rules as corpus |
| Hybrid retrieval + FocusProfile | Designed + partial PoC | “carrier N” style constraints from sources |
| ContextPack + evidence classes (ADR-0020) | PoC | Model sees only selected evidence; `model_inference` ≠ fact |
| Tool registry + structured I/O | PoC | `draft_ux_spec`, `validate_ux_spec`, `query_catalog` |
| AgentRun + trace + verifier | PoC | Replay why a screen was proposed |
| PolicySnapshot | Planned/partial | Allowed bricks, trust, tool gates |
| Lab as UX/DX/DSL consumer | Track in progress.md | Inspect packs/specs; not a core dependency |
| Live Postgres/pgvector stack | Chunks 09–12 pending | Real project corpora for demos |

Roadmap already says LLMs may **draft specs** and plan tool calls, but remain
**downstream of source-backed evidence**.

### UI toolkit / codegen (external)

| Capability | Generative UX use |
|------------|-------------------|
| Single brick source of truth | Agent proposes brick ids + props, not raw multi-stack code |
| Closed expression / node IR | Validation before emit |
| Emitters × runtimes | Same spec → Templ / React / Svelte / Vue / Latte / Twig |
| DOM parity tests | Gate for generated UI quality |
| Presentation-first bricks | Behavior stays in app/BFF/`@aria` layer |

### Platform / builder path (downstream, not this repo)

BFF screen/action model, Templ preview, AppBuilder loop: **consumers** of
Context packs + UX-spec artifacts.

---

## 4. Is the core enough to “form UX specs”?

### Verdict

**Yes for the evidence → decide → structured-output → verify → replay spine.**  
**No if “form UX specs” is misread as owning brick IR, codegen, or the builder UI.**

Context Runtime is **self-sufficient as the brain** of Generative UX.  
It is **not** self-sufficient as the **entire** Generative UX product.

### Covered without new core product types

Using existing primitives:

1. Register domain data + brick catalog docs + prior specs as `Source`/`Artifact`.  
2. `FocusProfile` limits task, sources, tools, budget.  
3. Build `ContextPack` with citations for data-bound UI.  
4. Model proposes a patch; a **typed tool** accepts/returns UX-spec JSON
   (`schema_version`, checksum).  
5. Verifier: JSON Schema / IR validate + require evidence ids for bound queries.  
6. Persist tool output as Artifact; link `AgentRun` + pack id.  
7. Next turn: retrieve previous spec artifact + new user constraint.

That is enough to start Generative UX against a frozen brick catalog.

### Gaps — promote to core only if neutral and reused

| Gap | Need for Generative UX | Recommendation |
|-----|------------------------|----------------|
| Explicit **artifact kind** + **schema_id** | Distinguish UX-spec JSON from logs/blobs | **Done in core (ADR-0022):** `ArtifactType` + `SchemaID`; UI IR stays outside |
| Evidence class for “bound UI claim” | UI node claims data from source X | Prefer existing `tool_output` + `source_text`; optional ADR note — **no new class unless verifier needs it** |
| Spec patch algebra | Refine screen without full regenerate | **Consumer** (builder): JSON patch / RFC6902 over UX-spec; core stores versions |
| Brick / DOM IR | Emit multi-runtime UI | **Stay outside core** (UI toolkit) |
| Screen preview runtime | iframe / Templ preview | **Stay outside core** |
| Catalog query tools | “providers where carrier=N” | **Downstream tools** registered in ToolRegistry; schemas versioned |
| Closed UX-spec schema | Shared between agent and builder | **Consumer contract** published next to builder; core only stores bytes + schema_id |
| Eval: “spec matches user intent + data” | Quality loop | Future eval harness in core **generic** (task outcome), fixtures from builder |

### What not to add to core

- `Brick`, `Variant`, `RenderNode`, DOM emitters  
- Product names, Generative UI chat layout, CMS themes  
- Hard dependency on any one UI toolkit repo  
- Treating model prose as the UX source of truth  

Matches roadmap **Non-Core Responsibilities**: product UI and generated app code
stay in consumers/adapters.

---

## 5. Minimal contract sketch (consumer schema, core-agnostic)

Illustrative only — lives in builder/toolkit, referenced by `schema_id`:

```text
UxSpec
  schema_id: "uxspec.screen.v1"
  checksum: ...
  task_id / agent_run_id / context_pack_id
  nodes[]:
    id
    brick_id          # e.g. ui.card, ui.table
    props             # canonical PascalCase or mapped
    children[]
    data_binding?:
      query_id
      evidence_ref[]  # source_id + span or tool_call_id
  actions[]:
    id, tool_name, input_schema_ref
```

Core obligations when such a document is produced:

- store as Artifact with `schema_id` + checksum  
- record ToolCall input/output versions  
- keep ContextPack id that justified the draft  
- allow re-ingest as Source for later retrieval  

---

## 6. End-to-end loop (target)

```text
1. User refinement (natural language or UI control)
2. FocusProfile update (task lens)
3. Retrieve: domain data + brick catalog + previous UxSpec
4. ContextPack (evidence classes enforced)
5. Tool: draft_or_patch_ux_spec → UxSpec JSON
6. Validate schema + evidence_refs
7. Optional: emit preview via toolkit/BFF (outside core)
8. Persist Artifact + AgentRun events
9. User sees screen; goto 1
```

Failure modes core must make visible:

| Failure | Detection |
|---------|-----------|
| Hallucinated filter/value | Missing evidence_ref / verifier fail |
| Invalid brick_id / props | Schema/IR validation tool |
| Pack budget drop of critical citation | ADR-0020 citation locking |
| Untrusted source drove layout policy | trust_level + FocusProfile |

---

## 7. Roadmap implications (no premature code)

| Action | Where | When |
|--------|-------|------|
| Keep forming UX specs via ToolRegistry + Artifacts | progress Chunks 07–08 already enable | Now (design against PoC) |
| Add generic `schema_id` / structured artifact metadata | ADR-0022 + `internal/artifacts` | **Done** (2026-07-13) |
| Document Lab fixture for ContextPack → fake UxSpec JSON | progress Lab track | After Chunk 08 CLI JSON stable |
| Do **not** fork UI8Kit IR into `fastygo/context` | roadmap boundary | Ever (unless a second consumer proves a neutral IR) |
| Lexical contracts for query language / morphology | ADR-0015/0016 adapters | When catalogs are multilingual |
| Eval harness for “refinement → spec → data truth” | future-layer / eval | After live stack (Chunk 12+) |

**Chunk priority for Generative UX demos:** finish **09–12** (real corpus +
pgvector) so “carrier N” is retrieved from data, not fixtures — then wire a
downstream tool that writes `UxSpec` artifacts.

---

## 8. Short answers

**Is the core already enough?**  
Enough to **drive and audit** UX-spec formation. Not enough to **be** the UI
codegen or builder product.

**What still belongs in core?**  
At most: stronger **structured artifact** metadata (`schema_id`, kind) — **shipped
in ADR-0022** — and verifier hooks that treat schema-valid tool output + evidence
refs as first-class. Everything brick/DOM/preview stays downstream.

**What factura to reuse tomorrow?**  
Context: pack + tools + trace. UI8Kit: brick catalog + closed IR + parity.
Builder/BFF: screen host and refinement UX.

**Success metric for a first spike**  
Same refinement twice → same UxSpec checksum given same corpus snapshot; change
carrier filter → pack evidence changes → spec nodes’ `data_binding` changes →
trace explains it.
