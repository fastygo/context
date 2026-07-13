# Context Core Progress

Status: active planning tracker  
Scope: Phase 3 Reliable Beta path after Phase 2 MVP exit (Chunks 14–20,
HTTP service ADR-0024). Phase 1 PoC archive: Chunks 01–13.

This file is intentionally written as copy-paste Plan chunks. Each chunk should
fit one agent planning session and stay small enough for controlled execution.

The `.project` folder is intentionally self-contained with this tracker,
`roadmap-context-core.md`, `future-layer.md`, and deferred plugin roadmaps under
`.project/plugins/`. Durable decisions live under `.project/decisions/`.

## Operating Rules

- Keep the core brand-neutral: no product, mascot, or companion identity in
  core packages, comments, examples, or public APIs.
- Keep concrete scenarios as adapters/plugins/downstream products.
- Keep source/observation events separate from runtime trace events.
- Do not map `Project` to a person in core contracts.
- Prefer `internal` packages until interfaces prove stable. The first public
consumer surface is HTTP (Chunk 20) plus a thin `pkg/contextkit` client
(Chunk 21); do not export domain ports wholesale.
- Add external dependencies only when the current chunk needs them.
- Language-specific and lexicographic content stay in adapters/resources.
- Every runtime behavior that can affect a model or tool decision must be
  traceable.
- Do not build a chat app. Build the context engine.
- Treat `Lab` as a downstream UX/DX shell only. Context must not import Lab.
- Do not implement `.project/future-layer.md` items unless a chunk is blocked.
- Update this file with completion notes only after verification.

## Phase Target

Phase 1 (Chunks 01–13) and Phase 2 (Chunks 14–20) are **complete**.

Current target is **Phase 3 Reliable Beta** (start):

```text
thin pkg/contextkit HTTP client (no internal import)
  -> operational metrics / eval history (later chunks)
  -> index rebuild/repair tools (later)
  -> multi-tenant isolation design (later ADR)
```

QDrant, Turbopuffer, `context-sparse`, and full `context-lang-*` / TEI adapters
remain deferred ([ADR-0017](decisions/0017-poc-backend-order.md),
[adapters-backlog.md](adapters-backlog.md)). Full multi-tenant auth and
production SDK generation remain deferred.

## Phase 1 Archive (Chunks 01–13)

Completed 2026-06-17 … 2026-07-13. Details lived in prior progress revisions and
ADRs 0001–0023; do not re-implement.

| Chunk | Result |
| --- | --- |
| 01 + Foundation Gate | ADRs, package/storage/index/linguistic/scoring boundaries |
| 02–03 | Domain ports; localfs artifacts; memory metadata |
| 04–05 | Indexing pipeline; exact/sparse/vector ports + fakes |
| 06–08 | ContextPack + verifier; tools/fake model/agent; `context-dev` CLI |
| 08A | Lineage + temporal contracts (ADR-0023) |
| 09–11 | Compose pgvector; VectorStore adapter; Postgres MetadataStore |
| 12 | E2E proof — hypothesis **validated** (`.project/proof/`) |
| 13 | Durable CLI metadata opt-in (`CONTEXT_METADATA_KIND=postgres`) |

Open gaps after Phase 2: semantic/provider Embedder still deferred; full SDK /
OpenAPI generation deferred; multi-tenant auth deferred. Thin HTTP (Chunk 20)
and `pkg/contextkit` client (Chunk 21) shipped.

## UX / DX / DSL Consumer Track

| Stage | Context progress | Lab responsibility | Boundary |
| --- | --- | --- | --- |
| Proof fixtures | Chunks 08–12 | Render `.project/proof/*.json` | No Context import |
| Durable local stack | Chunk 13 | Show postgres-backed project/snapshot/trace | Env-configured CLI |
| Eval / golden UX | Chunk 19 | Display eval reports | Lab consumes JSON reports |
| BFF/API consumer | **Chunk 20 done** | Call HTTP client | ADR-0024 |
| Go client surface | **Chunk 21 done** | Import `pkg/contextkit` only | No `internal/` |
| DSL workbench | After Chunk 21 | Edit FocusProfile / plans / policies | Neutral DTOs only |

## Plan Chunk 14: PostgreSQL FTS SparseSearchClient

Copy-paste prompt:

```text
Work in @Context only. Replace fake term-overlap sparse retrieval with a live
PostgreSQL full-text SparseSearchClient behind the existing port. Read
ADR-0017, ADR-0008, adapters-backlog.md, progress.md, and local-server.md.
Do not add context-sparse/Tantivy. Do not change domain models.

Plan and then implement:
1. Add Postgres FTS schema/migration for chunk text indexed by project_id +
   snapshot_id (and language if already on chunk rows).
2. Implement SparseSearchClient under internal/retrieval/sparse/postgresfts
   (or equivalent), requiring project_id and snapshot_id on every search.
3. Wire ingest PersistIngest / dense path to upsert FTS rows when
   CONTEXT_SPARSE_KIND=postgres_fts (default remains fake/memory for offline).
4. CLI search modes sparse|hybrid|hybrid-dense use FTS when configured.
5. BackendCapabilities: declare what FTS can/cannot filter server-side.
6. Integration tests gated by CONTEXT_PG_DSN; unit tests offline.
7. Record measured lexical limits vs fake sparse in .project/proof/ or
   progress completion notes (gate for later context-sparse).
8. Run go test ./... and update progress completion notes.

Acceptance criteria:
- Sparse/hybrid can use Postgres FTS without Docker-required unit tests.
- Fake sparse remains available when CONTEXT_SPARSE_KIND is unset/memory.
- No Tantivy/QDrant code.
```

Status: **completed** (2026-07-13)

### Completion notes

- Package: `internal/retrieval/sparse/postgresfts` (`Open` / `EnsureSchema` /
  `Upsert` / `Search` / `Capabilities`); table `context_sparse_fts` with
  generated `tsvector('simple')` + GIN.
- Generic `internal/retrieval/sparse.Retriever` maps hits through chunk index
  (language/temporal filters stay client-side).
- CLI: ingest upserts FTS when `CONTEXT_SPARSE_KIND=postgres_fts`; search modes
  `sparse|hybrid|hybrid-dense` report `sparse_backend=postgres_fts`.
- Default sparse remains fake/memory; no Tantivy/QDrant.
- Lexical limits vs fake (gate for later `context-sparse`):
  - FTS uses `simple` config (no stemming/morphology; multilingual tokenization
    only by whitespace/punctuation).
  - Ranking is `ts_rank_cd` + `plainto_tsquery` (no phrase/proximity operator
    surface yet; no BM25-style tunable term weights).
  - Fake term-overlap still useful offline and for fixture determinism; FTS is
    the live lexical path for compose Postgres.
- Tests: offline `go test ./internal/retrieval/sparse/`; gated
  `CONTEXT_PG_DSN=... go test ./internal/retrieval/sparse/postgresfts/ ./internal/devcli/ -run FTS`.

## Plan Chunk 15: Ingest Version Pins And Dense Upsert On Snapshot Commit

Copy-paste prompt:

```text
Work in @Context only. Close the gap where dense vectors are built lazily at
CLI search time and chunk rows lack stable analyzer/embed version pins.
Read ADR-0011, ADR-0018, ADR-0021, progress.md Chunk 10/13 notes.

Plan and then implement:
1. On ingest/snapshot commit, persist embedding_version, chunker_version,
   morph/analyzer_version (and dictionary_version when present) on chunk
   metadata rows.
2. When VectorStore is configured (postgres_pgvector), upsert dense points for
   the new snapshot_id during commit (same embedding_version as config).
3. Keep snapshot failure semantics (ADR-0021): failed dense upsert must not
   leave a Ready snapshot without recorded failure_reason.
4. Search must prefer already-upserted vectors; optional rebuild flag only.
5. Tests: offline fake VectorStore commit path; gated postgres integration.
6. Update progress completion notes.

Acceptance criteria:
- A ready snapshot implies version pins on chunks and dense rows for that
  snapshot when dense is enabled.
- Lazy search-time upsert is no longer the primary path.
```

Status: **completed** (2026-07-13)

### Completion notes

- Chunk metadata pins: `chunker_version`, `embedding_version`, `morph_version`,
  `dictionary_version`, `sparse_version` on `IndexedChunk` / `corpus.Chunk`
  (migration `002_chunk_version_pins.sql`).
- Snapshot annotate: `embed_model_version`, `morph_version`, `dense_enabled`,
  `sparse_enabled`, `vector_namespace` before activate.
- Dense commit: `dense.UpsertEmbedded` on ingest when `CONTEXT_ENABLE_DENSE=1`;
  failure → `status=failed` + `failure_reason=dense_write_failed` (or
  `sparse_write_failed`), prior `active_snapshot_id` retained (ADR-0021).
- Search prefers ingest-committed vectors; `CONTEXT_DENSE_REBUILD=1` or
  legacy `DenseEnabled=false` triggers search-time backfill only.
- Tests: offline pin + `UpsertEmbedded` fake store; gated
  `TestDenseCommitOnIngestIntegration`.

## Plan Chunk 16: Embedder Adapter Beyond Fake-Hash

Copy-paste prompt:

```text
Work in @Context only. Introduce a replaceable Embedder suitable for measurable
dense retrieval while keeping models/fake for unit tests. Read ADR-0005,
ADR-0017, config VectorStore defaults, adapters-backlog.md.

Plan and then implement:
1. Keep models.Embedder port; fake-hash remains default for tests.
2. Add one live-or-local adapter selectable by config (e.g. env
   CONTEXT_EMBEDDER_KIND / model id). Prefer a deterministic local option if a
   remote provider is not justified yet; document dimension + embedding_version.
3. Changing dimension requires a new embedding_version; do not silently rewrite
   old vector rows.
4. Wire Chunk 15 commit path and CLI dense modes through the selected Embedder.
5. Contract/unit tests for dimension mismatch rejection.
6. Gated integration if the chosen adapter needs network/files; otherwise
   fully offline.
7. Update local-server.md and progress notes with exact version/dim.

Acceptance criteria:
- Dense path is no longer hard-bound to HashEmbed dim=8 only.
- Fake embedder still powers go test ./... offline.
```

Status: **completed** (2026-07-13)

### Completion notes

- `CONTEXT_EMBEDDER_KIND=fake|local_hash` + `models/factory.OpenEmbedder`.
- `internal/models/localhash`: deterministic SHA256→L2 vectors, pin
  `local-hash-v1` / dim **32**.
- pgvector tables are dimension-scoped (`context_dense_vectors_d{N}`) so dim
  changes do not collide with `CREATE TABLE IF NOT EXISTS`.
- `ValidateEmbeddingPin`: `fake-hash-v1`⇒dim 8; `local-hash-v1`⇒dim 32;
  other versions free; dim change requires version bump.
- CLI ingest/search dense paths use selected Embedder (no longer hard-coded
  `modelfake.Embedder`).
- Fake remains default for offline `go test ./...`.

## Plan Chunk 17: Ignore Patterns And FocusProfile Persistence

Copy-paste prompt:

```text
Work in @Context only. Make real repositories ingestible and focus lenses
durable. Read roadmap Phase 2 (ignore patterns, FocusProfile), corpus/retrieval
FocusProfile types, Postgres metadata store.

Plan and then implement:
1. Add project-scoped ignore patterns (e.g. .contextignore or config list) used
   by LocalFiles / ingest to skip paths deterministically.
2. Persist FocusProfile in MetadataStore (memory + postgres) with list/get/put.
3. CLI: manage focus (put/list/get) and pass --focus on search/context-pack/
   agent-run when provided.
4. Tests for ignore matching and focus round-trip (postgres gated).
5. Update progress notes.

Acceptance criteria:
- Ingest of a typical repo can exclude build/vendor dirs via patterns.
- FocusProfile survives restart when CONTEXT_METADATA_KIND=postgres.
```

Status: **completed** (2026-07-13)

### Completion notes

- `internal/indexing/ignore`: defaults (`vendor/`, `node_modules/`, …) +
  `.contextignore` + `LocalFiles` dir skip.
- `FocusStore` on `MetadataStore` (memory + postgres migration
  `003_focus_profiles.sql`).
- CLI: `focus-put|get|list`; `--focus` on search / context-pack / agent-run.
- Offline + gated postgres round-trip tests.

## Plan Chunk 18: Language And Lexicon Contract-Test Harnesses

Copy-paste prompt:

```text
Work in @Context only. Add fixture harnesses that prove language and lexicon
adapters can plug in without changing vector/metadata adapters. Do not
implement production context-lang-* or TEI/SKOS dictionaries in this repo.
Read ADR-0015, ADR-0016, future-layer 05A/05B (contracts only), proof
multilingual/lexicon JSON.

Plan and then implement:
1. Contract-test harness for MorphAnalyzer / QueryExpander / Normalizer using
   simple fixtures (en + one additional language surface).
2. Contract-test harness for lexicon Sense/Concept/Attestation/LexiconSource
   fixtures via DocumentStore or typed ports already in internal/lexicon.
3. Golden assertions: token spans preserved; analyzer_version pinned;
   original surface not overwritten; sense/concept filters explainable.
4. No network; no large dictionary corpora in-repo.
5. Document how an external context-lang-* or TEI adapter would satisfy the
   harness in adapters-backlog.md.
6. Update progress notes.

Acceptance criteria:
- Harnesses fail if an adapter breaks span/version/original-text invariants.
- Core still has no product-specific language packs.
```

Status: **completed** (2026-07-13)

### Completion notes

- `internal/linguistic/harness.RunContract` — en + ru surfaces; normalize /
  analyze / expand; span + original surface + analyzer_version pins.
- `internal/lexicon/harness.RunContract` — Sense/Concept/Attestation/
  LexiconSource lookups + explainable sense/concept filters preserving
  checksum/span (aligned with proof `04-lexicon.json` IDs).
- CI defaults: `linguistic/simple` and `lexicon/fake` satisfy harnesses offline.
- External adapter checklist: `.project/adapters-backlog.md` (Chunk 18 section).

## Plan Chunk 19: Eval Golden Harness

Copy-paste prompt:

```text
Work in @Context only. Add a deterministic eval harness with a small golden
set covering exact, sparse, dense, and hybrid retrieval plus one pack/verify
check. Read internal/evals ports, proof corpus, Chunk 14–16 adapters.

Plan and then implement:
1. Define golden cases under testdata or .project/proof/eval/ (neutral fixtures).
2. Runner executes retrieval modes and records pass/fail + scores/reasons.
3. Include at least one multilingual and one lexicon-filter golden from Chunk 18
   fixtures if available.
4. CLI or go test entrypoint: context-dev eval or go test ./internal/evals/...
5. Offline by default; optional postgres-gated denser path.
6. Emit JSON report suitable for Lab.
7. Update progress notes with commands.

Acceptance criteria:
- go test or documented command fails on retrieval regressions.
- Report is machine-readable for Lab without importing internal packages.
```

Status: **completed** (2026-07-13)

### Completion notes

- Suite: `internal/evals/golden` — exact / sparse / dense / hybrid +
  multilingual + lexicon + pack_verify (offline fake dense).
- Specs listed in `.project/proof/eval/golden.json` (Lab-readable).
- Commands:
  - `go test ./internal/evals/golden/ -count=1`
  - `go run ./cmd/context-dev eval --out .project/proof/eval/report.json`
  (non-zero exit when `report.ok=false`).
- Report JSON keys: `ok`, `suite_id`, `generated_at`, `cases`, `summary`.

## Plan Chunk 20: Thin HTTP Or gRPC Service Contract

Copy-paste prompt:

```text
Work in @Context only. Expose a thin service API over existing CLI/domain
contracts so Lab/BFF can operate without importing internal/. Prefer HTTP+JSON
first unless an accepted ADR requires gRPC. Read roadmap Phase 2 service/SDK
notes, progress consumer track, brand-neutral-core rule.

Plan and then implement:
1. Add ADR for service boundary: endpoints mirror proven CLI operations
   (health, init/ingest status, search, context-pack, agent-run, trace, focus,
   eval report fetch); auth deferred or minimal shared-secret for local only.
2. Implement server under cmd/ or internal/httpserver (or grpc) wiring the same
   stores/config as context-dev.
3. Stable request/response JSON (or protobuf) aligned with existing CLI DTO
   field names where possible.
4. Do not leak host filesystem paths; use path_key / context_ref (ADR-0013).
5. Integration test: httptest or grpc test against memory or gated postgres.
6. Document curl/grpcurl examples in .project/local-server.md or api.md.
7. Explicitly out of scope: full SDK, multi-tenant auth, Lab UI, QDrant.
8. Run go test ./... and update progress completion notes.

Acceptance criteria:
- A downstream client can search + build a ContextPack + fetch a trace over the
  network API using only the public contract.
- Unit tests remain offline-green.
- Context still does not import Lab.
```

Status: **completed** (2026-07-13)

### Completion notes

- ADR-0024: HTTP+JSON first; CLI DTO field alignment; no host path leak;
  optional local shared-secret auth.
- `internal/httpserver` + `cmd/context-serve --data <dir> [--addr :8080] [--token]`.
- Routes: `/health`, `/v1/status`, `/v1/search`, `/v1/context-pack`,
  `/v1/agent-run`, `/v1/trace`, `/v1/focus`, `/v1/focuses`, `/v1/eval`,
  `/v1/ingest` (`path_key` relative to corpus only).
- Tests: `go test ./internal/httpserver/` (httptest, offline).
- Docs: `.project/local-server.md` curl examples; decisions index updated.

## Plan Chunk 21: Thin pkg/contextkit HTTP Client

Copy-paste prompt:

```text
Work in @Context only. Promote the first public Go consumer surface under
pkg/contextkit: an HTTP client over the Chunk 20 / ADR-0024 JSON contract.
Read ADR-0001, ADR-0024, progress.md. Do not export internal domain ports.
Do not add OpenAPI generation, gRPC, or multi-tenant auth.

Plan and then implement:
1. Add pkg/contextkit with Client (BaseURL, optional Token) and methods mirroring
   proven endpoints: Health, Status, Search, ContextPack, AgentRun, Trace,
   FocusPut/Get/List, Eval, Ingest.
2. Public request/response DTOs use the same JSON field names as CLI/HTTP;
   keep types minimal (strings/IDs as string) — no import of internal/.
3. Unit tests in pkg/contextkit against httptest mock JSON.
4. Compat smoke in internal/httpserver importing contextkit against a real
   Server handler (search + pack + agent + trace).
5. Document import path and example in README / local-server.md.
6. Explicitly out of scope: full domain SDK, codegen, QDrant, Lab UI.
7. Run go test ./... and update progress completion notes.

Acceptance criteria:
- A downstream Go module can call Context using only github.com/fastygo/context/pkg/contextkit.
- pkg/contextkit has zero imports of github.com/fastygo/context/internal/...
- Offline go test ./... stays green.
```

Status: **completed** (2026-07-13)

### Completion notes

- Package: `pkg/contextkit` — `Client` + DTOs aligned with ADR-0024 JSON.
- Methods: Health, Status, Search, ContextPack, AgentRun, Trace, Focus*, Eval,
  Ingest. Nested pack/run/events kept as `json.RawMessage` where full domain
  trees would otherwise leak into pkg/.
- Guard: `go list` test fails if `pkg/contextkit` imports `internal/`.
- Compat: `internal/httpserver` smoke uses contextkit against real Server.
- Docs: README + `.project/local-server.md`.

## Completion Notes

Use this section after each Phase 2 chunk. Keep notes short and factual.

### Phase 1 closed (2026-07-13)

- Chunks 01–13 completed; proof hypothesis validated; durable CLI opt-in.
- See `.project/proof/SUMMARY.md` and ADRs 0001–0023.
