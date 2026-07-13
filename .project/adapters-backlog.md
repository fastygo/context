# Adapter Backlog (Post–Chunk 10)

Ports are stable in `internal/retrieval` and `internal/models`. Live adapters
beyond PostgreSQL/pgvector wait for Chunk 12 proof or a measured blocker
([ADR-0017](decisions/0017-poc-backend-order.md)).

| Port | First live | Later adapters | Notes |
| --- | --- | --- | --- |
| `VectorStore` | `internal/retrieval/dense/postgresvector` | QDrant, Turbopuffer | Same `BackendCapabilities`; shared-collection + payload filters first |
| `SparseSearchClient` | `internal/retrieval/sparse/postgresfts` (Chunk 14) | `context-sparse` (Tantivy) | Fake/memory remains default offline; do not add Tantivy until FTS lexical limits are a measured blocker |
| `Embedder` | `models/fake` (default) + `models/localhash` (`local-hash-v1`, dim 32) | Provider embedding adapters | Select via `CONTEXT_EMBEDDER_KIND`; dim change requires new `embedding_version` |
| Language | `linguistic/simple` + `linguistic/harness` (Chunk 18) | `context-lang-*` repositories | Pass `harness.RunContract` (spans, analyzer_version, original surface, expansions). Do not change vector/metadata adapters. |
| Lexicon | `lexicon/fake` + `lexicon/harness` (Chunk 18) | TEI/SKOS/dictionary mappers | Implement `ResourceAdapter`; pass `harness.RunContract` (sense≠lemma, attestation quote+span, explainable filters). |
| `MetadataStore` | `internal/storage/postgres` | SQLite (optional), bbolt cache later | Migrations on Open; DocumentStore for lex/ling JSON |
| `ArtifactStore` | localfs | Object store | Unchanged by dense path |

## How external language / lexicon adapters satisfy Chunk 18 harnesses

### `context-lang-*` (MorphAnalyzer / Normalizer / QueryExpander)

1. Implement `linguistic.MorphAnalyzer`, `LexicalNormalizer`, and `QueryExpander`.
2. In the adapter repo (or a thin test package), call:

```go
linguistic_harness.RunContract(t, linguistic_harness.Ports{
  Normalizer: myNorm, Analyzer: myMorph, Expander: myExpand,
})
```

3. Provide expander fixtures covering at least `run`→`runners` (or document
   equivalent maps in the test). Failures mean: missing `adapter_id` /
   `analyzer_version`, mutated `TokenOccurrence.Surface`, or dropped spans.

### TEI / SKOS lexicon mappers (`ResourceAdapter`)

1. Map external resources into `lexicon.Sense` / `Concept` / `Attestation` /
   `LexiconSource` (do not invent attestations from generated wordforms).
2. Seed `lexicon/harness.DefaultSeed()` IDs or an equivalent fixture set and call
   `lexicon_harness.RunContract(t, adapter, seed)`.
3. Filters must remain explainable via retrieval reasons (`sense_filter`,
   `concept_filter`); original chunk text checksum and spans must not change.

Neither path may import product brands into the core or require network corpora.

## Capability contract

Adapters should implement `retrieval.CapabilityReporter` when they can declare:

- project/snapshot filter enforcement
- temporal / metadata filter enforcement (pgvector PoC: **false** — client index filters)
- dimension, metrics, namespace model, managed-service flag

## Trigger to add QDrant / Turbopuffer / context-sparse

1. Chunk 12 CLI proof with pgvector completed and limits recorded.
2. Superseding ADR updates PoC backend order if needed.
3. Contract tests pass against pgvector + the candidate adapter.
