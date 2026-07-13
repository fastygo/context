# Chunk 12 Proof Summary

- Status: **validated**
- Hypothesis: ingest → hybrid/dense retrieve → ContextPack → fake agent → verifier → replayable trace, with pgvector + postgres metadata
- Project: `proof` snapshot `snap_1`
- Checked at: 2026-07-13T12:00:12Z

## Steps

- [OK] `init-project` project=proof
- [OK] `ingest` chunks=9 → `01-ingest.json`
- [OK] `search-exact` candidates=2 backend= → `02-search-exact.json`
- [OK] `search-sparse` candidates=2 backend= → `02-search-sparse.json`
- [OK] `search-hybrid` candidates=2 backend= → `02-search-hybrid.json`
- [OK] `search-dense` candidates=9 backend=postgres_pgvector → `02-search-dense.json`
- [OK] `search-hybrid-dense` candidates=9 backend=postgres_pgvector → `02-search-hybrid-dense.json`
- [OK] `multilingual` language+expansion+token_span → `03-multilingual.json`
- [OK] `lexicon` sense/concept/attestation/register/region/time/authority → `04-lexicon.json`
- [OK] `context-pack` pack=pack_snap_1 → `05-context-pack.json`
- [OK] `agent-run` run=run_pack_snap_1 verify_ok=true → `06-agent-run.json`
- [OK] `trace` events=7 → `07-trace.json`
- [OK] `events-lineage-temporal` temporal filter + lineage ≠ runtime trace → `08-events-lineage-temporal.json`

## Gaps

- ~~CLI ingest still persists workspace state.json only~~ **Closed in Chunk 13:**
  opt-in `CONTEXT_METADATA_KIND=postgres` persists ingest/agent/trace; state.json
  remains an offline cache.
- ~~Sparse path remains fake term-overlap~~ **Closed in Chunk 14:**
  `CONTEXT_SPARSE_KIND=postgres_fts` live FTS; fake remains default offline.
  Lexical limits: [14-sparse-fts-limits.md](14-sparse-fts-limits.md).
- Dense embeddings: default `fake-hash-v1` dim=8; selectable offline
  `local_hash` (`local-hash-v1` dim=32) via `CONTEXT_EMBEDDER_KIND` (Chunk 16).
  Provider/TEI adapters remain deferred. Dense rows upsert on ingest when
  `CONTEXT_ENABLE_DENSE=1` (Chunk 15).
- Language/lexicon: contract harnesses in `internal/linguistic/harness` and
  `internal/lexicon/harness` (Chunk 18); production adapters stay external.
- Eval: offline golden suite `internal/evals/golden` + `context-dev eval`
  (Chunk 19); report under `.project/proof/eval/`.
- Service: thin HTTP+JSON `cmd/context-serve` / `internal/httpserver`
  (Chunk 20, ADR-0024); Lab/BFF call without importing `internal/`.
- Client: `pkg/contextkit` HTTP client (Chunk 21); zero `internal/` imports.
- Ops: append-only eval history + workspace metrics (Chunk 22);
  `ops/eval_history.jsonl` path_key under data dir.
- Repair: rebuild / retry-failed (Chunk 23, ADR-0021); `last_failed` in state.
- Isolation: ADR-0025 Tenant/Project boundary; leakage contract tests (Chunk 24).
- API v1 freeze: ADR-0026 + `.project/api-v1.md` (Chunk 25).
- Inspector: `inspect` / `POST /v1/inspect` Lab JSON (Chunk 26).
- Models: `localecho` Completer + `http` Completer/Embedder (Chunk 27).
- Soft quotas: allow/ask/deny via `CONTEXT_QUOTA_MAX_*` (Chunk 28).
- Multilingual/lexicon proofs use in-memory fixtures with simple-lang adapters; context-lang-* and TEI/SKOS lexicon adapters are not wired.

## Next decisions

- ~~Decide when ingest/agent-run should default MetadataStore to postgres~~ **Decided (Chunk 13):** opt-in via `CONTEXT_METADATA_KIND=postgres` (not default).
- Before QDrant/Turbopuffer: keep BackendCapabilities contract tests against pgvector + candidate.
- Before context-sparse: measure Postgres FTS / fake sparse lexical limits on a larger corpus.
- Before context-lang-*: pin analyzer_version/dictionary_version on chunk rows during ingest.
- Before TEI/SKOS lexicon adapters: promote DocumentStore sense/concept/attestation payloads into typed lexicon ports only after schema stability.
