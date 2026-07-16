# Context Runtime

Brand-neutral Go core for project-scoped context management: deterministic
indexing, hybrid retrieval, source-backed `ContextPack`s, typed tools and
agent runs, verification, and replayable traces.

`fastygo/context` is not a chat application, a generic RAG wrapper, or a
product companion. It is a reusable context operating layer for systems that
turn files, documents, logs, tool outputs, rules, lexicons, and external
sources into precise, inspectable, auditable context for automated work.

**Status (2026-07):** Lab Gate and Stabilization Gate are **passed**. Integrate
via HTTP API v1 and `pkg/contextkit`. How-to lives in [`docs/`](docs/README.md);
planning-only material lives in [`.project/`](.project/).

| Gate | State | Normative |
| --- | --- | --- |
| Lab / BFF contract | Passed (2026-07-13) | [docs/lab-gate.md](docs/lab-gate.md), ADR-0027 |
| Stabilization S0–S5 | Passed (2026-07-14) | [ADR-0042](docs/decisions/0042-stabilization-gate.md) |
| Public API | `v1` frozen; additives only | [docs/api/v1.md](docs/api/v1.md), [changelog](docs/api/v1-changelog.md) |
| ADRs | Through ADR-0043 | [docs/decisions/](docs/decisions/README.md) |

Default stance after S5: **do not reopen the core** without a measured blocker,
superseding ADR, and tests. Prefer adapters or downstream consumers.

## Why

Large language models are useful, but they do not solve context management. A
serious agent system needs to know what evidence was selected, where it came
from, why other evidence was rejected, which tool was allowed to run, which
model saw which context, and how the final result can be replayed or debugged.

Plain RAG is not enough for long-lived projects:

- vector search misses exact facts, citations, wordforms, morphology, source
  authority, and operational boundaries;
- unbounded chat history is noisy, lossy, and hard to audit;
- normalized text cannot replace original source text, snippets, quotes, or
  attestations;
- tool calls without typed policy create hidden side effects;
- background agents need ownership, cancellation, traces, and verification;
- source-backed work must preserve spans, versions, checksums, and decisions;
- model, vector, storage, language, lexicon, and tool providers must remain
  replaceable.

## What Ships Today

```text
ingest (corpus → chunks + optional dense/FTS)
  → search (exact | sparse | hybrid | dense* | hybrid-dense | query†)
  → context-pack / inspect
  → agent-run (foreground)  OR  jobs (background AgentRun)
  → schedules (durable tick/fire → jobs)
  → trace / metrics / quota / ready / index-status
  → tombstone / snapshot export-import / project export-delete
```

\* Dense needs Postgres/pgvector (`CONTEXT_ENABLE_DENSE=1`).
† Operator layer with morphology (`"phrase"`, AND/OR/NOT, `~term`, `lang:ru`) —
see [docs/search-operators.md](docs/search-operators.md) (ADR-0043).

Also shipped:

- project-scoped source/artifact memory with spans, checksums, and snapshots;
- Merkle manifests and incremental indexing;
- structured artifacts (`schema_id`), derived-artifact lineage, and neutral
  temporal source metadata (ADR-0022 / ADR-0023);
- language-neutral linguistic + lexicographic contracts;
- in-repo thin language adapters: `en` + rule-based `context-lang-ru`
  (`pkg/lang/ru`); public harness `pkg/langtestkit` / `pkg/langcontract`;
- FocusProfile-scoped retrieval and packing;
- typed tools with permission / risk / optional `needs_approval`;
- redaction on Lab-visible model text and previews;
- soft quotas, readiness/degraded probes, repair paths, golden + adversarial
  eval suites;
- thin HTTP service (`cmd/context-serve`) and Go client (`pkg/contextkit`).

## Documentation

| Doc | Purpose |
| --- | --- |
| [`docs/README.md`](docs/README.md) | Navigation for humans and LLMs |
| [`docs/getting-started.md`](docs/getting-started.md) | First offline loop |
| [`docs/cli.md`](docs/cli.md) | CLI reference |
| [`docs/api/v1.md`](docs/api/v1.md) | HTTP + contextkit |
| [`docs/search-operators.md`](docs/search-operators.md) | `mode=query` operators + morphology |
| [`docs/lab-gate.md`](docs/lab-gate.md) | Lab/BFF + Stabilization contract |
| [`docs/operations/local-server.md`](docs/operations/local-server.md) | Postgres / dense / FTS / env |
| [`docs/operations/runbook.md`](docs/operations/runbook.md) | Ops runbook |
| [`docs/decisions/`](docs/decisions/) | ADRs |
| [`.project/`](.project/) | Planned roadmaps / plugins / drafts only |
| [`.proofs/`](.proofs/) | Proof & eval JSON artifacts |

Product-facing status note (RU):
[`.project/context-runtime-seichas.ru.md`](.project/context-runtime-seichas.ru.md).

### Quick local loop

```bash
./scripts/dev.sh up && ./scripts/dev.sh health   # optional Postgres stack

go run ./cmd/context-dev init-project --root ./tmp/corpus --data ./tmp/data --project demo
go run ./cmd/context-dev ingest --data ./tmp/data --project demo
go run ./cmd/context-dev search --data ./tmp/data --project demo --query 'ZEBRA42' --mode hybrid
go run ./cmd/context-dev search --data ./tmp/data --project demo \
  --mode query --lang ru --query '"точная фраза" AND ~слово'
```

Useful env knobs:

| Variable | Effect |
| --- | --- |
| `CONTEXT_METADATA_KIND=postgres` + `CONTEXT_PG_DSN` | Durable metadata |
| `CONTEXT_SPARSE_KIND=postgres_fts` | Live Postgres FTS |
| `CONTEXT_EMBEDDER_KIND=fake\|local_hash\|http` | Embedder adapter |
| `CONTEXT_COMPLETER_KIND=fake\|localecho\|http` | Completer adapter |
| `CONTEXT_ENABLE_DENSE=1` | Dense upsert/search (pgvector) |
| `CONTEXT_REDACT` | Lab text redaction (default on) |

## Design Principles

- **Project scoped by default**: every index, artifact, run, and decision belongs
  to an explicit project or workspace.
- **Evidence before generation**: model calls receive selected context, not
  unbounded history.
- **Provenance is mandatory**: facts point back to source spans, versions,
  checksums, attestations, tool outputs, or derivation lineage.
- **Original text is preserved**: normalization, lemmatization, query expansion,
  and concept mapping never replace source text.
- **Hybrid retrieval wins**: dense, sparse, exact, morphology, filters, recency,
  and citation signals cooperate; graph UX stays a consumer projection
  (ADR-0040).
- **Multilingual by contract**: core owns stable contracts; language complexity
  lives in adapters (`context-lang-*`, in-repo thin packs or external engines).
- **Lexicons are evidence resources**: dictionaries and TEI/SKOS importers stay
  in resource adapters, not in core domain packages.
- **Models are replaceable**: LLMs, embedders, and rerankers are adapters.
- **Tools are typed**: name, schema, permission policy, risk, structured result.
- **Agents are configurations**: rules, skills, tools, and model preferences are
  data-driven where possible.
- **Background work is explicit**: jobs and schedules are observable,
  cancellable, and auditable (single-node today).
- **Brand neutrality is required**: products configure identity on top; core
  packages stay generic.

## Core Runtime Model

```text
TaskIntent
  -> PolicySnapshot
  -> FocusProfile
  -> RetrievalPlan
  -> RetrieverCalls
  -> CandidateSet
  -> RerankedEvidence
  -> ContextPack
  -> ModelCall | ToolCall | SubagentRun
  -> Verification
  -> Decision | Artifact | Result
  -> EvaluationTrace
```

The central handoff object is `ContextPack`: selected, ranked, budget-aware,
source-backed context for a model, tool, verifier, or subagent. Every step
should remain traceable to a task, policy snapshot, retrieval plan, accepted and
rejected evidence, source spans, checksums, permissions, and evaluation
signals. `ContextPack` and `AgentRun` records must be versioned and replayable.

## Core Concepts

- `Project`: isolation boundary for indexes, packs, runs, and jobs.
- `Source` / `Artifact` / `Chunk`: corpus material with spans and checksums.
- `IndexSnapshot` / `ManifestNode`: immutable index commits + Merkle sync.
- `ArtifactLineage` / `TemporalRange`: derived-output provenance and event-window
  metadata (neutral; not product event schemas).
- `ContextRef` / `PathAlias`: model-visible paths without host filesystem leaks.
- `TokenOccurrence`, `Lexeme`, `Lemma`, `WordForm`, `MorphAnalysis`,
  `QueryExpansion`: language-neutral lexical contracts.
- `Sense`, `Concept`, `Attestation`, `Variant`, `MultiwordExpression`,
  `LexiconSource`: lexicographic evidence contracts.
- `PolicySnapshot` / `FocusProfile` / `RetrievalPlan` / `EvidenceItem`.
- `ContextPack` / `ModelCall` / `AgentRun` / `ToolCall` / `Evaluation`.

## Linguistic And Lexicon Boundaries

```text
fastygo/context
  -> language-neutral contracts + thin in-repo adapters (en, ru)
  -> source spans, snapshots, retrieval, ContextPack, traces
  -> no heavy dictionaries or grammar engines in core domain

context-lang-* / external engines
  -> richer normalization, tokenization, morphology, expansion
  -> validated via pkg/langtestkit + pkg/langcontract
```

Lexicon resource adapters remain separate from analyzers. They map dictionaries,
TEI, SKOS/ISO 25964, and community vocabularies onto `Sense`, `Concept`,
`Attestation`, and `LexiconSource`.

## Indexing And Retrieval

```text
source adapter → artifact store → parser → chunker → tokenizer
  → language adapter → enricher → dual Merkle manifest
  → IndexSnapshot → dense (VectorStore) / sparse (SparseIndexRef) → metadata
```

```text
task → policy → focus → retrieval plan → parallel retrievers
  → merge → dedup → rerank → evidence validation → ContextPack
```

Shipped search modes: `exact`, `sparse`, `hybrid`, `dense`, `hybrid-dense`,
`query`. Lexicographic `TimePeriod` is not event time; use temporal metadata for
log/observation windows. Full Query AST / field DSL stays deferred (ADR-0041);
operator subset is ADR-0043.

## Tool And Agent Runtime

Tools register with name, schemas, permission policy, risk, side-effect class,
timeout, and background support. Foreground `agent-run` and background `jobs`
share one AgentRun path. Durable schedules (`once_at` / `interval` / `event`)
tick into the job registry on a single node; distributed workers remain deferred.

## Package Layout

```text
cmd/
  context-dev/       # local CLI
  context-serve/     # thin HTTP+JSON service (ADR-0024)

internal/            # domain + adapters (not for Lab imports)
  agentruntime/ artifacts/ config/ corpus/ evals/ foundation/
  graph/ httpserver/ indexing/ lexicon/ linguistic/ models/
  ops/ policy/ redaction/ retrieval/ storage/ tools/ tracing/ …

pkg/
  contextkit/        # HTTP client for Lab/BFF
  langcontract/      # public language-adapter contracts
  langtestkit/       # public adapter test harness
  lang/ru/           # in-repo context-lang-ru (rule-based)
```

Prefer `internal` for domain logic. Public surfaces stay small: HTTP /
`contextkit` / lang harnesses. **Do not** import `internal/` from Lab or
products.

## Non-Goals / Frozen-Deferred

Reopen only with measured blocker + ADR (see [future-layer](.project/future-layer.md)
and [adapters-backlog](.project/adapters-backlog.md)):

- Chat UI, Lab shell, or product brand inside this repository
- Multi-tenant OIDC / fine-grained ACL / billing
- OpenAPI codegen / gRPC
- QDrant, Turbopuffer, Tantivy `context-sparse` as first-class live adapters
- In-core graph store or full Query AST (consumer patterns: ADR-0040/0041)
- Object-store ArtifactStore, DOCX, fuzzy/`pg_trgm` in core (ADR-0039)
- OCR / spreadsheet / mailbox / crawler governance
- Distributed worker orchestration / leases / DLQ
- Hard dependency on one model or vector vendor
- Heavy language dictionaries or TEI/SKOS importers in core packages

## Engineering Notes

- Keep raw source storage, embeddings, metadata, indexes, traces, and generated
  artifacts separate.
- Version parsers, chunkers, analyzers, embeddings, and sparse indexes.
- Never replace original source text with normalized text.
- Do not collapse senses into lemmas or concepts into labels.
- Treat generated wordforms as expansion candidates, not attestations, unless
  witnessed in a source.
- Treat long tool outputs as searchable artifacts.
- Prefer deterministic verification for source-backed claims.
- Enforce permission and side-effect decisions outside the model.
- Design every background action with policy, trace, owner, and cancellation.

## Status

| Item | State |
| --- | --- |
| Architecture baseline | Accepted (`.project/roadmap-context-core.md`) |
| Lab Gate | Passed — [docs/lab-gate.md](docs/lab-gate.md) |
| Stabilization Gate S5 | Passed — [ADR-0042](docs/decisions/0042-stabilization-gate.md) |
| Architecture decisions | ADRs under `docs/decisions/` (through ADR-0043) |
| Go implementation | Lab-ready + stabilized core; Phase 4+ planned after S5 |
| Public API | HTTP v1 + `pkg/contextkit` (+ lang harness packages) |
| Dependencies | Prefer stdlib + narrow adapters (`pgx` when Postgres gated) |

How-to and integration: start at [`docs/README.md`](docs/README.md).
Planning after S5: start at [`.project/README.md`](.project/README.md).

**License:** [MIT](LICENSE)
