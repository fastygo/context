# Adapter Backlog (Post–Chunk 10)

Ports are stable in `internal/retrieval` and `internal/models`. Live adapters
beyond PostgreSQL/pgvector wait for a measured blocker
([ADR-0017](../docs/decisions/0017-poc-backend-order.md)).
Start from [`.project/README.md`](README.md) before promoting a later adapter.

| Port | First live | Later adapters | Notes |
| --- | --- | --- | --- |
| `VectorStore` | `internal/retrieval/dense/postgresvector` | QDrant, Turbopuffer | Same `BackendCapabilities`; shared-collection + payload filters first |
| `SparseSearchClient` | `internal/retrieval/sparse/postgresfts` (Chunk 14) | `context-sparse` (Tantivy) | Fake/memory remains default offline; do not add Tantivy until FTS lexical limits are a measured blocker |
| `Embedder` | `models/fake` (default) + `models/localhash` (`local-hash-v1`, dim 32) | Provider embedding adapters | Select via `CONTEXT_EMBEDDER_KIND`; dim change requires new `embedding_version` |
| Language | `context-lang-en` (`pkg/langtestkit/refen`, `internal/linguistic/en`) + `linguistic/simple` | external `context-lang-*` | Public harness: `pkg/langtestkit.RunContract` ([ADR-0037](../docs/decisions/0037-public-langtestkit.md)). |
| Lexicon | `lexicon/jsonres` (curated JSON) + `lexicon/fake` | TEI/SKOS mappers | Pass `lexicon/harness.RunContract` ([ADR-0038](../docs/decisions/0038-s3-thin-adapters.md)). |
| Parse | plaintext/markdown + `HTML` + `PDF` | richer PDF/OCR, DOCX deferred | Confidence on `Document` ([ADR-0038](../docs/decisions/0038-s3-thin-adapters.md)/[0039](../docs/decisions/0039-s3-adapter-freeze-defer.md)). |
| Event source | `source.NDJSONFiles` | message-export adapters | Idempotent batch + temporal filter ([ADR-0038](../docs/decisions/0038-s3-thin-adapters.md)). |
| `MetadataStore` | `internal/storage/postgres` | SQLite (optional), bbolt cache later | Migrations on Open; DocumentStore for lex/ling JSON |
| `ArtifactStore` | localfs (**freeze until measured**) | Object store | [ADR-0039](../docs/decisions/0039-s3-adapter-freeze-defer.md) |
| Completer / Embedder | `models/fake`, `localhash`, `localecho`, **`httpjson` (production)** | Vendor SDKs outside core | [ADR-0039](../docs/decisions/0039-s3-adapter-freeze-defer.md) |
| Schedule port | `agentruntime/scheduler` + file adapter | External cron/queue | [ADR-0031](../docs/decisions/0031-durable-schedule-port.md) |
| Graph store | **none** (stubs only) | Consumer projection | [ADR-0040](../docs/decisions/0040-graph-consumer-projection.md) |
| Query AST | **none** (FTS + filters) | Consumer boolean UX | [ADR-0041](../docs/decisions/0041-query-ast-defer-fts-filters.md) |

## How external language / lexicon adapters satisfy Chunk 18 harnesses

### `context-lang-*` (MorphAnalyzer / Normalizer / QueryExpander)

1. Depend on `github.com/fastygo/context/pkg/langcontract` +
   `github.com/fastygo/context/pkg/langtestkit`.
2. Implement the three ports; in the adapter repo call:

```go
langtestkit.RunContract(t, langtestkit.Ports{
  Normalizer: myNorm, Analyzer: myMorph, Expander: myExpand,
})
```

3. Provide expander fixtures covering at least `run`→`runners`. Reference:
   `pkg/langtestkit/refen`. Failures mean: missing `adapter_id` /
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
