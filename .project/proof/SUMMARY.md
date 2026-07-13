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

- CLI ingest still persists workspace state.json; postgres metadata is proven via meta path / proof helpers, not yet the default ingest backend.
- Sparse path remains fake term-overlap; Postgres FTS and context-sparse are not required for this proof.
- Dense embeddings use fake-hash-v1 dim=8; live embedding providers are deferred.
- Multilingual/lexicon proofs use in-memory fixtures with simple-lang adapters; context-lang-* and TEI/SKOS lexicon adapters are not wired.

## Next decisions

- Decide when ingest/agent-run should default MetadataStore to postgres (CONTEXT_METADATA_KIND=postgres).
- Before QDrant/Turbopuffer: keep BackendCapabilities contract tests against pgvector + candidate.
- Before context-sparse: measure Postgres FTS / fake sparse lexical limits on a larger corpus.
- Before context-lang-*: pin analyzer_version/dictionary_version on chunk rows during ingest.
- Before TEI/SKOS lexicon adapters: promote DocumentStore sense/concept/attestation payloads into typed lexicon ports only after schema stability.
