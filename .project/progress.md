# Context Core Progress

Status: active planning tracker  
Scope: implementation path from repository baseline to hypothesis validation and
proof of concept for a scalable project-scoped context engine.

This file is intentionally written as copy-paste Plan chunks. Each chunk should
fit one agent planning session and stay small enough for controlled execution.

The `.project` folder is intentionally self-contained with this tracker,
`roadmap-context-core.md`, `future-layer.md`, and deferred plugin roadmaps under
`.project/plugins/`. Future agents should not need deleted brainstorming
references. If more durable decisions are created, they should live under
`.project/decisions/` and be referenced by the relevant chunk.

## Operating Rules

- Keep the core brand-neutral: no product, mascot, or companion identity in
  core packages, comments, examples, or public APIs.
- Keep concrete scenarios as adapters/plugins/downstream products. Do not move
  message catalogs, timelines, CRM flows, calendars, dashboards, or methodology
  runtimes into neutral core packages.
- Prefer `internal` packages until interfaces prove stable.
- Start with deterministic behavior and in-memory/local adapters before adding
  infrastructure.
- Add external dependencies only when the current chunk needs them.
- Keep the core multilingual by contract: language-specific dictionaries,
  grammar rules, morphology engines, and query expansion must live in language
  adapters/plugins, not core domain models.
- Every runtime behavior that can affect a model or tool decision must become
  traceable.
- Do not build a chat app. Build the context engine that chat, browser
  assistants, background agents, and downstream products can use.
- Treat `Lab` as the first downstream UX/DX/DSL laboratory shell. `Lab` may
  consume CLI JSON, service APIs, SDKs, and exported traces, but `Context` must
  not import or depend on `Lab`.
- For each implementation chunk, update this file with completion notes only
  after tests or manual verification have run.

## Phase Target

The current target is hypothesis validation:

```text
local project corpus
  -> deterministic indexing
  -> PostgreSQL + pgvector-backed metadata/vector search path
  -> real CLI ingestion and retrieval
  -> context pack creation
  -> fake model/tool agent run
  -> source-backed verification trace
```

The proof is not a polished product. The proof is a working CLI loop that shows
the architecture can index project sources, retrieve relevant evidence, build a
context pack, execute a typed tool/model step, and replay/debug what happened.

## UX / DX / DSL Consumer Track

`Lab` is the practical downstream workspace for checking whether the core is
usable by humans and product BFFs. Keep this as a consumer track, not a core
dependency.

| Stage | Context progress | Lab responsibility | Boundary |
| --- | --- | --- | --- |
| UX fixtures | Chunks 02-05 | Mock project corpus, search results, evidence snippets, FocusProfile, ContextPack, trace screens | Fixtures only; no Context import |
| DX CLI bridge | Chunk 08 | Read `context-dev` JSON output and render it in a browser shell | Lab consumes CLI artifacts; Context stays CLI-first |
| Local stack dashboard | Chunks 09-12 | Show PostgreSQL/pgvector, active snapshot, source/chunk/vector counts, optional vector backend capability, and trace status | Lab reads health/status contracts |
| BFF/API consumer | After CLI proof | Call a `context-core` HTTP/gRPC service through a thin client | Service contract before Go SDK |
| DSL workbench | After ContextPack/FocusProfile stabilize | Edit/visualize FocusProfile, RetrievalPlan, ContextPackTemplate, ToolPolicy, AgentRunPolicy | DSL objects belong to Context contracts |

Do not add Lab-specific package names, screens, widgets, or workflows to
`Context`. Any UX requirement discovered in Lab should become a neutral DTO,
trace field, API contract, or ADR only when it benefits more than that one
consumer.

## Plan Chunk 01: Architecture Baseline And Decisions

Copy-paste prompt:

```text
Work in @Context only. Read README.md, .project/roadmap-context-core.md, and
.project/progress.md. Do not rely on deleted brainstorming references. Create
the initial architecture decision records needed before implementation, without
adding runtime code.

Plan and then implement:
1. Create .project/decisions/ if missing.
2. Add ADR for package boundary: internal-first, pkg/contextkit later.
3. Add ADR for metadata store path: memory first, PostgreSQL by PoC.
4. Add ADR for artifact store path: local filesystem first, object storage later.
5. Add ADR for `VectorStore` abstraction: hide pgvector, QDrant, Turbopuffer,
   and collection/namespace strategy behind a stable port.
6. Add ADR for sparse search path: deterministic/fake or PostgreSQL full-text
   first, measured `context-sparse` replacement later.
7. Add ADR for multilingual linguistic contracts and language adapter boundary:
   token spans, lexeme references, wordforms, morphology features, analyzer
   versions, query expansion, and `context-lang-*` repositories.
8. Add ADR for model adapters: fake deterministic provider first, real providers
   behind interface later.
9. Add ADR for trace/event model: append-only events and replayability.
10. Keep language brand-neutral and compatible with the MIT library boundary.
11. Update .project/progress.md completion notes for this chunk.
12. Run a read-only sanity check of changed markdown and report blockers only if
   they block the next chunk.

Acceptance criteria:
- ADR files exist and are concise.
- No product-specific companion naming is introduced.
- The next implementation chunk can start from stable decisions.
```

Status: **completed** (2026-06-17)

### Completion notes

- Created `.project/decisions/` with 14 accepted ADRs (see
  [decisions/README.md](decisions/README.md)).
- **Domain/interface baseline for Plan Chunks 02–04:** ADR-0001–0006 define
  internal-first packages, metadata/artifact store interfaces, `VectorNamespace`,
  fake model adapters, append-only `AgentRun`/`ContextPack` replay, and the first
  no-service test path.
- **Index architecture for Plan Chunks 04, 09, 10, and 12:** ADR-0007–0014 make
  the hybrid index shape normative: `IndexSnapshot` manifest, dual Merkle
  (`source_merkle_root` + `chunk_set_hash`), `VectorStore`/`VectorNamespace`,
  `SparseIndexRef`, `ContextRef`/`PathAlias`, storage role separation, and
  local/cloud service parity.
- **Needs superseding ADR before Chunk 02 implementation:** multilingual
  linguistic contracts and language adapter boundaries must be recorded so the
  core can model `TokenOccurrence`, `LexemeID`, `Lemma`, `WordForm`,
  `MorphFeatureSet`, `MorphAnalysis`, and `QueryExpansion` without importing
  Russian, German, Spanish, French, Hindi, or Indic implementations.
- **2026-06 planning correction:** the first PoC narrows the live dense-vector
  backend to PostgreSQL + pgvector behind `VectorStore`. QDrant and Turbopuffer
  remain explicit later adapters, not first-stack requirements. The existing
  QDrant-first ADR wording should be superseded by a follow-up ADR before
  implementation reaches Chunk 09.
- **Sparse path correction:** unit tests may use fake/Bleve-style doubles and
  the first live PoC may use PostgreSQL full-text search or a minimal local
  sparse adapter. `context-sparse` remains a measured replacement when lexical
  scale or morphology requirements justify another service.
- **Future-layer deferrals:** simhash copy-on-write seeding, cross-user Merkle
  proofs, incremental segment sync, production multi-tenant governance, and
  broad web crawling stay in `future-layer.md` until after the CLI proof works.
- Background drafts remain non-normative under `.project/.draft/`.

## Plan Chunk 02: Internal Package Skeleton And Domain Models

Copy-paste prompt:

```text
Work in @Context only. Read README.md, .project/roadmap-context-core.md,
.project/progress.md, and .project/decisions/*.md. Plan a minimal internal
package skeleton for the context core and implement only domain models plus
interfaces needed for tests. Do not add external services yet.

Plan and then implement:
1. Create internal package folders for corpus, artifacts, indexing, retrieval,
   tools, agentruntime, models, policy, tracing, storage, and evals.
2. Define core domain structs: Project, Source, Artifact, Chunk, SourceRef,
   EvidenceItem, FocusProfile, ContextPack, AgentRun, ToolCall, Evaluation.
3. Define indexing/sync structs from ADRs: IndexSnapshot, ManifestNode,
   ChunkAlias, ContextRef, PathAlias, VectorNamespace, SparseIndexRef,
   PolicySnapshot, ModelCall.
4. Define neutral linguistic structs: LanguageCode, ScriptCode,
   TokenOccurrence, LexemeID, Lemma, WordForm, MorphFeatureSet, MorphAnalysis,
   QueryExpansion, AnalyzerVersion, DictionaryVersion.
5. Use explicit IDs as typed strings or small value types.
6. Add validation methods only where invariants are obvious.
7. Define store interfaces but provide no durable database adapter yet.
8. Define model, embedding, reranker, morphology analyzer/generator, lexical
   normalizer, and query expander interfaces without provider or language
   dependencies.
9. Define tool registry schema types without executing tools yet.
10. Define trace event type and recorder interface, including fields for
   analyzer, dictionary, feature-schema, and query-expansion versions.
11. Add unit tests for basic invariants and zero-value rejection.
12. Run go test ./... and update .project/progress.md completion notes.

Acceptance criteria:
- go test ./... passes.
- Domain types compile without external infrastructure.
- Linguistic contracts compile without importing language-specific adapters.
- No implementation package imports downstream products.
```

Status: pending

## Plan Chunk 03: Local Artifact Store And In-Memory Metadata

Copy-paste prompt:

```text
Work in @Context only. Read current internal packages and roadmap/progress docs.
Implement local development storage needed for a real CLI PoC, but keep it
replaceable.

Plan and then implement:
1. Add local filesystem ArtifactStore adapter under internal/artifacts/localfs.
2. Add in-memory metadata store under internal/storage/memory.
3. Support create/read/list for projects, sources, artifacts, chunks, context
   packs, agent runs, tool calls, and trace events as needed.
4. Preserve checksums and media types for stored artifacts.
5. Ensure paths are project-scoped and cannot escape the configured root.
6. Add tests for path traversal rejection.
7. Add tests for artifact checksum verification.
8. Add tests for in-memory store deterministic ordering where needed.
9. Add simple error types for not found, conflict, validation, and permission.
10. Avoid Postgres until the in-memory path proves the contracts.
11. Run go test ./...
12. Update progress completion notes.

Acceptance criteria:
- Local artifacts can be persisted and read back.
- Metadata store tests prove the core can run without external services.
- Storage contracts are ready for PostgreSQL adapter later.
```

Status: pending

## Plan Chunk 04: Source Adapter, Parser, Chunker, And Merkle Manifest

Copy-paste prompt:

```text
Work in @Context only. Build the deterministic indexing baseline for local files
and stored artifacts. Read roadmap/progress and current indexing/corpus/storage
packages before planning.

Plan and then implement:
1. Define SourceAdapter, Parser, Chunker, Enricher, and ManifestBuilder
   interfaces if not already present.
2. Implement local file source adapter with ignore-pattern support placeholder.
3. Implement plaintext parser.
4. Implement Markdown parser that preserves heading ancestry at a basic level.
5. Implement paragraph-aware text chunker with span_start/span_end checksums.
6. Implement Markdown section-aware chunker.
7. Implement neutral token-span capture for chunks, preserving original surface
   text and byte/rune offset mapping for snippets and citations.
8. Add no-op/simple language adapter hooks that record tokenizer, normalizer,
   analyzer, dictionary, and feature-schema versions without full morphology.
9. Implement dual Merkle baseline: source tree hash and chunk set hash.
10. Implement manifest diff for added, removed, changed, unchanged sources and
   changed chunks.
11. Implement minimal IndexSnapshot commit model with parser/chunker/embed/morph
   version fields and status values.
12. Add golden tests for stable chunks, stable token spans, stable manifest
   hashes, and snapshot IDs; run go test ./... and update progress notes.

Acceptance criteria:
- Re-running indexing over unchanged files produces stable hashes.
- Editing one file marks only that branch/source as changed.
- Chunk spans, checksums, dual Merkle roots, and IndexSnapshot fields are
  test-covered.
- Token spans remain stable enough for snippets, highlighting, citations, and
  morphology traces.
```

Status: pending

## Plan Chunk 05: Retrieval Contracts, Exact Lookup, Sparse Contract, And VectorStore Port

Copy-paste prompt:

```text
Work in @Context only. Implement retrieval contracts and the deterministic exact
lookup path, then define sparse-search and VectorStore ports without choosing
live infrastructure yet. Do not bind domain models to PostgreSQL, QDrant,
Turbopuffer, or `context-sparse`.

Plan and then implement:
1. Define Retriever interface and Candidate model.
2. Implement exact source/span lookup retriever.
3. Define SparseSearchClient interface with project_id and snapshot_id required
   on every query.
4. Define VectorStore interface for embedding upsert/search with project_id,
   snapshot_id, embedding_version, and vector_namespace required on every call.
5. Define MorphAnalyzer, MorphGenerator, LexicalNormalizer, and QueryExpander
   contracts with no-op/simple implementations for tests only.
6. Add basic Unicode normalization hooks but avoid complex morphology for now.
7. Implement candidate score normalization for exact and sparse results.
8. Implement candidate deduplication by chunk_id/source/span/checksum.
9. Preserve score explanation fields, snapshot_id, analyzer versions, and
   query-expansion reasons.
10. Add fake sparse, fake vector, and fake language adapter clients for
   deterministic unit tests.
11. Add golden tests for exact phrase, keyword, lemma-vs-wordform,
   ambiguous-wordform, typo-negative, query-expansion false positive, and
   citation-like lookup if fixtures exist.
12. Add retrieval trace events for query, expansions, candidates, snapshot, and
   selected results; run go test ./... and update progress completion notes.

Acceptance criteria:
- A local corpus can be searched without vectors through exact lookup and a fake
  sparse client.
- Vector retrieval can be planned without importing any concrete vector backend.
- Morphology-aware retrieval can be planned without importing any language
  adapter repository.
- Candidate explanations are inspectable.
- Exact retrieval remains deterministic and source-backed.
- Sparse and vector backends remain replaceable through interfaces.
- Query expansions are explainable and can be rejected without changing source
  truth.
```

Status: pending

## Plan Chunk 06: Context Pack Builder And Verifier

Copy-paste prompt:

```text
Work in @Context only. Implement ContextPack construction as the central runtime
handoff object. Keep it independent from real LLM providers.

Plan and then implement:
1. Define FocusProfile and RetrievalPlan models with task summary, scope,
   strategies, budgets, and verification requirements.
2. Implement candidate merge and ranking pipeline.
3. Implement ContextPackBuilder with token/character budget estimates.
4. Include evidence, source refs, rank signals, rejected candidates, and next
   step instructions.
5. Add ContextPack checksum and replay metadata.
6. Implement baseline Verifier that checks each factual evidence item has a
   valid source reference and checksum.
7. Add tests for budget trimming without losing required citations.
8. Add tests for rejected unsupported evidence.
9. Add tests for replaying a context pack from stored IDs.
10. Emit trace events for context pack creation and verification.
11. Run go test ./...
12. Update progress completion notes.

Acceptance criteria:
- ContextPack is inspectable and replayable.
- Unsupported factual evidence is rejected or flagged.
- Budgeting behavior is deterministic under tests.
```

Status: pending

## Plan Chunk 07: Tool Registry, Fake Model, And Agent Run Loop

Copy-paste prompt:

```text
Work in @Context only. Implement the smallest agent runtime loop with typed
tools and a deterministic fake model. This is not a chat UI.

Plan and then implement:
1. Implement ToolRegistry with typed metadata, input/output schema versions,
   risk level, timeout, and permission requirements.
2. Implement fake read-only tool that returns structured output and optional
   artifact references.
3. Implement Policy decision model: allow, ask, deny.
4. Implement fake deterministic LLM provider for tests.
5. Implement AgentRun orchestrator for one task:
   retrieval plan -> context pack -> fake model/tool step -> verification.
6. Persist ToolCall and AgentRun status transitions.
7. Store long tool output as artifact rather than inline trace.
8. Add trace events for tool registration, policy decision, execution, and
   result.
9. Add tests for denied tool calls.
10. Add tests for replaying a completed agent run.
11. Run go test ./...
12. Update progress completion notes.

Acceptance criteria:
- A complete fake agent run can execute locally.
- Tool permissions are enforced outside the model.
- Run trace is enough to debug what happened.
```

Status: pending

## Plan Chunk 08: Developer CLI For Real Local Workflow

Copy-paste prompt:

```text
Work in @Context only. Add a small developer CLI to exercise the real engine
from the terminal. Keep it for development and hypothesis validation, not as a
finished product interface.

Plan and then implement:
1. Create cmd/context-dev.
2. Add command: init-project --root <dir> --data <dir>.
3. Add command: ingest --project <id> --path <dir-or-file>.
4. Add command: search --project <id> --query <text> --mode sparse|exact|hybrid.
5. Add command: context-pack --project <id> --query <text>.
6. Add command: agent-run --project <id> --query <text> using fake model/tool.
7. Add command: trace --run <id>.
8. Use local artifact store and in-memory or file-backed metadata only if ready.
9. Print stable machine-readable JSON for key outputs so Lab can consume it
   without importing Context internals.
10. Add CLI smoke tests where practical.
11. Run go test ./... and manually run at least one CLI command.
12. Update progress completion notes with exact commands used.

Acceptance criteria:
- A developer can ingest sample docs and search them from CLI.
- ContextPack JSON can be inspected without a UI.
- A fake agent run can be launched and traced from CLI.
- Lab can consume the CLI JSON as fixtures without a Go dependency on Context.
```

Status: pending

## Plan Chunk 09: Local Server Environment With PostgreSQL And pgvector

Copy-paste prompt:

```text
Work in @Context only. Prepare real local infrastructure for the hypothesis
validation path. Do not rewrite core contracts unless needed. Prefer Docker
Compose or clear shell scripts if the repository already uses that style.

Plan and then implement:
1. Add local development compose/config for PostgreSQL with the pgvector
   extension enabled.
2. Add .env.example or documented environment variables if needed.
3. Add Makefile or scripts for dev-up, dev-down, dev-reset, dev-logs, dev-ps if
   appropriate.
4. Add health-check documentation for PostgreSQL and pgvector extension checks.
5. Add storage configuration structs for metadata, vector, sparse, and artifact
   stores without hardcoding the vector backend.
6. Keep secrets out of git.
7. Add README or .project note for local server setup commands.
8. Verify containers start locally.
9. Verify PostgreSQL connection.
10. Verify `CREATE EXTENSION IF NOT EXISTS vector` succeeds.
11. Verify vector dimension compatibility for the first embedding model.
12. Run go test ./... and update progress completion notes with exact setup and
   verification commands.

Acceptance criteria:
- A new developer can start the PostgreSQL/pgvector local stack with one Docker
  service.
- Health checks are documented.
- Core still runs tests without services unless integration tests are requested.
```

Status: pending

## Plan Chunk 10: PostgreSQL VectorStore Adapter And Hybrid Retrieval

Copy-paste prompt:

```text
Work in @Context only. Implement the first live VectorStore adapter with
PostgreSQL/pgvector and keep QDrant, Turbopuffer, and `context-sparse` as later
replaceable adapters behind interfaces. Keep tests isolated and skip integration
tests unless PostgreSQL is available.

Plan and then implement:
1. Add PostgreSQL/pgvector dependency or SQL access only if needed.
2. Define VectorStore, VectorNamespace, SparseSearchClient, and HybridRetriever
   interfaces if not already stable.
3. Implement pgvector adapter under internal/retrieval/dense/postgresvector.
4. Require project_id and snapshot_id filters on dense and sparse search.
5. Record embedding_version, chunker_version, morph_version, context_ref, and
   snapshot_id in vector rows/results.
6. Add a backend capability model so QDrant and Turbopuffer can later declare
   filter, namespace, dimension, metric, and managed-service behavior.
7. Add fake embedding provider, fake vector store, and fake sparse client for
   deterministic tests.
8. Add fake/simple language adapter fixtures to prove dense/hybrid retrieval can
   carry language, token-span, analyzer-version, and query-expansion metadata.
9. Add integration tests gated by PostgreSQL environment variables.
10. Add CLI mode that uses dense/hybrid retrieval when PostgreSQL is configured.
11. Add a short adapter backlog for QDrant, Turbopuffer, `context-sparse`, and
   `context-lang-*` language repositories.
12. Run unit tests and, if services are up, integration tests; update progress
   completion notes with commands and results.

Acceptance criteria:
- Dense and sparse retrieval work through interfaces.
- PostgreSQL/pgvector works as the first live VectorStore.
- QDrant, Turbopuffer, and `context-sparse` can be added without changing domain
  models.
- Language adapters can be added without changing domain models or vector
  adapters.
- Integration tests do not fail when services are absent.
```

Status: pending

## Plan Chunk 11: PostgreSQL Metadata Adapter

Copy-paste prompt:

```text
Work in @Context only. Implement PostgreSQL metadata persistence behind the
existing store interfaces. Keep migrations explicit and tests gated.

Plan and then implement:
1. Add PostgreSQL driver and migration approach only if needed.
2. Create schema for projects, sources, artifacts, chunks, index_snapshots,
   manifest_nodes, chunk_aliases, embeddings/vector rows, context packs, agent
   runs, tool calls, evaluations, trace events, token occurrences, morphology
   analyses, and query expansions.
3. Add migration files under an internal or migrations folder.
4. Implement PostgreSQL store adapter behind existing interfaces.
5. Preserve transaction boundaries for indexing and agent run updates.
6. Add indexes for project_id, source_id, chunk_id, snapshot_id, context_ref,
   run_id, language, lexeme_id, analyzer_version, dictionary_version, and
   timestamps where needed.
7. Add integration tests gated by environment variable.
8. Add CLI option to use PostgreSQL metadata store.
9. Verify rollback/reset workflow for local development.
10. Run unit tests and, if services are up, integration tests.
11. Document exact commands.
12. Update progress completion notes.

Acceptance criteria:
- Metadata survives process restart.
- Store interface remains implementation-neutral.
- Linguistic metadata survives process restart without requiring language
  adapter dependencies in the storage adapter.
- Tests can run without PostgreSQL unless integration mode is enabled.
```

Status: pending

## Plan Chunk 12: End-To-End Hypothesis Validation

Copy-paste prompt:

```text
Work in @Context only. Run and harden the first real end-to-end CLI proof using
local infrastructure. The goal is evidence that the architecture works, not a
polished UX.

Plan and then implement/fix only what is needed:
1. Start PostgreSQL/pgvector locally.
2. Initialize a demo project through the CLI.
3. Ingest README.md and .project/*.md as the first project corpus.
4. Build dense pgvector rows and any available exact/sparse indexes for a
   committed IndexSnapshot.
5. Run exact, PostgreSQL full-text or fake sparse, dense, and hybrid search
   queries.
6. Run at least one multilingual fixture query that proves token spans,
   language code, lemma/wordform metadata, and query-expansion trace shape.
7. Generate a ContextPack for a roadmap-related query.
8. Run fake-model agent flow using the ContextPack.
9. Run verifier and inspect trace output.
10. Capture command outputs, JSON fixtures, or summaries in .project/proof/ if
   appropriate so Lab can replay the UX without hitting live services.
11. Fix only blocking bugs discovered by the proof.
12. Record known gaps and next decisions in progress.md, then report whether the
   hypothesis is validated, partially validated, or failed.

Acceptance criteria:
- Real CLI commands prove ingest -> search -> context pack -> agent run -> trace.
- PostgreSQL/pgvector is exercised as the first live metadata/vector stack.
- The proof identifies what must change before adding QDrant, Turbopuffer, or
  `context-sparse` adapters.
- The proof identifies what must change before adding `context-lang-*` language
  adapters.
- The proof produces enough artifacts to debug failure or demonstrate success.
- The proof exports enough neutral JSON for Lab to render project corpus, search,
  ContextPack, FocusProfile, AgentRun, and trace views.
```

Status: pending

## Completion Notes

Use this section after each chunk. Keep notes short and factual.

```text
### YYYY-MM-DD - Chunk NN

Result:
- ...

Verification:
- ...

Follow-up:
- ...
```
