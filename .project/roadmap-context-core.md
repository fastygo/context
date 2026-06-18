# Context Core Engineering Roadmap

Status: planning baseline  
Scope: `github.com/fastygo/context` as a brand-neutral context-management,
retrieval, indexing, and agent orchestration engine.

This roadmap treats the module as infrastructure for project-scoped corpora,
scientific/legal/linguistic text retrieval, browser assistants, background
agents, and downstream companions. Product-specific personas, names, workflows,
and UI surfaces belong in consumer applications or adapters, not in the reusable
core.

## Executive Summary

The engine should become a reliable context operating layer, not a generic chat
application and not a thin vector-search wrapper. Its core responsibility is to
turn project-local sources, documents, logs, web captures, tool outputs, rules,
skills, and user intent into inspectable, replayable, source-backed context
packs that agents and tools can safely act on.

The critical design choice is to make `ContextPack` the central runtime object.
Every model call, tool call, subagent handoff, and verifier step should be
traceable back to a task, retrieval plan, selected evidence, rejected evidence,
source spans, checksums, permissions, and evaluation signals.

The first implementation should stay deliberately small: one local project, one
file/artifact source adapter, one QDrant-backed vector store adapter, one sparse
or keyword search path, one context-pack builder, one tool registry, one agent
run trace, and one deterministic verifier. Horizontal scale, copy-on-write
index reuse, background agents, billing, multi-tenant security, and browser UI
should be designed for, but not prematurely implemented.

## Product-Agnostic Design Boundaries

### Core Responsibilities

- Project-scoped source registration and artifact tracking.
- Deterministic source parsing, chunking, enrichment, and manifest generation.
- Hybrid retrieval over dense vectors, sparse/keyword indexes, exact matching,
  metadata filters, recency, graph edges, and tool outputs.
- Context pack construction with source-backed evidence and model budget hints.
- Model, embedding, reranker, storage, and vector database adapter boundaries.
- Typed tool registration, permission policy, execution traces, and structured
  outputs.
- Agent and subagent run coordination with isolated context, resumable traces,
  and foreground/background execution modes.
- Evaluation harnesses for retrieval quality, factuality, context packing,
  tool decisions, and task outcomes.

### Non-Core Responsibilities

- Brand-specific companion names, mascots, prompts, or marketing copy.
- Product-specific UI, dashboard layout, CRM screens, or generated app code.
- Arbitrary remote code execution without a consumer-provided sandbox policy.
- Billing and subscription enforcement in the library core.
- A hard dependency on any one LLM provider, embedding provider, vector store,
  crawler, database, or web framework.

## Current Repository Findings

- `README.md` correctly frames the module as a universal Go core for
  project-scoped context management, retrieval, and agent orchestration.
- The project rule requires a brand-neutral core. Keep named companions and
  product-specific use cases outside the general API and documentation.
- `go.mod` is intentionally minimal and currently has no dependencies.
- `.project` is intentionally reduced to this roadmap, the progress tracker, and
  the future-layer backlog. Future agents should treat these files as the
  self-contained architectural source of truth.

## Self-Contained Conceptual Baseline

This section preserves the essential design intent so the roadmap does not
depend on deleted brainstorming notes.

### Cursor-Style Context Management

The engine should adapt the useful parts of modern coding-agent context
management without copying IDE-specific assumptions:

- Project-scoped index namespaces.
- Incremental indexing through manifests, checksums, and Merkle-style diffs.
- Hybrid retrieval instead of one-shot vector search.
- Dynamic context discovery through searchable artifacts instead of oversized
  prompts.
- Typed tool calls with policy decisions outside the model.
- Subagent isolation for noisy exploration, verification, research, and
  long-running background work.
- Replayable traces for retrieval plans, context packs, model calls, tool calls,
  and verifier results.

### Private Search And Archive-Style Retrieval

The engine should also preserve the deterministic strengths of classic private
search systems:

- Compact inverted indexes for exact and keyword retrieval.
- Position-aware spans so results can show precise evidence, not only document
  IDs.
- Incremental crawling/indexing based on checksums and modification state.
- Strong local/offline operation for private corpora.
- Predictable boolean, phrase, proximity, and citation-style lookup.
- Separation between raw artifacts, metadata, indexes, and generated outputs.

Historical systems and algorithms are design inspiration only. Public APIs must
use neutral generic terminology such as `source`, `artifact`, `chunk`,
`retriever`, `context pack`, `tool`, and `agent`.

### Deterministic IR And Linguistic Layer

The first retrieval layer should be deterministic and inspectable before adding
large-model behavior:

- Unicode normalization and script-aware tokenization.
- Lemmas, stems, surface forms, and language-specific morphology through
  adapters.
- Sparse/BM25-style retrieval for lexical precision.
- Exact phrase and source-span lookup for citations and legal/scientific text.
- N-gram, trigram, or sparse n-gram indexes when measured latency requires them.
- Damerau-Levenshtein, Jaro-Winkler, and phonetic hooks for typo and name
  matching.
- Text metrics such as entropy, density, repetition, and stop-word ratio where
  they help analysis or quality gates.

### Neural Layer

Neural models should be replaceable accelerators, not the source of truth:

- Dense embeddings help broad semantic recall.
- Rerankers help order mixed candidate sets.
- LLMs help classify intent, summarize evidence, draft specs, and plan tool
  calls.
- Model outputs must remain downstream of source-backed evidence for factual,
  scientific, legal, or operational claims.

### Browser Assistant And Background Runtime

The intended downstream shape is a browser-facing assistant and background
runtime over a persistent per-project corpus:

- The browser or product UI should call a BFF/API layer.
- The core should expose project memory, retrieval, context pack, tool, and
  agent runtime primitives.
- Background runs should be event-triggered or scheduled with explicit owner,
  policy, trace, cancellation, and report.
- The core must not assume that every interaction is a chat message.

## Architecture Tenets

1. Evidence before generation.
   Models should not see unbounded chat history. They should receive selected,
   ranked, inspectable evidence with citations and constraints.

2. Local/project scope first.
   Every artifact, chunk, index namespace, run, decision, and tool result must
   belong to a project or workspace boundary.

3. Hybrid retrieval by design.
   Dense vectors are useful but insufficient. Exact search, morphology,
   sparse/BM25 retrieval, graph traversal, source filters, recency, and
   deterministic verification are first-class retrieval paths.

4. Source spans are mandatory.
   Factual answers, legal/scientific claims, generated specs, and agent actions
   should be traceable to source spans, checksums, document versions, or tool
   outputs.

5. Context packs are versioned artifacts.
   A bad answer must be debuggable by replaying the retrieval plan, context pack,
   model request, tool calls, and verifier result.

6. Adapters own infrastructure.
   QDrant, Postgres, object storage, local filesystem, specific LLMs, web
   crawlers, and product integrations must be replaceable behind interfaces.

7. Background work is policy-bound.
   Event-triggered agents, scheduled monitors, and long-running subagents must
   be observable, cancellable, permission-scoped, and auditable.

8. Optimize after correctness.
   Start with clear data contracts and deterministic tests. Add zero-copy,
   mmap, sparse n-gram indexes, copy-on-write reuse, SIMD, and distributed
   execution only when measured bottlenecks justify them.

## Target Runtime Model

```text
TaskIntent
  -> PolicySnapshot
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

The runtime should prefer pull-based dynamic context discovery. Long tool
outputs, web captures, crawler logs, terminal output, model traces, and
subagent intermediate work should be stored as artifacts that can be searched
and read in slices instead of being injected wholesale into prompts.

## Proposed Package Layout

Keep the implementation under `internal` until interfaces stabilize. Expose
only minimal stable contracts through `pkg/contextkit` after at least one real
consumer uses them successfully.

```text
cmd/
  context-dev/              # local developer CLI for indexing, search, evals

internal/
  agentruntime/
    run.go                  # AgentRun lifecycle and state transitions
    orchestrator.go         # parent agent coordination
    subagent.go             # isolated subagent execution contracts
    scheduler.go            # background run scheduling abstraction

  artifacts/
    artifact.go             # durable raw/source/generated artifact metadata
    store.go                # artifact storage interface
    localfs/                # local filesystem artifact store

  config/
    project.go              # project/workspace config loader
    rules.go                # rules/skills/policy config parsing
    ignore.go               # ignore/indexing-ignore patterns

  corpus/
    project.go              # project boundary and corpus identifiers
    source.go               # source registration and source versions
    chunk.go                # chunk model and span metadata
    provenance.go           # source references, checksums, evidence lineage

  evals/
    dataset.go              # golden retrieval and task datasets
    runner.go               # reproducible eval runner
    metrics.go              # recall@k, MRR, citation accuracy, tool accuracy

  graph/
    edge.go                 # entity, citation, co-occurrence, dependency edges
    store.go                # graph store interface
    traversal.go            # bounded graph expansion for retrieval

  indexing/
    pipeline.go             # source -> parser -> chunker -> enrichers -> stores
    manifest.go             # Merkle tree, checksums, index state
    simhash.go              # index similarity fingerprint
    parser.go               # parser interface
    chunker.go              # chunker interface
    enricher.go             # metadata, morphology, entity enrichment

  indexing/parsers/
    plaintext/
    markdown/
    html/
    json/

  indexing/chunkers/
    textwindow/             # safe baseline text chunker
    markdownstruct/         # heading/section-aware chunker
    citationaware/          # paragraph/citation/span-aware chunker

  linguistic/
    normalize.go            # Unicode, casing, punctuation, script handling
    token.go                # tokens, lemmas, stems, surface forms
    morphology.go           # morphology adapter interface
    fuzzy.go                # edit distance, Jaro-Winkler, phonetic hooks
    metrics.go              # water ratio, density, entropy, term statistics

  models/
    llm.go                  # completion/chat model interface
    embedding.go            # embedding model interface
    reranker.go             # reranker interface
    registry.go             # model adapter selection and versioning

  policy/
    permission.go           # allow/ask/deny policies
    risk.go                 # tool and action risk levels
    approval.go             # human approval contracts

  retrieval/
    planner.go              # task -> retrieval plan
    retriever.go            # retriever interface
    candidates.go           # merge, dedupe, score normalization
    rerank.go               # reranking pipeline
    contextpack.go          # central handoff artifact
    budget.go               # token/window/model budget planner
    verifier.go             # evidence validation before generation

  retrieval/dense/
    qdrant/                 # QDrant adapter
    memory/                 # in-memory test adapter

  retrieval/sparse/
    inverted/               # baseline inverted index
    trigram/                # future exact/fuzzy phrase index

  storage/
    tx.go                   # transaction boundary abstractions
    memory/                 # deterministic tests
    postgres/               # future durable metadata adapter

  tools/
    schema.go               # typed input/output schema model
    registry.go             # tool registration and discovery
    executor.go             # execution, policy, tracing
    result.go               # structured output and artifact references

  tracing/
    event.go                # append-only runtime events
    recorder.go             # run/tool/retrieval trace writer
    redaction.go            # secrets and PII redaction hooks

pkg/
  contextkit/
    doc.go                  # stable public surface only after core hardens
```

## Data Model Baseline

### Project

Represents the isolation boundary for indexes, artifacts, runs, and decisions.

Required fields:

- `id`
- `slug`
- `display_name`
- `created_at`
- `updated_at`
- `policy_id`
- `metadata`

### Source

Represents an origin of knowledge or runtime data.

Required fields:

- `id`
- `project_id`
- `source_type`: `file`, `url`, `document`, `log`, `tool_output`,
  `chat_history`, `spec`, `database_snapshot`
- `uri`
- `version`
- `checksum`
- `content_type`
- `language`
- `trust_level`
- `created_at`
- `updated_at`

### Artifact

Stores raw or generated material separately from metadata.

Required fields:

- `id`
- `project_id`
- `source_id`
- `artifact_type`
- `storage_uri`
- `checksum`
- `byte_size`
- `media_type`
- `retention_policy`

### Chunk

Represents an indexed span.

Required fields:

- `id`
- `project_id`
- `source_id`
- `artifact_id`
- `chunker_version`
- `span_start`
- `span_end`
- `text_checksum`
- `language`
- `tokens`
- `lemmas`
- `entities`
- `citations`
- `metadata`
- `embedding_version`
- `sparse_version`

### ContextPack

Represents selected evidence for a model, tool, or subagent step.

Required fields:

- `id`
- `project_id`
- `task_id`
- `retrieval_plan_id`
- `purpose`
- `budget`
- `evidence_items`
- `rejected_items`
- `instructions`
- `verification_requirements`
- `created_at`
- `checksum`

### AgentRun

Represents an orchestrated foreground or background execution.

Required fields:

- `id`
- `project_id`
- `parent_run_id`
- `mode`: `foreground`, `background`, `scheduled`, `event_triggered`
- `agent_config_id`
- `policy_snapshot_id`
- `status`
- `started_at`
- `finished_at`
- `trace_uri`

### ToolCall

Represents typed tool execution.

Required fields:

- `id`
- `project_id`
- `agent_run_id`
- `tool_name`
- `input_schema_version`
- `input_checksum`
- `permission_decision`
- `risk_level`
- `status`
- `output_ref`
- `error`
- `started_at`
- `finished_at`

## Indexing Architecture

### Pipeline

```text
SourceAdapter
  -> ArtifactStore
  -> Parser
  -> Chunker
  -> Enricher
  -> ManifestBuilder
  -> DenseWriter
  -> SparseWriter
  -> GraphWriter
  -> MetadataStore
```

### Source Adapters

Phase 1 adapters:

- Local files.
- Stored artifacts.
- Plain text and Markdown.

Phase 2 adapters:

- HTML/web captures.
- JSON and structured data snapshots.
- Logs.
- Chat history.

Later adapters:

- PDF, DOCX, email archives, compressed archives.
- Browser history and page snapshots.
- Product-specific data stores through external modules.

### Chunking Strategy

Do not use one generic token splitter for all sources.

Baseline chunkers:

- Plain text: paragraph-aware chunking with max token budget and overlap.
- Markdown: heading/section-aware chunks preserving heading ancestry.
- Scientific/legal text: citation-aware chunks preserving paragraph IDs,
  footnotes, references, quoted spans, and edition/version metadata.
- Logs: event-window chunks grouped by time, service, request ID, and severity.
- Tool output: command/tool sections with stdout/stderr/status separation.

Future chunkers:

- Source code AST-aware chunking.
- Dialogue turn-aware chunking.
- Table-aware chunking.
- Formula-aware chunking.

### Manifest And Incremental Indexing

Implement a Merkle-style manifest before optimizing ingestion.

Baseline:

- Leaf hash: normalized source bytes or parsed text span.
- Parent hash: deterministic hash of child names and hashes.
- Manifest version: includes parser, chunker, enricher, embedding, sparse, and
  graph schema versions.
- Diff mode: only reparse/rechunk/reindex changed branches.

Future:

- Simhash over manifest leaves to find near-duplicate corpora.
- Copy-on-write index reuse across project templates or team workspaces.
- Content proofs to prevent shared-index leakage.

### QDrant Strategy

Use QDrant as an adapter, not as the domain model.

Recommended initial mapping:

- One collection per environment or tenant class, with `project_id` payload
  filters, unless operational tests prove collection-per-project is better.
- Payload fields: `project_id`, `source_id`, `artifact_id`, `chunk_id`,
  `span_start`, `span_end`, `language`, `source_type`, `tags`, `entities`,
  `embedding_version`, `chunker_version`, `trust_level`.
- Dense vectors for semantic similarity.
- Sparse vectors or a sidecar sparse index for BM25-like retrieval.

Decision to revisit:

- Collection-per-project is conceptually clean and close to Cursor-style
  namespaces, but may create operational overhead with many small projects.
- Shared collection with strict payload filters is easier to operate early.
- Build an internal `VectorNamespace` abstraction so this can change later.

## Retrieval Architecture

### Retrieval Planner

The planner maps a task to one or more retrievers.

Inputs:

- User intent or event trigger.
- Project policy.
- Available indexes.
- Current active artifacts.
- Model budget.
- Risk level.
- Required confidence.

Outputs:

- Dense semantic searches.
- Sparse keyword searches.
- Exact phrase searches.
- Entity/citation lookups.
- Graph traversals.
- Recent activity retrieval.
- Tool-output retrieval.
- Web/external retrieval requests when enabled.

### Candidate Merge

Candidate merge must be deterministic and explainable.

Required behaviors:

- Deduplicate by `source_id + span_start + span_end + checksum`.
- Preserve all contributing scores.
- Normalize scores per retriever type.
- Track why a candidate was included or rejected.
- Enforce source access and project boundary filters before reranking.

### Reranking

Phase 1:

- Weighted score merge.
- Recency boost.
- Exact-match boost.
- Trust-level boost.
- Citation-presence boost for factual tasks.

Phase 2:

- Model-based reranker adapter.
- Query rewriting with trace.
- Task-specific reranking policies.

### Context Pack Builder

Context pack construction must be budget-aware.

Required behaviors:

- Keep source spans complete where possible.
- Avoid duplicate evidence.
- Include short evidence summaries only after preserving source references.
- Separate facts, instructions, policies, and tool outputs.
- Include verification requirements.
- Store rejected high-scoring candidates for debugging when useful.

## Linguistic And NLP Considerations

The engine should combine deterministic NLP and neural retrieval instead of
choosing one side.

### Deterministic Layer

Implement or adapt in phases:

- Unicode normalization and script detection.
- Tokenization.
- Stop-word handling.
- Lemmatization/morphology adapter interface.
- Damerau-Levenshtein for typo-tolerant matching.
- Jaro-Winkler for prefix-sensitive fuzzy matching.
- N-gram fingerprints for candidate narrowing.
- Phonetic hooks such as Metaphone/Double Metaphone where relevant.
- Text metrics: water ratio, keyword density, entropy, repeated-term patterns.

### Neural Layer

Use neural models where they add value:

- Dense semantic retrieval.
- Query expansion.
- Reranking.
- Summarization.
- Intent classification.
- Spec drafting.

Do not use neural outputs as unverified facts. Factual and scientific claims
must be backed by retrieved evidence spans or explicit external tool results.

### Scientific/Legal/Linguistic Text Requirements

These corpora need stricter provenance than normal documentation:

- Stable document IDs.
- Edition/version metadata.
- Page/paragraph/section spans.
- Citation graph.
- Quote preservation.
- Source trust levels.
- Explicit uncertainty.
- Verification matrix for claims.

## Agent Runtime

### Agent Configuration

Agents are configurations over the core.

Required fields:

- `name`
- `description`
- `instructions`
- `model_preferences`
- `available_tools`
- `retrieval_policy`
- `approval_policy`
- `context_budget`
- `background_allowed`
- `verification_requirements`

### Subagents

Subagents should be isolated workers with narrow roles.

Initial generic subagents:

- `explorer`: broad project/corpus search and source mapping.
- `researcher`: external source search and capture, if enabled.
- `specifier`: converts intent and evidence into implementation or analysis
  specs.
- `verifier`: checks claims, citations, tool results, and acceptance criteria.
- `monitor`: background checks over configured systems and artifacts.

Each subagent receives a task package:

- Task summary.
- Allowed tools.
- Context pack or retrieval permission.
- Budget.
- Expected output schema.
- Stop conditions.

The parent run should receive structured artifacts, not raw noisy logs.

### Background Agents

Background agents are event-driven or scheduled. They should never be implicit.

Examples:

- A project source changed.
- A document corpus was updated.
- A health check failed.
- A user action implies a missing schema or configuration.
- A scheduled daily summary is due.

Every background run must have:

- Trigger event.
- Owner.
- Policy snapshot.
- Allowed actions.
- Approval mode.
- Cancellation path.
- Trace and report.

## Tool System

### Tool Metadata

Every tool must be registered with:

- Name.
- Description.
- Input schema.
- Output schema.
- Risk level.
- Permission policy.
- Timeout.
- Retry policy.
- Idempotency marker.
- Background support.
- Artifact output behavior.

### Permission Levels

Start with:

- `read`: safe introspection.
- `suggest`: creates plans/specs but no external side effects.
- `write_project`: modifies project-owned artifacts.
- `execute`: runs commands or external operations.
- `network`: reaches external services.
- `admin`: changes policy, credentials, or billing-sensitive state.

### Tool Execution

Execution must:

- Validate input schema before execution.
- Record policy decision.
- Enforce timeout.
- Persist structured result.
- Store long output as artifact.
- Redact secrets before model-visible summaries.
- Emit trace events.

## Storage And Scaling Strategy

### Proof Of Concept

- In-memory metadata store for tests.
- Local filesystem artifact store.
- QDrant local/dev adapter.
- Optional SQLite or Postgres only if persistence is needed immediately.

### MVP

- Postgres for metadata, runs, decisions, permissions, and eval results.
- QDrant for vector search.
- Local or S3-compatible object storage for artifacts.
- Background job queue abstraction.

### Stable

- Postgres with migrations and backups.
- QDrant cluster or managed vector store.
- Object storage lifecycle policies.
- Redis/NATS/Temporal-like queue adapter if needed.
- Per-tenant/project quotas.
- Cold/warm artifact and index policies.

### Horizontal Scaling

Design these boundaries early:

- Stateless API/BFF processes.
- Worker pool for indexing.
- Worker pool for agent runs.
- Separate vector store.
- Separate metadata store.
- Separate artifact store.
- Queue-mediated long-running tasks.

### Vertical Scaling

Optimize after measurement:

- Chunk batching.
- Embedding batching.
- mmap or compact posting lists for sparse search.
- Delta-coded posting lists.
- Sparse n-gram indexes.
- SIMD or cgo/Rust only for proven hot loops.

## Cursor-Like Features To Adapt Carefully

### Adopt Early

- Project-scoped namespaces.
- Incremental indexing manifest.
- Hybrid retrieval.
- Context packs.
- Dynamic context discovery through artifacts.
- Subagent isolation.
- Rules/skills as files or config.
- Traceable tool calls.

### Defer

- Cross-user copy-on-write index reuse.
- Content-proof filtering for shared copied indexes.
- Custom embedding model trained on run traces.
- Large-scale instant regex index.
- Browser/desktop remote control.
- Team-wide enforced policies.

### Avoid Blind Copying

- Code AST assumptions do not map directly to scientific/legal text.
- Embeddings are not a substitute for morphology or citation graphs.
- Long autonomous runs are unsafe without approval policy and replayable traces.
- Namespace-per-project may be operationally costly without testing QDrant
  behavior at high namespace counts.

## Roadmap

### Phase 0: Architecture Baseline

Goal: lock the boundaries before writing broad implementation code.

Deliverables:

- This roadmap accepted as planning baseline.
- Architecture decision records for:
  - QDrant namespace strategy.
  - Metadata store choice.
  - Artifact store choice.
  - First supported source types.
  - First supported model providers.
- Package layout skeleton under `internal`.
- Core domain models for project, source, artifact, chunk, context pack,
  agent run, tool call, and evaluation.
- Interface-only boundaries for models, storage, retrieval, indexing, tools,
  and tracing.

Exit criteria:

- Public API remains minimal.
- Tests compile with no external services.
- The first consumer can be planned without changing core terminology.

### Phase 1: Proof Of Concept

Goal: prove the full context loop locally.

Scope:

- File and artifact source adapters.
- Plain text and Markdown parsing.
- Paragraph and Markdown section chunking.
- Local artifact store.
- In-memory metadata store.
- QDrant dense vector adapter behind an interface.
- Simple sparse keyword retriever.
- Retrieval planner with dense + sparse + exact path.
- Context pack builder.
- One LLM adapter interface with a fake deterministic test model.
- Tool registry with one read-only example tool.
- Agent run trace.
- Verifier requiring source-backed evidence.

Demo flow:

```text
user intent
  -> retrieve project documents
  -> build context pack
  -> generate a source-backed spec
  -> run a read-only tool
  -> verify claims
  -> persist trace
```

Exit criteria:

- A project can be indexed incrementally.
- A query can retrieve dense and sparse candidates.
- A context pack is persisted and inspectable.
- A model/tool/subagent step can be replayed from stored trace metadata.
- Factual output includes source spans.

### Phase 2: MVP

Goal: make the engine usable by a real downstream product.

Scope:

- Durable Postgres metadata adapter.
- Artifact store abstraction with local and S3-compatible implementations.
- QDrant payload filters and versioned embeddings.
- Ignore patterns for source and indexing exclusion.
- Rule and skill config loading.
- Tool permission policy.
- Background job abstraction.
- Explorer and verifier subagents.
- Web capture adapter with strict crawl limits.
- Eval harness with golden retrieval datasets.
- Context inspector output format for browser UI consumers.

Exit criteria:

- Multiple projects can coexist safely.
- Indexing and retrieval are deterministic under test fixtures.
- A downstream product can register tools without modifying core code.
- Model provider can be swapped through config.
- The system can explain why a context pack was built.
- Verifier catches unsupported factual claims in tests.

### Phase 3: Reliable Beta

Goal: support external users with controlled reliability.

Scope:

- Multi-tenant isolation design.
- Project quotas.
- Background agent scheduling.
- Tool approval workflows.
- Retrieval regression benchmarks.
- Redaction and PII handling.
- Structured operational metrics.
- Failure dashboards.
- Index rebuild and repair tools.
- Backfill and migration strategy.
- Initial sparse n-gram or trigram index if exact/fuzzy search latency requires
  it.

Exit criteria:

- Indexing failures are recoverable.
- Tool failures are visible and retryable.
- Background runs are cancellable.
- Retrieval metrics are tracked over time.
- Incident debugging does not require reading raw database rows manually.

### Phase 4: Stable Commercial-Grade Core

Goal: provide a reliable foundation for paid products and self-hosted users.

Scope:

- Hardened storage migrations.
- Backup and restore procedures.
- Tenant and project-level retention policies.
- Audit logs.
- Fine-grained permissions.
- Cost accounting hooks for model calls, embeddings, tools, storage, and
  background jobs.
- Copy-on-write index reuse for project templates.
- Content-proof filtering if shared index reuse crosses users or teams.
- Custom retrieval-model training data export from agent traces.
- Compatibility and migration policy for public interfaces.

Exit criteria:

- Versioned APIs.
- Reproducible deployments.
- Clear operational SLOs.
- Security review passed.
- Performance budgets documented.

### Phase 5: Ecosystem

Goal: allow third parties to build companions, adapters, tools, and corpora on
top of the core.

Scope:

- Stable `pkg/contextkit` API.
- Adapter SDK.
- Tool SDK.
- Companion configuration format.
- Self-hosting guide.
- Example projects with neutral names.
- Reference eval datasets.
- Compatibility tests for third-party adapters.

Exit criteria:

- External companion implementations do not require forks.
- Third-party adapters can be tested against contract suites.
- The core remains brand-neutral.

## Testing Strategy

### Unit Tests

Required from Phase 1:

- Chunker span correctness.
- Unicode normalization.
- Manifest hash stability.
- Merkle diff behavior.
- Candidate deduplication.
- Score merge determinism.
- Context budget packing.
- Tool schema validation.
- Permission decisions.
- Trace event ordering.

### Integration Tests

Required from MVP:

- File source -> parser -> chunker -> metadata store.
- File source -> QDrant adapter -> dense retrieval.
- Dense + sparse candidate merge.
- Context pack persistence and replay.
- Tool execution with artifact output.
- Agent run with fake model and fake tools.

### Golden Retrieval Tests

Create small corpora with expected answers:

- Exact phrase query.
- Morphological variant query.
- Fuzzy typo query.
- Citation lookup.
- Ambiguous entity query.
- Conflicting source query.
- Recency-sensitive query.

Metrics:

- Recall@k.
- MRR.
- Citation accuracy.
- Unsupported-claim rate.
- Context token waste.
- Duplicate evidence rate.

### Agent Task Tests

Use deterministic fake models first.

Scenarios:

- Generate a spec from project docs.
- Reject unsupported factual claim.
- Ask for approval before risky tool.
- Run explorer subagent and merge its result.
- Recover context from stored artifact instead of long prompt injection.

### Property And Fuzz Tests

Add after baseline:

- Parser and chunker fuzzing.
- Unicode edge cases.
- Path normalization.
- Manifest diff invariants.
- Tool schema validation fuzzing.
- Redaction fuzzing.

### Performance Tests

Measure before optimizing:

- Indexing throughput.
- Chunk count per document.
- QDrant query latency by project size.
- Sparse search latency.
- Context pack build latency.
- Model-independent agent overhead.
- Artifact read slicing latency.

### Failure Injection

Required before stable:

- QDrant unavailable.
- Metadata store unavailable.
- Artifact store read/write failure.
- Embedding provider timeout.
- Model provider timeout.
- Tool timeout.
- Partial indexing failure.
- Background run cancellation.

## Debugging And Observability

### Trace Everything Important

Every run should produce an append-only trace:

- User or event trigger.
- Policy snapshot.
- Retrieval plan.
- Retriever calls.
- Candidate counts and scores.
- Context pack checksum.
- Model calls and provider versions.
- Tool calls and permission decisions.
- Subagent starts/stops.
- Verifier results.
- Final artifacts and decisions.

### Debug Views

The engine should emit data suitable for these views:

- Project index status.
- Source manifest diff.
- Retrieval trace.
- Context pack inspector.
- Tool call timeline.
- Agent run timeline.
- Evaluation report.
- Background job queue.

### Log Levels

- `debug`: retriever internals, candidate scoring, budget decisions.
- `info`: run lifecycle, indexing lifecycle, tool lifecycle.
- `warn`: skipped sources, partial failures, degraded retrieval.
- `error`: failed runs, corrupted manifests, unavailable stores.

### Redaction

Redaction must happen before model-visible summaries and logs intended for user
support. Preserve raw artifacts only under explicit retention and access policy.

## Review Rules

### Architecture Review

Require review when a change:

- Adds a public interface.
- Adds a storage dependency.
- Changes domain models.
- Changes retrieval scoring.
- Changes permission policy.
- Changes trace schemas.
- Adds background execution.

Checklist:

- Is the core still brand-neutral?
- Is the dependency behind an adapter?
- Can behavior be replayed from traces?
- Are source spans and checksums preserved?
- Does this introduce hidden global state?
- Does this create a migration burden?
- Is the failure mode observable?

### Retrieval Review

Checklist:

- Does the change improve a measured metric?
- Does it preserve exact/citation retrieval?
- Are score contributions explainable?
- Are rejected candidates available for debugging when needed?
- Are source filters enforced before model calls?
- Are embedding/chunking versions recorded?

### Agent/Tool Review

Checklist:

- Is the tool schema typed?
- Is the risk level correct?
- Is approval required for side effects?
- Are outputs structured?
- Are long outputs stored as artifacts?
- Is the tool idempotent or clearly marked non-idempotent?
- Is timeout/retry behavior defined?

### Security Review

Checklist:

- Project boundary enforced.
- No cross-project search leakage.
- Ignore rules respected.
- Secrets redacted.
- Tool permissions enforced outside the model.
- Background actions have owner and policy snapshot.
- External network access is explicit.

### NLP/Factuality Review

Checklist:

- Claims have source spans.
- Scientific/legal text preserves citations and versions.
- Morphology/fuzzy matching does not silently change meaning.
- Summaries distinguish evidence from inference.
- Uncertainty is represented instead of hidden.

## Open Decisions

### QDrant Namespace Strategy

Decide between:

- Collection per project.
- Shared collection with `project_id` filters.
- Collection per tenant plus `project_id` filters.

Start with an abstraction and benchmark before committing.

### Metadata Store

Options:

- In-memory for tests.
- SQLite for local single-node deployments.
- Postgres for MVP and multi-user systems.

Recommended path: memory first, Postgres by MVP.

### Sparse Search

Options:

- Simple in-memory inverted index.
- Postgres full-text search.
- QDrant sparse vectors.
- Dedicated sparse n-gram index.

Recommended path: simple deterministic sparse retriever first, then measure.

### Morphology

Options:

- Pure Go minimal normalization.
- External dictionaries.
- cgo/Rust adapters.
- Language-specific plugins.

Recommended path: adapter interface first, simple English/Russian-friendly
normalization baseline, then language-specific plugins.

### Web Crawling

Web crawling can create legal, reliability, and abuse risks. Start with explicit
URL capture and strict limits. Defer broad crawling.

## Immediate Next Steps

1. Add architecture decision records under `.project/decisions/`.
2. Create the internal package skeleton without external dependencies.
3. Implement domain models and interfaces only.
4. Add deterministic unit tests for manifest, chunking, context pack, and tool
   schema behavior.
5. Implement local artifact store and in-memory metadata store.
6. Implement one source adapter, one parser, one chunker, and one retriever.
7. Add QDrant adapter behind an interface after the in-memory path is tested.
8. Add a fake model provider and fake tool executor for agent-run tests.
9. Build the first golden retrieval dataset before adding more algorithms.
