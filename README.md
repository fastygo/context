# context

Universal Go core for project-scoped context management, retrieval, indexing,
and agent orchestration.

`github.com/fastygo/context` is not a chat application. It is a reusable context
operating layer for systems that need to turn user intent, files, documents,
logs, tool outputs, and external sources into precise, inspectable, auditable
context for automated work.

## Why

Large language models are useful, but they do not solve context management by
themselves. A serious agent system needs to know what evidence was selected,
where it came from, why other evidence was rejected, which tool was allowed to
run, which model saw which context, and how the final result can be replayed or
debugged.

This module exists because plain RAG is not enough for long-lived projects:

- vector search misses exact facts, citations, morphology, and operational
  boundaries;
- unbounded chat history is noisy, lossy, and hard to audit;
- tool calls without typed policy create hidden side effects;
- background agents need ownership, cancellation, traces, and verification;
- source-backed work must preserve spans, versions, checksums, and decisions;
- model, vector, storage, and tool providers must remain replaceable.

The engine is designed to combine deterministic information retrieval,
source-backed context packs, typed tools, replaceable model adapters, and
agent/subagent orchestration without baking product identity or scenario-specific
products into the core.

## What It Provides

The module is intended to provide foundations for:

- project-scoped source and artifact memory;
- deterministic indexing with manifests and incremental updates;
- hybrid retrieval over dense vectors, sparse/keyword indexes, exact matching,
  graph traversal, source filters, recency, and tool outputs;
- focus policies that keep retrieval and context packing scoped to the current
  task;
- replayable `ContextPack` objects for model calls, tools, and subagents;
- typed tool registration with permissions, risk levels, and structured results;
- foreground and background `AgentRun` traces;
- verification and evaluation loops for retrieval quality and factuality;
- adapter boundaries for LLMs, embeddings, rerankers, vector stores, metadata
  stores, artifact stores, crawlers, and product integrations.

## Design Principles

- **Project scoped by default**: every index, artifact, run, and decision belongs
  to an explicit project or workspace.
- **Evidence before generation**: model calls receive selected context, not
  unbounded history.
- **Provenance is mandatory**: facts point back to source spans, versions,
  checksums, or tool outputs.
- **Hybrid retrieval wins**: dense vectors, sparse search, exact matching,
  morphology, graph traversal, and recency signals cooperate.
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
- **Specific products are plugins**: message catalogs, timelines, CRM flows,
  calendars, dashboards, and methodology packs belong in adapters or downstream
  repositories.

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

The central object is `ContextPack`: the selected, ranked, budget-aware,
source-backed context handed to a model, tool, verifier, or subagent. It should
be versioned and replayable so bad retrieval, bad generation, or bad tool
decisions can be debugged later.

## Core Concepts

- `Project`: an isolated workspace or tenant boundary.
- `Source`: a file, document, URL, log stream, database snapshot, spec, chat
  history, or tool output.
- `Artifact`: stored source material or generated intermediate output.
- `Chunk`: an indexed span with metadata and provenance.
- `FocusProfile`: the task-specific lens that defines scope, freshness,
  exactness, citation strictness, budgets, allowed tools, and irrelevant areas.
- `ContextPack`: selected evidence and instructions for a model/tool/agent step.
- `AgentRun`: a foreground, background, scheduled, or event-triggered execution
  trace.
- `ToolCall`: a typed invocation with input, output, status, permissions, and
  side-effect metadata.
- `Decision`: an accepted plan, spec, architectural note, or human approval.
- `Evaluation`: a reproducible check for retrieval quality or task correctness.

## Indexing Pipeline

The indexing pipeline is source-agnostic:

```text
source adapter
  -> artifact store
  -> parser
  -> chunker
  -> enricher
  -> manifest
  -> dense vector index
  -> sparse/exact index
  -> graph index
  -> metadata store
```

Different source types need different chunking strategies. Source code,
technical documentation, scientific text, legal text, chat history, logs, web
captures, and tool output should not be split with the same rules.

## Retrieval Pipeline

Retrieval is a planning problem, not one vector query:

```text
task intent
  -> focus profile
  -> retrieval plan
  -> parallel retriever calls
  -> candidate merge
  -> deduplication
  -> reranking
  -> evidence validation
  -> context pack
```

Supported retrieval paths should include:

- dense vector search;
- sparse/BM25-style search;
- exact phrase and source-span search;
- entity and metadata filters;
- citation lookup;
- graph traversal;
- recent activity retrieval;
- tool result retrieval;
- external source retrieval through explicit adapters.

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
or artifacts to the parent run. This keeps broad search, long-running shell
work, browser automation, research, and verification from polluting the main
task context.

Concrete products should integrate through adapters, tools, graph projections,
rules, skills, contracts, or companion configuration. The core should not know
whether a source came from a messaging archive, calendar, issue tracker, CRM, or
enterprise catalog beyond generic source metadata and adapter-provided fields.

## Architecture Guidance

The canonical planning documents live in `.project/`:

- `.project/roadmap-context-core.md` — architectural baseline and phased roadmap.
- `.project/progress.md` — copy-paste plan chunks from baseline to PoC.
- `.project/future-layer.md` — deferred production-grade layers and review gates.

The project skill lives in `.cursor/skills/context-core-steward/` and should be
used when planning, implementing, reviewing, or debugging this repository. It
keeps work aligned with DDD, Clean Architecture, SOLID, DRY, TDD, traceability,
brand-neutral API boundaries, and the current roadmap.

## Suggested Package Direction

The public API should stay small until boundaries are stable.

```text
cmd/
  context-dev/              # local developer CLI for indexing, search, evals

internal/
  agentruntime/             # agent runs, orchestration, subagents, scheduling
  artifacts/                # artifact metadata and stores
  config/                   # project config, rules, ignore patterns
  corpus/                   # projects, sources, chunks, provenance
  evals/                    # retrieval and task evaluation harnesses
  graph/                    # entity, citation, co-occurrence, dependency edges
  indexing/                 # parsing, chunking, enrichment, manifests
  linguistic/               # normalization, tokens, morphology, fuzzy matching
  models/                   # LLM, embedding, reranker interfaces
  policy/                   # permissions, risks, approvals
  retrieval/                # planners, retrievers, rerankers, context packs
                            # and focus profiles
  storage/                  # metadata store abstractions and adapters
  tools/                    # registry, schemas, execution
  tracing/                  # append-only runtime events and redaction

pkg/
  contextkit/               # stable public interfaces, added only when proven
```

Prefer `internal` while interfaces are changing. Move packages to `pkg` only
when another module needs a stable import surface.

## First Proof Target

The current hypothesis-validation path is:

```text
local project corpus
  -> deterministic indexing
  -> QDrant + PostgreSQL-backed metadata/search path
  -> real CLI ingestion and retrieval
  -> context pack creation
  -> fake model/tool agent run
  -> source-backed verification trace
```

The first proof is not a polished product. It is a working CLI loop that shows
the architecture can ingest project sources, retrieve relevant evidence, build a
context pack, execute a typed model/tool step, verify source-backed claims, and
replay the trace.

## Non-Goals For The First Version

- Generic autonomous control of arbitrary systems.
- Unlimited web crawling.
- Plugin marketplace.
- Scenario-specific products in the core package tree.
- Multi-tenant billing.
- Complex UI framework ownership.
- Hard dependency on one model provider.
- Hard dependency on one vector database.
- Implicit background writes without audit and approval policy.
- Full distributed worker orchestration.
- Production-grade query language.

## Engineering Notes

- Keep raw source storage, embeddings, metadata, indexes, traces, and generated
  artifacts separate.
- Version embedding models, parsers, chunkers, enrichers, sparse indexes, and
  graph schemas.
- Treat long tool outputs as artifacts that can be searched and read in slices.
- Prefer deterministic verification for claims that depend on source material.
- Record enough run data to reproduce bad retrieval and bad tool decisions.
- Enforce permission and side-effect decisions outside the model.
- Design every background action as an event with policy, trace, owner, and
  cancellation.

## Status

Early design-stage module. Public APIs are expected to change until the core
runtime and proof-of-concept workflows stabilize.

**License:** [MIT](LICENSE)