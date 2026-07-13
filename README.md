# Context Runtime

Universal Go core for project-scoped context management, retrieval, indexing,
lexical evidence, and agent orchestration. Context Runtime — project-scoped corpus engine that assembles evidence-backed context packs.

`fastygo/context` is not a chat application, a generic RAG wrapper,
or a product companion. It is a reusable context operating layer for systems
that need to turn user intent, files, documents, logs, tool outputs, rules,
lexicons, and external sources into precise, inspectable, auditable context for
automated work.

The repository is in an early planning-and-implementation stage: architecture
decisions are recorded, and the first proof-of-concept code path is defined in
`.project/progress.md`. Target capabilities below describe the intended design,
not a finished runtime.

## Why

Large language models are useful, but they do not solve context management. A
serious agent system needs to know what evidence was selected, where it came
from, why other evidence was rejected, which tool was allowed to run, which
model saw which context, and how the final result can be replayed or debugged.

This module exists because plain RAG is not enough for long-lived projects:

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

The engine is designed to combine deterministic information retrieval,
source-backed context packs, typed tools, replaceable adapters, and
agent/subagent orchestration without baking product identity, language-specific
grammar, dictionary content, or scenario-specific products into the core.

## What It Targets

The module is designed to provide foundations for:

- project-scoped source and artifact memory;
- deterministic indexing with manifests and incremental updates;
- hybrid retrieval over dense vectors, sparse/keyword indexes, exact matching,
  morphology-aware lexical paths, graph traversal, source filters, recency, and
  tool outputs;
- language-neutral contracts for tokens, lemmas, lexemes, wordforms,
  morphology features, analyses, and query expansion;
- lexicographic contracts for senses, concepts, attestations, variants,
  registers, regions, time periods, and lexicon sources;
- focus policies that keep retrieval and context packing scoped to the current
  task;
- replayable `ContextPack` objects for model calls, tools, and subagents;
- typed tool registration with permissions, risk levels, and structured results;
- foreground and background `AgentRun` traces;
- verification and evaluation loops for retrieval quality and factuality;
- adapter boundaries for LLMs, embeddings, rerankers, vector stores, metadata
  stores, artifact stores, language analyzers, lexicon resources, crawlers, and
  product integrations.

## Current Focus

Nearest work follows `.project/progress.md` plan chunks:

| Phase | Status | Goal |
| --- | --- | --- |
| **0 — Architecture baseline** | Chunk 01 + Foundation Gate done | Lock package, storage, index, trace, linguistic, and scoring boundaries |
| **1 — Proof of concept** | Chunks 02–13 done | CLI loop + pgvector/Postgres + durable metadata opt-in; proof validated |
| **2 — MVP toward service API** | Chunks **14–20** done | Thin HTTP service (ADR-0024) |
| **3 — Reliable Beta** | Chunks **21–29** Lab-ready track done | redaction / background optional |

Immediate next step: **Chunk 28** — quota soft-limits.
Redaction / background scheduling follow after Chunk 29.

Phase 2 MVP service boundary: thin HTTP+JSON (`cmd/context-serve`, ADR-0024).
Phase 3 starts with `pkg/contextkit` (Chunk 21). See `.project/progress.md`.

Local stack: `./scripts/dev.sh up` then `./scripts/dev.sh health`.
Durable CLI: `CONTEXT_METADATA_KIND=postgres` + `CONTEXT_PG_DSN=...`.
Sparse FTS: `CONTEXT_SPARSE_KIND=postgres_fts` + `CONTEXT_PG_DSN=...`.
Embedder: `CONTEXT_EMBEDDER_KIND=fake|local_hash|http` (local-hash-v1 dim 32;
`CONTEXT_EMBEDDER_HTTP_URL` for http).
Completer: `CONTEXT_COMPLETER_KIND=fake|localecho|http`
(`CONTEXT_COMPLETER_HTTP_URL` for http).
Dense on ingest: `CONTEXT_ENABLE_DENSE=1` (search uses committed vectors;
`CONTEXT_DENSE_REBUILD=1` optional).
Dense modes: `--mode dense` / `--mode hybrid-dense`.
Proof artifacts: [`.project/proof/`](.project/proof/).
See [`.project/local-server.md`](.project/local-server.md).

PoC storage progression (from accepted ADRs):

- **Artifacts:** local filesystem first; object storage later.
- **Metadata:** in-memory first; PostgreSQL when the live stack is introduced.
- **Dense vectors:** `VectorStore` port first; PostgreSQL + pgvector as the first
  live backend behind that port.
- **Sparse search:** fake or PostgreSQL full-text baseline first; dedicated sparse
  services such as `context-sparse` only when measurements justify them.
- **Models and language:** fake deterministic providers and simple/no-op linguistic
  fixtures first; real providers and `context-lang-*` adapters after the CLI proof.

Downstream UX shells such as `Lab` may consume CLI JSON and proof artifacts.
They must not become imports or dependencies of the core module.

## Design Principles

- **Project scoped by default**: every index, artifact, run, and decision belongs
  to an explicit project or workspace.
- **Evidence before generation**: model calls receive selected context, not
  unbounded history.
- **Provenance is mandatory**: facts point back to source spans, versions,
  checksums, attestations, or tool outputs.
- **Original text is preserved**: normalization, lemmatization, query expansion,
  and concept mapping never replace source text.
- **Hybrid retrieval wins**: dense vectors, sparse search, exact matching,
  morphology, graph traversal, recency, sense/concept filters, and citation
  signals cooperate.
- **Multilingual by contract**: the core defines stable contracts; language
  complexity lives in `context-lang-*` adapters.
- **Lexicons are evidence resources**: dictionaries, thesauri, historical
  lexicons, slang, and regional vocabularies live in resource adapters, not in
  the core.
- **Models are replaceable**: LLMs, embedding models, and rerankers are adapters,
  not hardcoded infrastructure.
- **Tools are typed**: every tool has a name, schema, permission policy, risk
  level, and structured result.
- **Agents are configurations**: orchestration policy, rules, skills, tools, and
  model preferences are data-driven where possible.
- **Background work is explicit**: scheduled or event-triggered agents are
  observable, cancellable, and auditable.
- **Brand neutrality is required**: downstream products and companions configure
  identity on top of the core; core packages stay generic.

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
source-backed context for a model, tool, verifier, or subagent. Every step should
remain traceable to a task, policy snapshot, retrieval plan, accepted and
rejected evidence, source spans, checksums, permissions, and evaluation
signals. `ContextPack` and `AgentRun` records must be versioned and replayable
so bad retrieval, generation, or tool decisions can be debugged later.

## Core Concepts

- `Project`: an isolated workspace or tenant boundary.
- `Source`: a file, document, URL, log stream, database snapshot, spec, chat
  history, or tool output.
- `Artifact`: stored source material or generated intermediate output.
- `IndexSnapshot`: an immutable index commit with parser, chunker, embedding,
  morphology, and lexicon-resource version fields.
- `ManifestNode`: a Merkle-style manifest node for incremental source-tree sync.
- `Chunk`: an indexed source span with metadata and provenance.
- `ContextRef`, `PathAlias`: stable references for model-visible source paths
  without leaking host filesystem layout.
- `VectorNamespace`, `SparseIndexRef`: project- and snapshot-scoped index handles
  behind replaceable vector and sparse backends.
- `TokenOccurrence`: original token text with stable source offsets.
- `Lexeme`, `Lemma`, `WordForm`: language-neutral references for lexical forms.
- `MorphAnalysis`: one possible morphology analysis; ambiguity stays explicit.
- `QueryExpansion`: explainable lexical or morphology-driven expansion.
- `Sense`: a specific meaning of a lexeme.
- `Concept`: a language-independent or domain concept connected to labels and
  senses.
- `Attestation`: witnessed usage in a source, with quote, span, date, region,
  register, authority, and confidence.
- `Variant`: orthographic, historical, regional, slang, spelling, or script
  variant.
- `MultiwordExpression`: lexical unit spanning multiple tokens or syntactic
  words.
- `LexiconSource`: dictionary, corpus, thesaurus, glossary, authority list, or
  community vocabulary source.
- `PolicySnapshot`: frozen permission, risk, and approval policy for one run.
- `FocusProfile`: the task-specific lens that defines scope, freshness,
  exactness, citation strictness, budgets, allowed tools, and irrelevant areas.
- `RetrievalPlan`: chosen retriever paths, filters, and budgets for a task.
- `EvidenceItem`: one source-backed candidate with rank signals and rejection
  reasons.
- `ContextPack`: selected evidence and instructions for a model/tool/agent step.
- `ModelCall`: one model invocation with inputs, outputs, and provider version.
- `AgentRun`: a foreground, background, scheduled, or event-triggered execution
  trace.
- `ToolCall`: a typed invocation with input, output, status, permissions, and
  side-effect metadata.
- `Evaluation`: a reproducible check for retrieval quality or task correctness.

## Linguistic And Lexicon Boundaries

The core is multilingual by contract, not by embedding every language inside the
repository.

```text
fastygo/context
  -> language-neutral contracts
  -> source spans, snapshots, retrieval, ContextPack, traces
  -> no language-specific dictionaries or grammar rules

context-lang-*
  -> normalization
  -> tokenization
  -> lexeme and wordform analysis
  -> morphology generation
  -> query expansion
  -> language-specific eval fixtures
```

Language adapters may support Russian, English, German, Spanish, French, Hindi,
Indic languages, and future languages. They must preserve source offsets,
analyzer versions, dictionary versions, ambiguity candidates, and expansion
provenance. Core contracts should stay compatible with portable schemes such as
Universal Dependencies and UniMorph while allowing adapter-owned raw metadata.

Lexicon resource adapters are separate from language analyzers. They map
dictionaries, TEI resources, SKOS/ISO 25964 concept schemes, historical
lexicons, regional vocabularies, slang, and community terminology to neutral
contracts such as `Sense`, `Concept`, `Attestation`, and `LexiconSource`.

```text
TokenOccurrence
  -> WordForm
  -> Lemma
  -> Lexeme
  -> Sense
  -> Concept
  -> Attestation
  -> SourceSpan
  -> ContextPackEvidence
```

Lexeme and morphology answer "which form." Sense, concept, and attestation
answer "which meaning, where, when, in which register, and according to which
evidence."

## Indexing Pipeline

The indexing pipeline is source-agnostic and snapshot-oriented:

```text
source adapter
  -> artifact store
  -> parser
  -> chunker
  -> tokenizer
  -> language adapter
  -> enricher
  -> dual Merkle manifest (source tree + chunk set)
  -> IndexSnapshot commit
  -> dense vector index (VectorStore)
  -> sparse/exact index (SparseIndexRef)
  -> graph index
  -> metadata store
```

Incremental sync compares manifests and checksums so unchanged sources and chunks
are not reprocessed. Different source types need different chunking strategies.
Source code, technical documentation, scientific text, legal text, dictionary
entries, usage citations, chat history, logs, web captures, and tool output
should not be split with the same rules.

First PoC scope: local file and artifact adapters, plain text and Markdown
parsing, paragraph- and section-aware chunking, neutral token-span capture, and
simple or no-op language hooks—not production morphology or dictionary imports.

## Retrieval Pipeline

Retrieval is a planning problem, not one vector query. Every retriever call should
carry `project_id`, `snapshot_id`, analyzer or embedding versions, and
explainable match reasons:

```text
task intent
  -> policy snapshot
  -> focus profile
  -> retrieval plan
  -> parallel retriever calls
  -> candidate merge
  -> deduplication
  -> reranking
  -> evidence validation
  -> context pack
```

Supported retrieval paths include:

- dense vector search;
- sparse/BM25-style search;
- exact phrase and source-span search;
- lemma, wordform, and morphology-expanded search;
- sense, concept, attestation, register, region, and time-period filters;
- entity and metadata filters;
- citation lookup;
- graph traversal;
- recent activity retrieval;
- tool result retrieval;
- external source retrieval through explicit adapters.

Every retrieval contribution should be explainable. A candidate that matched
through a generated wordform, fuzzy variant, concept label, or attestation must
preserve the original surface text and source span.

## Tool And Agent Runtime

Tools are registered through typed metadata:

```text
tool
  name
  description
  input schema
  output schema
  permission policy
  risk level
  side-effect class
  timeout
  background support
```

Agents coordinate retrieval, context packs, model calls, tools, verifiers, and
subagents. Subagents run with isolated context and return structured summaries
or artifacts to the parent run. Concrete products should integrate through
adapters, tools, graph projections, rules, skills, contracts, or companion
configuration.

## Architecture Guidance

The canonical planning documents live in `.project/`:

- `.project/roadmap-context-core.md`: architectural baseline and phased roadmap.
- `.project/progress.md`: copy-paste plan chunks from baseline to PoC.
- `.project/decisions/`: accepted ADRs for package, storage, index, trace, and
  adapter boundaries.
- `.project/future-layer.md`: deferred production-grade layers and review gates.
- `.project/plugins/language-adapters.md`: roadmap for `context-lang-*`
  language adapters.
- `.project/plugins/lexicon-resources.md`: roadmap for dictionaries, thesauri,
  attestations, historical lexicons, and controlled vocabulary resources.

The project skill lives in `.cursor/skills/context-core-steward/` and should be
used when planning, implementing, reviewing, or debugging this repository. It
keeps work aligned with DDD, Clean Architecture, SOLID, DRY, TDD, traceability,
brand-neutral API boundaries, and the active plan chunk in `progress.md`.

## Planned Package Direction

The public API should stay small until boundaries are stable. Layout below is
the target skeleton for Chunk 02 onward, not an existing tree.

```text
cmd/
  context-dev/              # local developer CLI for indexing, search, evals
  context-serve/            # thin HTTP+JSON service (ADR-0024)

internal/
  agentruntime/             # agent runs, orchestration, subagents, scheduling
  artifacts/                # artifact metadata and stores
  config/                   # project config, rules, ignore patterns
  corpus/                   # projects, sources, chunks, provenance
  evals/                    # retrieval and task evaluation harnesses
  ops/                      # metrics snapshot + append-only eval history
  graph/                    # entity, citation, co-occurrence, dependency edges
  indexing/                 # parsing, chunking, enrichment, manifests
  lexicon/                  # sense, concept, attestation, resource contracts
  linguistic/               # language-neutral contracts and simple adapters
  models/                   # LLM, embedding, reranker interfaces
  policy/                   # permissions, risks, approvals
  retrieval/                # planners, retrievers, rerankers, context packs
                            # and focus profiles
  storage/                  # metadata store abstractions and adapters
  tools/                    # registry, schemas, execution
  tracing/                  # append-only runtime events and redaction

pkg/
  contextkit/               # thin HTTP client over ADR-0024 (Chunk 21)
```

Prefer `internal` for domain/adapters. `pkg/contextkit` is the first public
consumer surface (HTTP client only — not a dump of domain ports).

## First Proof Target

The hypothesis-validation path is split into two stages in `progress.md`:

**Stage A — no external services (Chunks 02–08)**

```text
domain models and store ports
  -> local artifact store + in-memory metadata
  -> deterministic indexing with Merkle manifest and IndexSnapshot
  -> exact/sparse/vector retrieval through ports and fakes
  -> context pack builder and verifier
  -> fake model/tool agent run
  -> context-dev CLI with machine-readable JSON
  -> context-serve HTTP+JSON (same DTOs; Lab/BFF without internal/)
```

**Stage B — live local stack (Chunks 09–12)**

```text
local project corpus
  -> deterministic indexing
  -> PostgreSQL metadata + pgvector-backed VectorStore
  -> PostgreSQL full-text or fake sparse baseline
  -> CLI ingest, search, context pack, and agent-run trace
  -> source-backed verification and replay/debug output
```

The first proof is not a polished product or chat UI. It is a working CLI loop
that shows the architecture can index project sources incrementally, retrieve
relevant evidence through hybrid paths, build an inspectable `ContextPack`,
execute a typed model/tool step with policy enforced outside the model, verify
source-backed claims, and replay the trace. It must prove neutral linguistic and
lexicographic contracts with simple or fake fixtures before adding production
`context-lang-*` adapters or TEI/SKOS/dictionary importers.

Expected first demo corpus: repository docs such as `README.md` and
`.project/*.md`, with proof artifacts suitable for downstream UX replay.

## Non-Goals For The First Version

- Chat UI, browser shell, or downstream `Lab` code inside the core module.
- Generic autonomous control of arbitrary systems.
- Unlimited web crawling.
- Plugin marketplace.
- Scenario-specific products in the core package tree.
- Multi-tenant billing.
- Complex UI framework ownership.
- Hard dependency on one model provider.
- Hard dependency on one vector database; QDrant and Turbopuffer remain optional
  later adapters behind `VectorStore`.
- Language-specific dictionaries, grammar rules, or morphology engines in core.
- TEI/SKOS importers, historical dictionaries, regional vocabularies, slang
  lexicons, or community lexicon resources in the first PoC.
- Implicit background writes without audit and approval policy.
- Full distributed worker orchestration.
- Production-grade query language.

## Engineering Notes

- Keep raw source storage, embeddings, metadata, indexes, traces, and generated
  artifacts separate.
- Version embedding models, parsers, chunkers, tokenizers, analyzers,
  dictionaries, sparse indexes, and graph schemas.
- Never replace original source text with normalized text.
- Do not collapse senses into lemmas or concepts into labels.
- Treat generated wordforms as expansion candidates, not as attestations unless
  they are witnessed in a source.
- Treat long tool outputs as artifacts that can be searched and read in slices.
- Prefer deterministic verification for claims that depend on source material.
- Record enough run data to reproduce bad retrieval and bad tool decisions.
- Enforce permission and side-effect decisions outside the model.
- Design every background action as an event with policy, trace, owner, and
  cancellation.

## Status

| Item | State |
| --- | --- |
| Planning baseline | Accepted (`roadmap-context-core.md`) |
| Architecture decisions | ADRs under `.project/decisions/` (through ADR-0024) |
| Go implementation | Phases 1–2 done (Chunks 01–20); Phase 3 starts Chunk 21 |
| Public API | `pkg/contextkit` HTTP client (Chunk 21) |
| Dependencies | Prefer stdlib + narrow adapters (`pgx` when Postgres gated) |

Public APIs are expected to change until the PoC CLI loop and core runtime
stabilize.

**License:** [MIT](LICENSE)