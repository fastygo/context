# context

Universal Go core for project-scoped context management, retrieval, and agent
orchestration.

The module is designed as an engine for systems that need to turn user intent,
files, documents, logs, tool outputs, and external sources into precise,
auditable context for automated work.

## Purpose

`github.com/fastygo/context` provides the foundation for:

- indexing project-specific knowledge;
- retrieving relevant evidence for a task;
- building compact context packs for model calls;
- routing work through tools and subagents;
- tracking provenance, decisions, and verification results;
- supporting foreground and background automation.

The goal is not to be a chat application. The goal is to provide a reliable
context operating layer that other products, companions, dashboards, and
automation systems can build on top of.

## Design Principles

- **Project scoped by default**: every index, artifact, run, and decision belongs
  to an explicit project or workspace.
- **Evidence before generation**: model calls should receive selected,
  inspectable context, not unbounded history.
- **Provenance is mandatory**: facts should point back to source spans,
  versions, checksums, or tool outputs.
- **Hybrid retrieval wins**: semantic vectors, keyword search, exact matching,
  graph traversal, and recency signals should cooperate.
- **Models are replaceable**: LLMs, embedding models, and rerankers are adapters,
  not hardcoded infrastructure.
- **Tools are typed**: every tool should have a name, schema, permission policy,
  risk level, and structured result.
- **Agents are configurations**: orchestration policy, rules, skills, and tool
  access should be data-driven where possible.
- **Background work is explicit**: scheduled or event-triggered agents should be
  observable, cancellable, and auditable.

## Core Concepts

### Project Memory

Project memory is the durable record of what the system knows and what it has
done.

Expected entities include:

- `Project`: an isolated workspace or tenant boundary.
- `Source`: a file, document, URL, log stream, database snapshot, or tool output.
- `Artifact`: stored source material or generated intermediate output.
- `Chunk`: an indexed span with metadata and provenance.
- `ContextPack`: the selected evidence passed into a model or agent step.
- `AgentRun`: a foreground or background execution trace.
- `ToolCall`: a typed invocation with inputs, outputs, status, and permissions.
- `Decision`: an accepted plan, spec, architectural note, or human approval.
- `Evaluation`: a reproducible check for retrieval quality or task correctness.

### Indexing Pipeline

The indexing pipeline should be source-agnostic:

```text
source adapter
  -> parser
  -> chunker
  -> enricher
  -> embedder
  -> sparse index
  -> vector index
  -> graph index
  -> manifest
```

Different source types need different chunking strategies. Source code,
technical documentation, scientific text, chat history, logs, and web pages
should not be split with the same rules.

### Retrieval Pipeline

Retrieval is a planning problem, not a single vector search.

```text
task intent
  -> retrieval plan
  -> parallel searches
  -> candidate merge
  -> reranking
  -> evidence validation
  -> context pack
```

Supported retrieval strategies should include:

- dense vector search;
- sparse or keyword search;
- exact phrase and symbol search;
- source and citation lookup;
- entity and metadata filtering;
- graph traversal;
- recent activity search;
- tool result search;
- external source search.

### Context Packs

A `ContextPack` is the central handoff object between retrieval, models, tools,
and agents.

It should contain:

- task summary;
- selected evidence;
- source references;
- confidence and ranking signals;
- excluded or rejected candidates when useful;
- model budget hints;
- instructions for the next step;
- verification requirements.

Context packs should be versioned and replayable so behavior can be debugged,
evaluated, and improved.

### Tool Registry

Tools should be registered through typed metadata:

```text
tool
  name
  description
  input schema
  output schema
  permission policy
  risk level
  cost class
  latency class
  background support
```

The runtime should be able to decide whether a tool can run automatically,
requires user approval, or is forbidden in the current policy.

### Agent Runtime

Agents coordinate retrieval, model calls, tools, and subagents.

An agent should be defined by:

- system instructions;
- rules and skills;
- model preferences;
- available tools;
- approval policy;
- context budget;
- foreground or background mode;
- verification requirements.

Subagents should run with isolated context and return structured summaries or
artifacts to the parent agent. This keeps noisy exploration, long-running shell
work, browser automation, research, and verification from polluting the main
task context.

## Suggested Package Layout

The public API should stay small until the engine boundaries are stable.

```text
internal/
  agents/       # agent runtime, runs, subagent coordination
  adapters/     # source, model, storage, and tool adapters
  evals/        # retrieval and task evaluation harnesses
  graph/        # entity, citation, dependency, and relation indexes
  indexing/     # parsing, chunking, enrichment, manifests
  memory/       # project memory, artifacts, provenance
  models/       # LLM, embedding, and reranker interfaces
  retrieval/    # planners, retrievers, rerankers, context packs
  tools/        # registry, schemas, permissions, execution

pkg/
  contextkit/   # stable public interfaces, added only when needed
```

Prefer `internal` while interfaces are changing. Move packages to `pkg` only
when another module needs a stable import surface.

## MVP Scope

The first useful version should prove the full loop:

```text
intent
  -> retrieve project context
  -> build context pack
  -> generate a spec or plan
  -> call typed tools
  -> verify result
  -> persist trace and decisions
```

Recommended MVP components:

- project memory schema;
- source adapters for files and stored artifacts;
- incremental indexing manifest;
- dense and sparse retrieval;
- model provider abstraction;
- tool registry and permission policy;
- agent run tracing;
- context pack inspection;
- basic verifier for source-backed answers.

## Non-Goals For The First Version

- generic autonomous control of arbitrary systems;
- unlimited web crawling;
- plugin marketplace;
- multi-tenant billing;
- complex UI framework ownership;
- hard dependency on one model provider;
- hard dependency on one vector database;
- implicit background writes without audit and approval policy.

## Engineering Notes

- Keep raw source storage, embeddings, metadata, and generated artifacts as
  separate concerns.
- Version embedding models and chunking algorithms; retrieval quality depends on
  both.
- Treat long tool outputs as artifacts that can be searched and read in slices.
- Prefer deterministic verification for claims that depend on source material.
- Record enough run data to reproduce bad retrieval and bad tool decisions.
- Design every background action as an event with policy, trace, and ownership.

## Status

Early design-stage module. Public APIs are expected to change until the core
runtime and MVP workflows stabilize.

**License:** [MIT](LICENSE)