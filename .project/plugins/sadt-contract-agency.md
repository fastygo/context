# SADT Contract-Agency Plugin Note

Status: deferred methodology plugin  
Scope: optional pack that teaches agents and exporters to reason with
**reasonable ICOM contracts** — fixed boundaries and Done-iff criteria, free
choice of path inside those boundaries — using Context Runtime primitives.

SADT (Structured Analysis and Design Technique; later IDEF0-style activity
diagrams) is treated here as a **consumer methodology**, not a core brand.
Core package names stay neutral (`FocusProfile`, `ContextPack`, `PolicySnapshot`,
`ToolCall`, `Evaluation`).

## Purpose

Large prompts and persona lines (“you are a senior engineer”) invent an ad-hoc
process model every time. SADT already separates:

| Arrow | Meaning |
| --- | --- |
| **Input** | What is transformed |
| **Control** | Rules and constraints that make the result lawful |
| **Output** | Required result / acceptance |
| **Mechanism** | How the work is performed (model, tools, skills) |

An **Action** is correct only when its **contract** is satisfied. Any contract
violation makes the Action invalid — not “almost done.”

This plugin maps that discipline onto Context Runtime so products (for example
coding gateways and training corpora) can:

1. express tasks as compact ICOM contracts instead of role costumes;
2. assemble evidence and tool policy as Control/Input, not as chat noise;
3. leave the model free to choose Mechanism steps inside the contract;
4. verify Done-iff and record a replayable trace.

## Soft Contract Principle (normative for this plugin)

The contract must be **reasonable, not rigid**:

```text
contract  = frame + Done iff + hard boundaries
inside    = model freedom (tool order, style, decomposition depth)
outside   = Action invalid
```

| Free inside the contract | Hard (violation ⇒ invalid Action) |
| --- | --- |
| Tool order and how many reads | Tool outside allowlist |
| Wording, refactoring style | Invented facts when citation is strict |
| Optional PLAN depth / thinking effort | Secrets in outputs |
| Alternative correct implementations | Declaring done when Done-iff fails |
| Honest GAP when evidence is missing | Silent fabrication instead of GAP |

Control limits **lawfulness** of the transformation; it does not micromanage
Mechanism. That preserves model strengths while still producing the required
Output.

## Mapping To Context Core

| SADT idea | Context Runtime concept |
| --- | --- |
| System / A-0 Action | `Project` + top-level task intent |
| Child Actions (decomposition) | Nested runs, FocusProfiles, intermediate `Artifact`s |
| Control | `PolicySnapshot`, FocusProfile, tool permission/risk, quotas |
| Input | Sources / Artifacts / Chunks + user task |
| Output | Decision / Artifact / verified result |
| Mechanism | Completer adapter, typed tools, `AgentRun` |
| Done iff | `Evaluation` / verification requirements |
| Contract violation | Failed verify, policy deny, or rejected Action label |
| Compact parent + fixed intermediates | Budgeted `ContextPack` + persisted pack/artifact ids |
| Instruction ≠ evidence | Pack sections: `instructions[]` vs `evidence_items[]` (ADR-0020) |

Pipeline alignment:

```text
TaskIntent
  -> PolicySnapshot          # Control
  -> FocusProfile            # Control (scoped, soft where possible)
  -> RetrievalPlan
  -> ContextPack             # Input handoff (budgeted, source-backed)
  -> ModelCall | ToolCall    # Mechanism (path free inside policy)
  -> Verification            # Done iff
  -> Decision | Artifact
  -> EvaluationTrace         # replay / training export
```

## Possible Plugin Shape

A future plugin may provide **configuration and schemas only** (no core fork):

1. **FocusProfile presets** — e.g. `coding.edit`, `coding.debug`, `docs.cite`
   with soft budgets, citation strictness, and allowlisted tool sets.
2. **ICOM contract templates** — small structured docs or `schema_id` artifacts:
   `action`, `input_refs`, `control_refs`, `output_expected`, `done_iff`,
   `mechanism_hints` (hints, not scripts).
3. **Reasoning scaffold (optional)** for agents / SFT export — versioned
   `sadt_reason_v1` text shape:

   ```text
   CONTRACT
   I: …
   C: …
   O_expected: …
   M: …                 # available means, not ordered script
   GAP: …               # missing evidence; do not invent
   DONE_IFF: …

   CHECK
   O_actual: …
   contract: ok | violated
   ```

   PLAN remains optional so training does not reward a single tool trace.
4. **ContextPack templates** — instruction block = Control; evidence = Input;
   never merge retrieved source into system persona.
5. **Verification rubrics** — Done-iff checks: citations present when required,
   tools in policy, no secret leakage, required artifact produced.
6. **Export hooks** — map runs + packs + CHECK labels into consumer trajectory
   formats (OpenAI-compatible messages, etc.) without storing vendor token ids
   in core.
7. **Eval fixtures** — golden trajectories where multiple Mechanisms succeed
   under one contract; negatives where Output appears but Control was broken.

## Consumer Pattern (downstream, not core)

```text
Product Gateway / IDE agent
  -> plugin Focus + ICOM template
  -> contextkit: pack / policy / tools
  -> coding runtime (Mechanism; path free)
  -> verify + trace
  -> optional training blob (contract_ok, pack_id, effort)
```

Context Runtime stays brand-neutral. Product names, companion personas, and
vendor SDK labels belong in the consumer or this plugin’s config — never in
`internal/` domain types.

## Non-Goals

- Do not add `sadt`, IDEF0, or methodology terms to core package or type names.
- Do not make SADT the only supported methodology (see also
  [grace-vivanov.md](grace-vivanov.md) as a separate contract-first track).
- Do not encode rigid step scripts as Policy; that kills soft-contract freedom.
- Do not put retrieval or Context DB inside untrusted preview/sandbox runtimes.
- Do not treat LLM prose as source truth; evidence remains span-backed.
- Do not reopen core for this plugin without the promotion rule in
  [README.md](README.md).

## Research / Design Tasks Before Code

1. Freeze a minimal ICOM artifact schema (`schema_id`) vs FocusProfile-only
   presets — prefer the smaller surface that two consumers can share.
2. Define which Done-iff checks are deterministic (policy, allowlist, pack
   budget) vs evaluative (rubric / human / arena).
3. Specify soft vs hard Focus fields (e.g. `citation_strictness=strict` is hard;
   `suggested_tools` is soft).
4. Align reasoning scaffold with training exporters so SFT rewards contract
   satisfaction, not a single PLAN.
5. Write an ADR only if a **neutral** invariant must enter core (e.g. a shared
   `Contract` artifact type used by unrelated consumers). Otherwise keep the
   schema in the plugin.

## Acceptance Gate

This plugin becomes actionable when:

- Lab/Stabilization gates remain the integration baseline (HTTP v1 /
  `contextkit`);
- at least one downstream consumer needs Focus presets + verify labels for
  agent turns or trajectory export;
- soft-contract principle is written into plugin tests (multiple Mechanisms,
  one Done-iff; hard boundary violations fail);
- no methodology terminology leaks into core APIs.

## Relationship To Other Plugins

| Plugin | Overlap | Separation |
| --- | --- | --- |
| [grace-vivanov.md](grace-vivanov.md) | Contract-first engineering | GRACE/PCAM research track; this pack is ICOM / soft-agency oriented |
| [observation-event-adapters.md](observation-event-adapters.md) | Traceable inputs | Event corpora as Input sources, not methodology |
| Language / lexicon plugins | Evidence quality | Linguistic contracts stay evidence; not Control scripts |
