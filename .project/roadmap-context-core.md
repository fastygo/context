# Context Core Engineering Roadmap

Status: planning baseline  
Scope: `github.com/fastygo/context` as a brand-neutral context-management,
retrieval, indexing, and agent orchestration engine.

This roadmap treats the module as infrastructure for project-scoped corpora,
scientific/legal/linguistic text retrieval, browser assistants, background
agents, and downstream companions. Product-specific personas, names, workflows,
and UI surfaces belong in consumer applications or adapters, not in the reusable
core. Scenario-specific systems such as message atlases, enterprise catalogs,
project timelines, CRM assistants, methodology packs, and companion products
must be built as plugins, adapters, or separate repositories over the neutral
engine.

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
file/artifact source adapter, one PostgreSQL-backed metadata store, one pgvector
`VectorStore` adapter, one exact/sparse retrieval baseline, one context-pack
builder, one tool registry, one agent run trace, and one deterministic verifier.
QDrant, Turbopuffer, `context-sparse`, horizontal scale, copy-on-write index
reuse, background agents, billing, multi-tenant security, and browser UI should
be designed for, but not prematurely implemented.

## Product-Agnostic Design Boundaries

### Core Responsibilities

- Project-scoped source registration and artifact tracking.
- Deterministic source parsing, chunking, enrichment, and manifest generation.
- Language-neutral lexical and morphology contracts for tokens, lemmas, lexemes,
  wordforms, feature bundles, and analyzer versions.
- Language-neutral lexicographic contracts for senses, concepts, attestations,
  variants, registers, regions, time periods, and lexicon sources.
- Hybrid retrieval over dense vectors, sparse/keyword indexes, exact matching,
  metadata filters, recency, graph edges, and tool outputs.
- Focus policies that constrain retrieval scope, evidence budgets, source
  preferences, freshness, and verification strictness for a task.
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
- Scenario-specific domain products such as messaging-platform catalogs,
  timeline/Gantt tools, CRM workflows, calendar assistants, or methodology
  runtimes.
- Arbitrary remote code execution without a consumer-provided sandbox policy.
- Billing and subscription enforcement in the library core.
- A hard dependency on any one LLM provider, embedding provider, vector store,
  crawler, database, or web framework.
- Language-specific dictionaries, grammar rules, lexicons, morphology engines,
  or query-expansion heuristics inside the neutral core.
- Dictionary, thesaurus, historical lexicon, slang, regional, or community
  vocabulary content as built-in core data.

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
- Lexeme, lemma, wordform, token-span, and morphology-feature contracts that are
  stable across languages.
- Explicit ambiguity handling: analyzers may return multiple candidate analyses,
  and retrieval traces must show which candidates were used or rejected.
- Sparse/BM25-style retrieval for lexical precision.
- Exact phrase and source-span lookup for citations and legal/scientific text.
- N-gram, trigram, or sparse n-gram indexes when measured latency requires them.
- Damerau-Levenshtein, Jaro-Winkler, and phonetic hooks for typo and name
  matching.
- Text metrics such as entropy, density, repetition, and stop-word ratio where
  they help analysis or quality gates.

### Multilingual Language Adapter Boundary

The core is multilingual by contract, not by embedding every language inside the
repository. Language-specific complexity belongs in versioned adapters.

```text
fastygo/context
  -> multilingual contracts
  -> provenance, spans, snapshots, ContextPack, retrieval orchestration
  -> no language-specific dictionaries or grammar rules

context-lang-*
  -> language-specific analyzers
  -> lexeme paradigms
  -> morphology generation
  -> query expansion
  -> language evaluation fixtures
```

Core contracts should be compatible with Universal Dependencies and UniMorph as
cross-language feature schemes. Adapter-specific raw tags such as OpenCorpora
grammemes are allowed only as adapter-owned metadata carried through a neutral
`MorphFeatureSet`.

Initial language adapters should be planned outside the core for Russian,
English, German, Spanish, French, Hindi, and broader Indic language families.
Each adapter must preserve source offsets, analyzer versions, dictionary
versions, ambiguity candidates, and query-expansion provenance.

### Lexicographic Context Layer

Lexeme and morphology answer "which form." Context also needs to answer "which
meaning, where, when, in which register, and according to which evidence." This
is essential for dictionaries, historical corpora, regional vocabularies, slang,
scientific terminology, legal terms, and community-specific lexicons.

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

Neutral contracts should distinguish:

- `Sense`: a specific meaning of a lexeme.
- `Concept`: a language-independent or domain concept connected to labels and
  senses.
- `Attestation`: a witnessed use in a source with quote, span, date, region,
  register, source authority, and confidence.
- `Variant`: orthographic, historical, regional, slang, spelling, or script
  variant.
- `MultiwordExpression`: lexical unit spanning multiple tokens or syntactic
  words.
- `Register`: usage layer such as formal, scientific, legal, slang, archaic, or
  dialectal.
- `DialectRegion` and `TimePeriod`: contextual filters for regional and
  diachronic retrieval.
- `LexiconSource`: dictionary, corpus, thesaurus, glossary, authority list, or
  community vocabulary source.

TEI-style dictionary entries and SKOS/ISO 25964-style thesauri are future
interoperability targets. The core should model their neutral concepts and
provenance, not embed any specific dictionary content or controlled vocabulary.

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

### Product And Plugin Boundary

Concrete products should be expressed through adapters, tools, policies,
contracts, and companion configuration:

- Messaging archives are source adapters plus graph projections, not core
  package names.
- Calendars, Gantt charts, issue trackers, and project dashboards are event and
  tool adapters over core timelines and traces.
- CRM, catalog, and enterprise-document assistants are downstream products over
  sources, chunks, retrieval plans, and context packs.
- Methodology-specific systems are optional rule, skill, contract, or graph
  plugins.

The core should provide stable generic primitives so these products can exist
without forks or hardcoded assumptions.

### Lab As Downstream UX/DX/DSL Consumer

`Lab` is the current downstream laboratory shell for validating usability, BFF
integration, and DSL ergonomics. It is not a core dependency.

- **UX:** Lab can render fixture or real JSON views for project corpus status,
  evidence snippets, FocusProfile, ContextPack, AgentRun, and trace timelines.
- **DX:** Lab can consume `context-dev` CLI output, health/status responses, and
  proof artifacts to make debugging visible in a browser.
- **DSL:** Lab can later edit and visualize neutral contracts such as
  FocusProfile, RetrievalPlan, ContextPackTemplate, ToolPolicy, SourceAdapter
  configuration, and AgentRunPolicy.

The dependency direction is always:

```text
Lab -> Context CLI/API/SDK contracts
Context -> no Lab dependency
```

When Lab exposes a useful requirement, promote it into Context only as a neutral
contract, DTO, trace field, API endpoint, or ADR. Do not add Lab-specific package
names or UI concepts to the core.

### Focus Control

Large corpora need deterministic focusing before model calls. A `FocusProfile`
or `FocusPolicy` should describe the current task lens:

- task objective;
- active project scope;
- preferred and forbidden source types;
- required source trust level;
- freshness window;
- exactness level;
- citation strictness;
- context budget;
- allowed tools and subagents;
- negative assumptions or known irrelevant areas.

Focus control is not a product feature. It is a neutral runtime policy that
prevents context overflow, topic drift, and unnecessary model usage.

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
   PostgreSQL/pgvector, QDrant, Turbopuffer, object storage, local filesystem,
   specific LLMs, web crawlers, and product integrations must be replaceable
   behind interfaces.

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
    language.go             # language/script identifiers and adapter metadata
    normalize.go            # neutral Unicode, casing, punctuation contracts
    token.go                # tokens, token spans, lemmas, wordforms
    lexeme.go               # lexeme and lemma contracts, not dictionaries
    morphology.go           # morphology analyzer/generator interfaces
    features.go             # UD/UniMorph-compatible feature-set carrier
    queryexpand.go          # explainable lexical/morphology expansion contract
    registry.go             # adapter discovery without importing language repos
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
    focus.go                # task focus profiles and scope constraints
    budget.go               # token/window/model budget planner
    verifier.go             # evidence validation before generation

  retrieval/dense/
    postgresvector/         # first live pgvector VectorStore adapter
    qdrant/                 # future QDrant VectorStore adapter
    turbopuffer/            # future managed VectorStore adapter
    memory/                 # in-memory test adapter

  retrieval/sparse/
    inverted/               # baseline inverted index
    trigram/                # future exact/fuzzy phrase index

  linguistic/adapters/
    noop/                   # deterministic no-op adapter for tests
    simple/                 # small fixture adapter for PoC contract tests

  lexicon/
    sense.go                # word-sense contract independent from dictionaries
    concept.go              # concept/label/thesaurus-style contract
    attestation.go          # witnessed usage with source span and authority
    variant.go              # orthographic, historical, regional variants
    register.go             # register, dialect/region, time-period metadata
    resource.go             # dictionary/corpus/thesaurus source metadata

  storage/
    tx.go                   # transaction boundary abstractions
    memory/                 # deterministic tests
    postgres/               # durable metadata and pgvector-backed PoC adapter

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
- `token_spans`
- `lemmas`
- `lexeme_refs`
- `wordforms`
- `morph_features`
- `morph_analysis_refs`
- `entities`
- `citations`
- `metadata`
- `embedding_version`
- `sparse_version`

### TokenOccurrence

Represents an offset-preserving token occurrence inside a source span. It is
not a dictionary entry.

Required fields:

- `id`
- `project_id`
- `source_id`
- `chunk_id`
- `language`
- `script`
- `surface`
- `normalized`
- `span_start`
- `span_end`
- `tokenizer_version`
- `normalizer_version`

### Lexeme And WordForm

The core stores language-neutral references to lexical objects. Full
language-specific paradigms belong in adapters.

Required fields:

- `lexeme_id`
- `language`
- `lemma`
- `wordform`
- `feature_scheme`: `UD`, `UniMorph`, or adapter-specific.
- `features`
- `raw_feature_scheme`
- `raw_features`
- `adapter_id`
- `dictionary_version`

### MorphAnalysis

Represents one possible analysis of a token occurrence. Ambiguity is explicit:
one token may have multiple analyses with confidence and selection state.

Required fields:

- `id`
- `token_id`
- `language`
- `lemma`
- `lexeme_id`
- `part_of_speech`
- `features`
- `raw_features`
- `confidence`
- `selected`
- `selection_reason`
- `analyzer_id`
- `analyzer_version`
- `dictionary_version`

### QueryExpansion

Represents explainable lexical or morphology-driven query expansion. It must
never silently change the user's meaning.

Required fields:

- `id`
- `query_id`
- `language`
- `original_term`
- `expanded_term`
- `expansion_type`: `lemma`, `wordform`, `compound`, `accent`, `fuzzy`,
  `synonym`, `transliteration`
- `features`
- `confidence`
- `reason`
- `adapter_id`
- `adapter_version`

### Sense

Represents one meaning of a lexeme. It must not be collapsed into the lemma.

Required fields:

- `id`
- `project_id`
- `lexeme_id`
- `language`
- `definition`
- `concept_id`
- `register`
- `region`
- `time_period`
- `source_id`
- `confidence`
- `metadata`

### Concept

Represents a language-independent or domain-specific concept that may have many
labels, senses, translations, variants, or thesaurus relations.

Required fields:

- `id`
- `project_id`
- `preferred_label`
- `labels`
- `source_id`
- `concept_scheme`
- `broader_concepts`
- `narrower_concepts`
- `related_concepts`
- `exact_matches`
- `close_matches`
- `metadata`

### Attestation

Represents witnessed usage of a token, wordform, lexeme, sense, or concept in a
source. Attestations are the evidence layer for historical dictionaries,
regional lexicons, slang, and scientific/legal terminology.

Required fields:

- `id`
- `project_id`
- `source_id`
- `chunk_id`
- `span_start`
- `span_end`
- `quote`
- `language`
- `lexeme_id`
- `sense_id`
- `concept_id`
- `attested_at`
- `region`
- `register`
- `source_authority`
- `confidence`
- `metadata`

### Variant

Represents a non-canonical form that is meaningful for retrieval, history,
regional usage, spelling, transliteration, script, slang, or orthographic
policy.

Required fields:

- `id`
- `project_id`
- `canonical_ref`
- `variant`
- `variant_type`: `orthographic`, `historical`, `regional`, `slang`,
  `spelling`, `script`, `transliteration`
- `language`
- `script`
- `region`
- `time_period`
- `source_id`
- `confidence`

### MultiwordExpression

Represents a lexical unit that spans multiple tokens or syntactic words.

Required fields:

- `id`
- `project_id`
- `surface`
- `normalized`
- `language`
- `token_ids`
- `span_start`
- `span_end`
- `lexeme_id`
- `sense_id`
- `expression_type`
- `analyzer_version`
- `confidence`

### Register, DialectRegion, TimePeriod, And LexiconSource

These records describe contextual constraints for lexical evidence:

- `Register`: formal, scientific, legal, slang, archaic, dialectal, colloquial,
  technical, or domain-specific usage layer.
- `DialectRegion`: geographic, cultural, community, or ethnolinguistic usage
  boundary.
- `TimePeriod`: date range, era, edition period, or historical orthography
  policy.
- `LexiconSource`: dictionary, corpus, thesaurus, glossary, authority list,
  regional vocabulary, community vocabulary, or historical source collection.

They should be modeled as metadata-rich references rather than hardcoded enums
when the vocabulary is product-, region-, or corpus-specific.

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

### FocusProfile

Represents the task-specific focus lens applied before retrieval and context
packing.

Required fields:

- `id`
- `project_id`
- `task_id`
- `objective`
- `scope`
- `preferred_source_types`
- `forbidden_source_types`
- `required_trust_level`
- `freshness_window`
- `exactness_level`
- `citation_strictness`
- `context_budget`
- `allowed_tools`
- `allowed_subagents`
- `negative_assumptions`

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
  -> Tokenizer
  -> LanguageAdapter
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
- Lexicographic resources: entry/sense/attestation-aware chunks preserving
  headwords, definitions, usage examples, source citations, register, region,
  and time-period metadata.
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
- Linguistic version fields include tokenizer, normalizer, morphology adapter,
  analyzer, generator, dictionary, and feature-schema versions.
- Diff mode: only reparse/rechunk/reindex changed branches.

Future:

- Simhash over manifest leaves to find near-duplicate corpora.
- Copy-on-write index reuse across project templates or team workspaces.
- Content proofs to prevent shared-index leakage.

### VectorStore Strategy

Use `VectorStore` as the domain-facing port. PostgreSQL/pgvector, QDrant,
Turbopuffer, and any future managed vector backend are adapters, not the domain
model.

Recommended first PoC mapping:

- PostgreSQL stores metadata, chunks, traces, and pgvector embedding rows in one
  local Docker service.
- Every vector row includes `project_id`, `snapshot_id`, `source_id`,
  `artifact_id`, `chunk_id`, `context_ref`, `span_start`, `span_end`,
  `embedding_version`, `chunker_version`, `language`, `source_type`, and
  `trust_level`.
- `VectorNamespace` is still required, but it describes a logical index
  boundary rather than a QDrant collection.
- Dense vectors provide semantic recall; exact and sparse search remain separate
  retrieval paths.

Future adapters:

- **QDrant:** use when vector volume, filter latency, quantization, memory
  tuning, or independent vector scaling outgrow pgvector.
- **Turbopuffer:** use when a managed serverless vector backend is preferred and
  adapter tests prove project/snapshot filters, replayability, and provenance
  constraints are preserved.
- **Other backends:** allowed only through the same `VectorStore` contract and
  capability model.

Decision to revisit:

- pgvector keeps the PoC simple and debuggable, but it should not leak into
  domain models.
- Collection-per-project, namespace-per-project, and shared-index payload
  filters are backend-specific implementation choices.
- Build adapter capability checks for dimension, metric, filter semantics,
  namespace behavior, payload limits, and managed-service consistency guarantees.

## Retrieval Architecture

### Retrieval Planner

The planner maps a task to one or more retrievers.

Inputs:

- User intent or event trigger.
- Project policy.
- Focus profile.
- Available indexes.
- Current active artifacts.
- Model budget.
- Risk level.
- Required confidence.

Outputs:

- Dense semantic searches.
- Sparse keyword searches.
- Exact phrase searches.
- Lemma/wordform searches.
- Morphology-expanded searches.
- Sense and concept searches.
- Attestation, register, region, and time-period filtered searches.
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
- Preserve lexical/morphology expansion reasons for every contributing score.
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
- Preserve original surface text even when retrieval matched a lemma, generated
  wordform, compound part, accent-folded form, or fuzzy variant.
- Distinguish original source text, lexical analysis, sense claim, concept
  mapping, and attestation evidence.
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
- Token-span preservation for snippets, citations, highlighting, and replay.
- Lemma, lexeme, wordform, and morphology-feature contracts.
- Ambiguity-aware morphology analysis with explicit selected/rejected analyses.
- Stop-word handling.
- Lemmatization/morphology adapter interface.
- Morphology generation adapter interface for controlled wordform expansion.
- Explainable query expansion with `lemma`, `wordform`, `compound`, `accent`,
  `fuzzy`, and `transliteration` reasons.
- Sense, concept, variant, register, region, time-period, and attestation
  references as optional retrieval constraints.
- Damerau-Levenshtein for typo-tolerant matching.
- Jaro-Winkler for prefix-sensitive fuzzy matching.
- N-gram fingerprints for candidate narrowing.
- Phonetic hooks such as Metaphone/Double Metaphone where relevant.
- Text metrics: water ratio, keyword density, entropy, repeated-term patterns.

Language-specific behavior must be implemented outside the core:

- Russian: rich inflection, aspect, animacy, `ё/е` policy, OpenCorpora-like raw
  tags, and ambiguity resolution.
- German: case/gender agreement, separable verbs, compound splitting, and
  capitalization-sensitive nouns.
- Spanish: verb conjugation, gender/number agreement, accent policy, and clitic
  handling.
- French: elision, contractions, accents, agreement, and silent morphology.
- Hindi: Devanagari normalization, postpositions, oblique case, gender/number,
  and compound verbs.
- Indic family: script-specific normalization, transliteration boundaries,
  segmentation, and adapter-specific dictionary resources.

### Lexicographic And Corpus Layer

The lexicographic layer is evidence management, not just NLP preprocessing.

Design for:

- dictionary entries and senses;
- concept schemes and controlled vocabularies;
- thesaurus relations and mappings;
- historical spelling and orthographic variants;
- region, dialect, community, and ethnogroup labels;
- slang and register-specific vocabularies;
- attestations with source quotes, spans, dates, and authority metadata;
- TEI dictionary import/export as a future adapter path;
- SKOS/ISO 25964 interoperability for thesauri and controlled vocabularies.

Rules:

- Do not collapse senses into lemmas.
- Do not collapse concepts into labels.
- Do not replace original source text with normalized text.
- Do not treat an unattested generated wordform as evidence.
- Every lexicographic claim used in a `ContextPack` must be traceable to a
  source span, authority source, or explicitly marked inference.

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
- PostgreSQL compose path with pgvector enabled for metadata persistence,
  vector search, and end-to-end CLI proof checks.
- PostgreSQL full-text search or fake sparse adapters for first lexical tests.
- SQLite remains optional for a later single-node/offline adapter.
- QDrant, Turbopuffer, and `context-sparse` remain optional adapters after the
  pgvector PoC proves the contracts.

### MVP

- Postgres for metadata, runs, decisions, permissions, eval results, and
  pgvector-backed embeddings unless measurements justify a separate VectorStore.
- Optional QDrant or Turbopuffer adapter behind `VectorStore`.
- Optional `context-sparse` adapter for lexical retrieval when PostgreSQL
  full-text search is no longer enough.
- Local or S3-compatible object storage for artifacts.
- Background job queue abstraction.

### Stable

- Postgres with migrations and backups.
- Specialized VectorStore backend if needed: QDrant cluster, Turbopuffer, or
  another managed store.
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
- Namespace-per-project may be operationally costly without testing real backend
  behavior at high namespace counts.

## Roadmap

### Phase 0: Architecture Baseline

Goal: lock the boundaries before writing broad implementation code.

Deliverables:

- This roadmap accepted as planning baseline.
- Architecture decision records for:
  - VectorStore backend and namespace strategy.
  - Metadata store choice.
  - Artifact store choice.
  - Multilingual linguistic contracts and language adapter boundary.
  - Sense, concept, attestation, and lexicon-resource boundary.
  - First supported source types.
  - First supported model providers.
- Package layout skeleton under `internal`.
- Core domain models for project, source, artifact, chunk, context pack,
  agent run, tool call, evaluation, token occurrences, lexeme references,
  wordforms, morphology analyses, senses, concepts, attestations, variants, and
  lexicon sources.
- Interface-only boundaries for models, storage, retrieval, indexing, tools,
  language adapters, and tracing.

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
- Neutral token-span capture and simple/fake language analyzer contracts.
- Neutral sense/concept/attestation fixtures that preserve source spans and do
  not require a real dictionary import.
- Local artifact store.
- In-memory metadata store.
- PostgreSQL/pgvector dense vector adapter behind `VectorStore`.
- Simple sparse keyword retriever or PostgreSQL full-text baseline.
- Retrieval planner with dense + sparse + exact path.
- Explainable query-expansion trace shape, even if the PoC uses only fake/simple
  adapters.
- Baseline focus profile that constrains source types and context budget.
- Context pack builder.
- One LLM adapter interface with a fake deterministic test model.
- Tool registry with one read-only example tool.
- Agent run trace.
- Verifier requiring source-backed evidence.
- Machine-readable CLI JSON that downstream UX shells can consume as fixtures.

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
- Lexical/morphology metadata is versioned and traceable without requiring a
  full Russian, German, Spanish, French, Hindi, or Indic adapter in core.
- Sense, concept, and attestation metadata can be represented in proof JSON
  without importing dictionary or thesaurus adapters.
- A downstream lab shell can render proof artifacts without importing Context
  internals.

### Phase 2: MVP

Goal: make the engine usable by a real downstream product.

Scope:

- Durable Postgres metadata adapter.
- Artifact store abstraction with local and S3-compatible implementations.
- Versioned embeddings through `VectorStore`; evaluate QDrant or Turbopuffer if
  pgvector measurements no longer satisfy scale, filtering, or latency goals.
- Ignore patterns for source and indexing exclusion.
- Rule and skill config loading.
- Focus profile inspection and persistence.
- Tool permission policy.
- Background job abstraction.
- Explorer and verifier subagents.
- Web capture adapter with strict crawl limits.
- Eval harness with golden retrieval datasets.
- Language adapter contract-test harness with fixture analyzers.
- Lexicon resource contract-test harness with fixture dictionaries,
  attestations, and concept schemes.
- Context inspector output format for browser UI consumers.
- Thin HTTP/gRPC service contract or SDK-client contract suitable for Lab/BFF
  integration, after CLI contracts stabilize.

Exit criteria:

- Multiple projects can coexist safely.
- Indexing and retrieval are deterministic under test fixtures.
- Language adapter outputs are deterministic under contract fixtures.
- Sense/concept/attestation outputs are deterministic under contract fixtures.
- A downstream product can register tools without modifying core code.
- Model provider can be swapped through config.
- The system can explain why a context pack was built.
- Verifier catches unsupported factual claims in tests.
- A BFF consumer can call Context through a service/client boundary without
  depending on internal packages.

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
- Language adapter SDK/testkit for `context-lang-*` repositories.
- Tool SDK.
- Plugin boundary guide for scenario-specific products and methodologies.
- DSL schema guide for FocusProfile, RetrievalPlan, ContextPackTemplate,
  ToolPolicy, SourceAdapterConfig, and AgentRunPolicy.
- Companion configuration format.
- Self-hosting guide.
- Example projects with neutral names.
- Reference eval datasets.
- Compatibility tests for third-party adapters.
- Compatibility tests for language adapters across `LanguageCode`,
  `MorphFeatureSet`, `TokenSpan`, and query-expansion contracts.
- Compatibility tests for lexicon resources across `Sense`, `Concept`,
  `Attestation`, `Variant`, `Register`, `DialectRegion`, `TimePeriod`, and
  source-span contracts.

Exit criteria:

- External companion implementations do not require forks.
- Third-party adapters can be tested against contract suites.
- The core remains brand-neutral.

## Testing Strategy

### Unit Tests

Required from Phase 1:

- Chunker span correctness.
- Unicode normalization.
- Token-span offset preservation.
- MorphFeatureSet validation without language-specific enums in core.
- MorphAnalysis ambiguity and selection invariants.
- QueryExpansion reason and confidence validation.
- Sense, Concept, Attestation, Variant, and LexiconSource validation.
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
- File source -> tokenizer -> simple language adapter -> metadata store.
- File source -> VectorStore adapter -> dense retrieval, with pgvector as the
  first live adapter.
- Dense + sparse candidate merge.
- Context pack persistence and replay.
- Tool execution with artifact output.
- Agent run with fake model and fake tools.

### Golden Retrieval Tests

Create small corpora with expected answers:

- Exact phrase query.
- Morphological variant query.
- Lemma-vs-wordform query.
- Ambiguous wordform query.
- Sense disambiguation query.
- Concept label query.
- Attestation date/region/register query.
- Query-expansion false-positive query.
- Fuzzy typo query.
- Citation lookup.
- Ambiguous entity query.
- Conflicting source query.
- Recency-sensitive query.

Metrics:

- Recall@k.
- MRR.
- Citation accuracy.
- Lemma recall@k.
- Inflection coverage.
- Morph expansion precision.
- Sense precision.
- Concept mapping precision.
- Attestation span accuracy.
- False morph expansion rate.
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
- Script detection edge cases.
- Token/span round-trip invariants.
- MorphFeatureSet serialization invariants.
- QueryExpansion does not cross project/source/trust boundaries.
- Sense/concept/attestation filters do not cross project/source/trust
  boundaries.
- Path normalization.
- Manifest diff invariants.
- Tool schema validation fuzzing.
- Redaction fuzzing.

### Performance Tests

Measure before optimizing:

- Indexing throughput.
- Chunk count per document.
- Vector query latency by backend, project size, snapshot size, and filter shape.
- Sparse search latency.
- Context pack build latency.
- Model-independent agent overhead.
- Artifact read slicing latency.

### Failure Injection

Required before stable:

- VectorStore unavailable or degraded.
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
- Language adapter calls and versions when used.
- Query expansions and rejected expansion candidates.
- Sense/concept mappings and rejected mapping candidates.
- Attestation filters and selected usage evidence.
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
- Language/morphology trace.
- Lexicon/sense/attestation trace.
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
- Are lexical/morphology expansions explainable?
- Are sense/concept/attestation mappings explainable?
- Are analyzer, dictionary, and feature-schema versions recorded?
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
- Ambiguous morphology remains explicit instead of being silently collapsed.
- Language-specific rules stay in adapters, not core domain models.
- Token spans remain stable for snippets, citations, and highlighting.
- Senses are not collapsed into lemmas.
- Concepts are not collapsed into labels.
- Attestations include source span, quote, authority, region/register/time
  metadata where available.
- Summaries distinguish evidence from inference.
- Uncertainty is represented instead of hidden.

## Open Decisions

### VectorStore Backend Strategy

Decide between:

- PostgreSQL/pgvector for the first local PoC.
- QDrant for a dedicated vector engine.
- Turbopuffer for a managed/serverless vector backend.
- Collection, namespace, or table per project.
- Shared collection/table with strict `project_id` and `snapshot_id` filters.
- Tenant-level namespace plus project filters.

Recommended path: PostgreSQL/pgvector first, `VectorStore` abstraction always,
QDrant or Turbopuffer only after measurements or deployment constraints justify
another backend.

### Metadata Store

Options:

- In-memory for tests.
- SQLite for local single-node deployments.
- Postgres for the first live PoC and multi-user systems.

Recommended path: memory first, Postgres by PoC.

### Sparse Search

Options:

- PostgreSQL full-text search for the first one-container PoC.
- Tantivy via `context-sparse` Docker sidecar after lexical scale is proven.
- Bleve as test double behind the same HTTP interface.
- QDrant sparse vectors (optional later).

Recommended path: PostgreSQL full-text or fake sparse first; `context-sparse`
only after the pgvector PoC proves retrieval contracts and lexical requirements
justify another service.

### Morphology

Options:

- Pure Go minimal normalization.
- External dictionaries.
- cgo/Rust adapters.
- Language-specific plugins.

Recommended path: multilingual contracts first, simple/no-op adapters for PoC
tests, then language-specific `context-lang-*` repositories with contract tests.
The core must not encode Russian, German, Spanish, French, Hindi, or Indic
grammar as first-class domain enums.

### Lexicon Resources And Attestations

Options:

- Neutral in-core contracts only, with importers as future adapters.
- TEI dictionary import/export adapter for structured dictionaries and
  historical lexicons.
- SKOS/ISO 25964-compatible adapter for thesauri and controlled vocabularies.
- Corpus attestation adapter for witnessed usage in source collections.
- Community/slang/regional lexicon adapters with explicit authority and license
  metadata.

Recommended path: neutral `Sense`, `Concept`, `Attestation`, `Variant`,
`Register`, `DialectRegion`, `TimePeriod`, and `LexiconSource` contracts first.
Implement real TEI/SKOS/dictionary importers only after the PoC proves source
span, trace, and `ContextPack` compatibility.

### Language Adapter Repositories

Planned external repositories:

- `context-lang-ru`
- `context-lang-en`
- `context-lang-de`
- `context-lang-es`
- `context-lang-fr`
- `context-lang-hi`
- `context-lang-indic`
- `context-lang-testkit`

Recommended path: define the adapter contract in core and keep repository
roadmaps under `.project/plugins/`. Add official adapters only after the PoC
proves `TokenSpan`, `MorphFeatureSet`, `MorphAnalysis`, `QueryExpansion`, and
trace compatibility.

### Product Plugin Boundary

Scenario-specific systems must choose one of these integration shapes:

- source adapter;
- parser/chunker/enricher plugin;
- graph projection;
- tool pack;
- rule/skill pack;
- companion configuration;
- downstream product repository.

They should not add scenario names to core package names or domain entities.

### Web Crawling

Web crawling can create legal, reliability, and abuse risks. Start with explicit
URL capture and strict limits. Defer broad crawling.

## Architecture decisions

Normative decisions live under [`.project/decisions/`](decisions/README.md) (14
ADRs as of 2026-06-17). Phase mapping:

- **Domain and no-service baseline:** ADR-0001–0006 define internal-first
  packages, metadata/artifact/model/trace interfaces, deterministic fakes, and
  replayable `AgentRun`/`ContextPack` snapshots.
- **PoC index contracts:** ADR-0007–0011 and ADR-0013–0014 define embedded KV as
  cache/intermediate storage only, hybrid retrieval + manifest, dual Merkle
  (`source_merkle_root`, `chunk_set_hash`), `IndexSnapshot`, `VectorStore`,
  `VectorNamespace`, `SparseIndexRef`, `ContextRef`, `PathAlias`, and storage
  role separation.
- **Multilingual language contracts:** add a follow-up ADR before implementation
  reaches domain model work for token spans, lexeme references, wordforms,
  morphology feature sets, analyzer/generator interfaces, query expansion, and
  language adapter repositories.
- **Lexicographic context contracts:** add a follow-up ADR before implementation
  reaches domain model work for `Sense`, `Concept`, `Attestation`, `Variant`,
  `MultiwordExpression`, `Register`, `DialectRegion`, `TimePeriod`, and
  `LexiconSource`.
- **2026-06 planning correction:** the first live PoC uses PostgreSQL/pgvector as
  the initial `VectorStore` adapter and PostgreSQL full-text or fake sparse
  search for lexical tests. QDrant, Turbopuffer, and `context-sparse` remain
  explicit future adapters. Add a superseding ADR before implementation reaches
  the local server chunk.
- **MVP/local-cloud parity:** ADR-0010 and ADR-0012 still require endpoint-style
  parity across metadata, vector, sparse, and artifact stores. Snapshot
  export/import and local pull are planned after the local CLI proof.
- **Future-layer deferrals:** simhash copy-on-write seeding, cross-user Merkle
  proofs, incremental segment sync, broad crawling, production governance, and
  large-scale distributed workers remain in `future-layer.md`.

Draft research notes stay in `.project/.draft/` and are non-normative.

## Immediate Next Steps

1. ~~Add architecture decision records under `.project/decisions/`.~~ **Done** — see
   [decisions/README.md](decisions/README.md).
2. Close the foundation gate before runtime code: multilingual and
   lexicographic contracts, PoC backend order, deterministic identity/span
   hashing, phase-1 retrieval scoring, `ContextPack` budget behavior, and
   snapshot commit failure semantics. Do not implement graph traversal,
   production KV caches, QDrant/Turbopuffer, `context-sparse`, crawlers, or
   distributed workers at this gate.
3. Create the internal package skeleton without external dependencies.
4. Implement domain models and interfaces only, including `IndexSnapshot`,
   `ManifestNode`, `ContextRef`, `PathAlias`, `VectorNamespace`,
   `SparseIndexRef`, `PolicySnapshot`, `LanguageCode`, `ScriptCode`,
   `TokenOccurrence`, `MorphFeatureSet`, `MorphAnalysis`, `QueryExpansion`,
   `Sense`, `Concept`, `Attestation`, `Variant`, `MultiwordExpression`,
   `Register`, `DialectRegion`, `TimePeriod`, and `LexiconSource` from the ADR
   set.
5. Add deterministic unit tests for manifest, chunking, context pack, and tool
   schema behavior.
6. Implement local artifact store and in-memory metadata store.
7. Implement one source adapter, one parser, one chunker, dual Merkle manifest,
   and `IndexSnapshot` commit model.
8. Implement retrieval interfaces, exact lookup, fake sparse client, and hybrid
   candidate merge without requiring live services in unit tests.
9. Add PostgreSQL/pgvector as the first live `VectorStore` adapter after local
   service contracts exist; keep QDrant, Turbopuffer, and `context-sparse` as
   measured later adapters behind the same interfaces.
10. Add a fake model provider and fake tool executor for agent-run tests.
11. Build the first golden retrieval dataset, including lexical/morphology
    and sense/concept/attestation fixture cases, before adding more algorithms.
