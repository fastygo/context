# Agent notes

`github.com/fastygo/context` is a brand-neutral Go context operating core:
deterministic project memory, precise indexing, hybrid retrieval, source-backed
`ContextPack`s, typed tools and runs, verification, and replayable traces.

It is not a chat application, generic RAG framework, agent demo, product
companion, or wrapper around provider SDKs.

## Required direction

Every change must preserve:

```text
deterministic project memory
  -> precise indexing
  -> hybrid retrieval
  -> source-backed ContextPack
  -> typed tool/model/subagent step
  -> verification
  -> replayable trace
```

`ContextPack` is the central handoff. Every model, tool, verifier, or subagent
step must remain traceable to the task, PolicySnapshot, RetrievalPlan, accepted
and rejected evidence, source spans, checksums, permissions, and evaluation
signals.

## Current status and authority

- Lab Gate and Stabilization S0-S5 are passed.
- HTTP API v1 is frozen for additive evolution only.
- Default post-S5 stance: do not reopen the core without a measured blocker,
  superseding ADR, and tests.

Read before planning or editing:

1. `docs/README.md` - shipped documentation map.
2. `docs/lab-gate.md` - frozen integration contract.
3. `docs/api/v1.md` and `docs/api/v1-changelog.md` - public HTTP/JSON contract.
4. `docs/decisions/README.md` - durable boundaries.
5. `.project/README.md` - planning and deferral map.
6. `.project/stabilization-roadmap.md` - post-Lab freeze.
7. `.project/future-layer.md` and `.project/adapters-backlog.md` - deferred
   capabilities and promotion triggers.
8. `.project/roadmap-context-core.md` - architecture baseline when needed.

Authority by location:

- `docs/` - shipped behavior and integration how-to.
- `docs/decisions/` - normative architecture decisions.
- `.project/` - planned, deferred, and plugin material only.
- `.proofs/` - measured proof and evaluation artifacts.

Do not describe planned plugin behavior as shipped core behavior.

## Public integration boundary

Consumers integrate through:

- HTTP API v1 (`cmd/context-serve`);
- `pkg/contextkit`;
- `pkg/langcontract` and `pkg/langtestkit` for external language adapters;
- documented thin public language packages.

Never import `internal/` from downstream products or sibling repositories.
Prefer `internal/` for new core implementation until a public interface has
proven stable.

Public API rules:

- require explicit `project_id`;
- expose `path_key`, never host filesystem paths;
- preserve v1 fields and semantics; additive changes only;
- document additive changes in `docs/api/v1-changelog.md`;
- use an ADR before changing a durable boundary.

## Non-negotiable invariants

1. **Project isolation**: indexes, artifacts, packs, runs, jobs, and schedules
   belong to an explicit project.
2. **Evidence before generation**: models receive selected evidence, not
   unbounded history.
3. **Provenance is mandatory**: preserve source ids, spans, checksums, versions,
   attestations, tool outputs, and derivation lineage.
4. **Original text is preserved**: normalization, morphology, and concept
   mapping never replace source bytes.
5. **Instruction is not evidence**: instructions and policy never enter
   `evidence_items`.
6. **Model output is not source truth**: generated inference may summarize or
   propose, but cannot independently justify factual claims.
7. **Permissions are external to the model**: allow/ask/deny, risk, approval,
   and side-effect gates run in deterministic policy code.
8. **Conflicts remain visible**: do not silently collapse contradictory
   authoritative evidence.
9. **Providers are replaceable adapters**: domain logic must not depend on one
   model, vector store, database, crawler, or tool provider.
10. **Runs are observable and replayable**: background work needs task, owner,
    budget, trace, cancellation, and explicit failure semantics.

## Dependency and package discipline

The module targets Go 1.25 and intentionally keeps dependencies narrow.
Prefer the standard library and internal ports. PostgreSQL integration uses
`pgx` behind adapters; language support uses `golang.org/x/text`.

- Do not add a dependency without a measured need and adapter boundary.
- Domain packages must not import HTTP clients, provider SDKs, PostgreSQL,
  pgvector, object stores, or host filesystem concerns.
- Keep model, embedding, reranking, storage, vector, language, and tool
  implementations behind narrow interfaces.
- Avoid premature abstractions; require two real uses before generalizing.

## Jev and typed-decision consumers

Jev is currently available through the direct TypeSafe API and OpenRouter, and
additional providers may appear. Context core must not depend on any of them.

LeX or another downstream consumer may:

```text
Context API -> frozen ContextPack
  -> external typed-decision adapter
  -> deterministic consumer verifier
  -> store DecisionSet / VerificationReport as structured artifacts with lineage
```

Rules:

- no TypeSafe, OpenRouter, Jev, or provider SDK imports in core domain packages;
- no provider names in core public types, package names, or schemas;
- do not force typed-decision semantics into a prose `Completer` interface if
  the contract does not fit;
- provider endpoints, credentials, retries, aliases, and billing belong in the
  consumer or sibling adapter;
- Context may preserve provider-neutral structured artifacts, lineage, and
  trace metadata;
- a model decision remains `model_inference` unless independently verified.

The direct and hosted Jev APIs are downstream transport choices, not reasons to
reopen frozen Context API v1.

## SADT / ICOM methodology boundary

`.project/plugins/sadt-contract-agency.md` is a deferred consumer methodology
plugin, not a core package plan.

Its useful mapping is:

- Input -> Sources, Artifacts, Chunks, user task.
- Control -> PolicySnapshot, FocusProfile, permissions, quotas, Done-iff.
- Mechanism -> model adapter, typed tools, AgentRun.
- Output -> Decision, Artifact, verified result.

Apply the soft-contract principle:

```text
contract = frame + Done-iff + hard boundaries
inside   = mechanism freedom
outside  = invalid action
```

Do not add `sadt`, `idef0`, or methodology-specific names to core identifiers.
Do not encode rigid tool-order scripts as policy. Promote a neutral contract
artifact into core only after two unrelated consumers prove the need and an ADR
accepts the boundary.

## Engineering rules

- Use explicit domain language: `Project`, `Source`, `Artifact`, `Chunk`,
  `FocusProfile`, `ContextPack`, `AgentRun`, `ToolCall`, `Evaluation`.
- Keep classic deterministic IR first-class: exact, sparse, phrase/span,
  morphology, filters, and source lookup.
- Preserve every retriever contribution and explanation through merge/dedup.
- Treat generated wordforms as expansion candidates, not attestations.
- Keep lexicon resources separate from language analyzers.
- Keep graph projections consumer-owned.
- Prefer deterministic verification for source-backed claims.
- Add tests around invariants, edge cases, and regressions before or alongside
  implementation.

Terminology:

- use `lexeme`, `lemma`, `wordform`, `morphology`, and `lexicon` precisely;
- do not use trademarked construction-toy terminology in identifiers, package
  names, commands, config keys, or schemas;
- write code comments in English.

## Verification

Default offline suite:

```bash
go test ./... -count=1
```

Relevant focused suites:

```bash
go test ./internal/httpserver/ -run TestLabGateSmoke -count=1
go test ./internal/evals/golden/ ./internal/evals/adversarial/ -count=1
```

When changing language adapters, storage adapters, Postgres paths, or API
contracts, run the documented focused tests in `docs/operations/` and
`docs/lab-gate.md`. Default tests must remain usable without optional
infrastructure.

Before completion:

- run tests proportional to the change;
- check edited files for lint errors;
- update shipped docs for user-visible behavior;
- add an ADR for a durable boundary;
- update progress/status only after verification.

## Do not

Without a measured blocker, ADR, and tests, do not add:

- chat UI, product shell, companion identity, billing, or product workflows;
- non-additive API v1 changes;
- OpenAPI generation or gRPC;
- full Query AST or in-core graph store;
- QDrant, Turbopuffer, or Tantivy as first-class core dependencies;
- object-store ArtifactStore, DOCX, OCR, or fuzzy search in core;
- distributed workers, leases, or DLQ;
- full OIDC, fine-grained ACL, or multi-tenant billing;
- heavy dictionaries or TEI/SKOS resource importers in domain packages;
- provider-specific model, embedding, or typed-decision contracts;
- arbitrary tool execution or side effects without consumer sandbox and policy.

Review every proposal with:

> Does this make the core more precise, auditable, minimal, and replayable, or
> does it move the project toward another opaque RAG framework?
