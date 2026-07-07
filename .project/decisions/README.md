# Architecture Decision Records

Durable decisions for `github.com/fastygo/context`. Each ADR captures **why**
a boundary exists; implementation detail lives in code and
`.project/roadmap-context-core.md`.

**How this folder relates to other docs**

| Document | Role |
| --- | --- |
| [roadmap-context-core.md](../roadmap-context-core.md) | Full architecture baseline and phased scope |
| [progress.md](../progress.md) | Active plan chunks, acceptance criteria, completion notes |
| **decisions/** (here) | Normative **why** for boundaries; individual ADRs stay canonical |

When ADR wording and `progress.md` disagree on **PoC stack order**, follow
`progress.md` and [roadmap open decisions](../roadmap-context-core.md#open-decisions)
until a superseding ADR is published. ADR-0004/0008/0009/0010 describe target
ports and long-term shape; they do not mandate QDrant or Tantivy for the first
live proof.

## Foundation decision gate

Before runtime code begins, add or supersede only decisions that affect domain
types, adapter ports, trace records, deterministic hashes, retrieval scoring, or
`ContextPack` replay. This keeps the foundation stable without pulling future
infrastructure into the first PoC.

Foundation decisions still needed:

- **Multilingual linguistic contracts:** token spans, lexeme/lemma/wordform
  references, morphology features, ambiguity, query-expansion reasons, analyzer
  versions, and `context-lang-*` adapter lifecycle.
- **Lexicographic context contracts:** senses, concepts, attestations, variants,
  multiword expressions, register/region/time metadata, source authority,
  licensing metadata, and TEI/SKOS/resource-adapter boundaries.
- **PoC backend order:** PostgreSQL + pgvector first for dense vectors,
  PostgreSQL full-text or fake sparse first for lexical tests, later QDrant,
  Turbopuffer, and `context-sparse` only behind the same ports.
- **Deterministic identity and spans:** path keys, source/chunk checksum inputs,
  byte/rune span convention, Unicode/newline normalization, and snapshot hash
  rules.
- **Phase-1 retrieval scoring:** merge/dedup, score normalization boundaries,
  stable tie-breaking, score explanation fields, and deferred model-reranker
  policy.
- **ContextPack budget and evidence classes:** source text vs inference,
  instruction/data separation, trust labels, citation locking, rejected
  candidates, and deterministic trimming.
- **Snapshot commit failure semantics:** minimal `building`, `ready`, `failed`,
  and `superseded` behavior plus idempotent retry rules.

Explicitly **not** foundation blockers: graph traversal engines, QDrant,
Turbopuffer, `context-sparse`, bbolt, Badger, prompt-injection classifiers,
fine-grained ACLs, crawlers, distributed workers, multimodal parsing, and
production retention. Keep extension points where needed; implement those layers
only when a later chunk or measurement requires them.

## Index

| ADR | Title | Status |
|-----|-------|--------|
| [0001](0001-package-boundary-internal-first.md) | Package boundary: internal-first | Accepted |
| [0002](0002-metadata-store-progression.md) | Metadata store progression | Accepted |
| [0003](0003-artifact-store-progression.md) | Artifact store progression | Accepted |
| [0004](0004-vector-namespace-abstraction.md) | Vector namespace abstraction | Accepted |
| [0005](0005-model-adapters-fake-first.md) | Model adapters: fake first | Accepted |
| [0006](0006-trace-event-append-only-replay.md) | Trace events: append-only replay | Accepted |
| [0007](0007-embedded-kv-intermediate-layers.md) | Embedded KV for intermediate layers | Accepted |
| [0008](0008-hybrid-index-architecture.md) | Hybrid index: dense + sparse + manifest | Accepted |
| [0009](0009-context-sparse-tantivy-sidecar.md) | Sparse search via `context-sparse` sidecar | Accepted |
| [0010](0010-local-cloud-deployment-parity.md) | Local and cloud deployment parity | Accepted |
| [0011](0011-merkle-manifest-and-snapshot-namespace.md) | Merkle manifest and snapshot namespace | Accepted |
| [0012](0012-index-snapshot-sync-replication.md) | Index snapshot sync and replication | Accepted |
| [0013](0013-context-ref-and-path-alias.md) | ContextRef and path alias for model context | Accepted |
| [0014](0014-storage-role-separation.md) | Storage role separation (session vs index) | Accepted |

## Adapter progression by plan chunk

Domain code must talk to **ports** (`VectorStore`, `MetadataStore`,
`ArtifactStore`, `SparseSearchClient`), never to pgvector, QDrant, bbolt, or
Postgres types directly. Backend choice is an adapter concern gated by
`progress.md` chunks.

### VectorStore (dense embeddings)

| Stage | Plan chunks | Backend | Notes |
| --- | --- | --- | --- |
| Contracts + fakes | **02–05** | In-memory / deterministic fake `VectorStore` | Prove `project_id`, `snapshot_id`, `embedding_version`, and explainable candidates without Docker |
| CLI proof (no live vector DB) | **06–08** | Still fakes or file-backed test doubles | End-to-end agent loop and `context-dev` JSON without PostgreSQL |
| Live stack bootstrap | **09** | PostgreSQL + `pgvector` extension only | Compose, health checks, config structs — **no domain rewrite** |
| First live adapter | **10** | **`internal/.../postgresvector`** | First production-shaped dense path; integration tests gated by env |
| Hypothesis validation | **12** | Same pgvector adapter | E2E: ingest → hybrid search → `ContextPack` → trace; document gaps before other backends |
| Optional replacement | **After 12** (+ superseding ADR) | **QDrant** or **Turbopuffer** via new `VectorStore` adapter | Same port; domain models unchanged |

**PostgreSQL/pgvector is the default dense backend through Chunk 12.** Keep
QDrant and Turbopuffer behind `VectorStore` ([0004](0004-vector-namespace-abstraction.md));
add them only when measurements or deployment constraints justify a switch.

**When QDrant (or Turbopuffer) may replace pgvector as the preferred adapter**

All of the following should hold:

1. Chunk **12** CLI proof succeeded with pgvector and identified concrete limits
   (latency, filter cost, index size, snapshot restore, multi-tenant isolation).
2. A new **superseding ADR** records the switch trigger, namespace mapping, and
   migration path ([0004](0004-vector-namespace-abstraction.md),
   [0011](0011-merkle-manifest-and-snapshot-namespace.md)).
3. At least two adapters implement the same contract tests (pgvector + candidate).
4. Retrieval still requires `project_id` + `snapshot_id` on every query; index
   payloads still avoid absolute host paths ([0013](0013-context-ref-and-path-alias.md)).

Typical triggers (from roadmap): vector volume, filter latency, quantization,
memory tuning, or independent vector-tier scaling — not “QDrant is mentioned in
ADR-0008.”

### Metadata store (relational, not vectors)

| Stage | Plan chunks | Backend | Notes |
| --- | --- | --- | --- |
| Early implementation | **02–03** | In-memory metadata adapter | `go test ./...` with zero external services |
| CLI proof | **04–08** | In-memory or file-backed metadata | Manifest, snapshots, traces in tests |
| Live stack | **09, 11** | **PostgreSQL** for durable metadata | Chunks, snapshots, `chunk_aliases`, agent runs — separate from vector rows |
| MVP+ | Phase 2+ | Postgres remains default authority | SQLite acceptable for single-node dev per [0002](0002-metadata-store-progression.md) |

Metadata Postgres and pgvector often share one PostgreSQL instance in PoC; they
remain **different roles** ([0014](0014-storage-role-separation.md)): traces and
manifest authority are not a substitute for `VectorStore`, and vectors do not
belong in the session/trace tables ([0006](0006-trace-event-append-only-replay.md)).

### Sparse search (lexical / BM25-style)

| Stage | Plan chunks | Backend | Notes |
| --- | --- | --- | --- |
| Contracts + fakes | **05** | Fake / in-process sparse client | Same `snapshot_id` contract as dense |
| CLI proof | **06–08** | Fake or minimal local sparse double | Exact lookup path is deterministic first |
| Live PoC | **10–12** | **PostgreSQL full-text** or fake sparse baseline | Preferred for one-container PoC per progress correction |
| Optional replacement | After pgvector PoC | **`context-sparse` (Tantivy) sidecar** | [0009](0009-context-sparse-tantivy-sidecar.md); add when lexical scale or morphology in sparse index justifies another service |

ADR-0009 targets Tantivy in Docker for long-term local/cloud parity; that does
**not** block PoC with Postgres FTS or fakes in Chunks 10–12.

### Embedded KV (bbolt / Badger) — cache only, not search

[0007](0007-embedded-kv-intermediate-layers.md) defines **optional acceleration
layers**. They are not required for Chunk 12 success.

| Data | Store | Required for PoC? | When to add |
| --- | --- | --- | --- |
| Active manifest pointer (offline) | bbolt | No | Local offline manifest cache; Postgres is authority when online |
| Embed cache by `chunk_hash` | Badger (or adapter) | No | When re-embedding cost matters; safe to delete and rebuild |
| Morph / rules version pins | Immutable blob + manifest ref | No | With real morphology adapters |
| Merkle node cache | Badger | No | P1 monorepo rescan optimization |
| Inverted index / postings | **Not KV** | — | Sparse port ([0009](0009-context-sparse-tantivy-sidecar.md) or Postgres FTS) |
| Dense vectors | **Not KV** | — | `VectorStore` (pgvector first) |

**KV never replaces PostgreSQL metadata or `VectorStore`.** bbolt/Badger are
single-process caches; cloud source of truth stays Postgres + object store
([0002](0002-metadata-store-progression.md), [0012](0012-index-snapshot-sync-replication.md)).

Add embedded KV adapters only when a chunk needs measured win (embed dedup,
offline manifest pointer) — typically **after** the in-memory path works, not
before Chunk 03 contracts are proven.

### Artifact store

| Stage | Plan chunks | Backend |
| --- | --- | --- |
| PoC | **03+** | Local filesystem under project data dir ([0003](0003-artifact-store-progression.md)) |
| Cloud / team | Phase 2+ | S3-compatible object storage |

## Supersedes notes

- Plan Chunk 01 originally suggested an in-memory sparse retriever first.
  **[0009](0009-context-sparse-tantivy-sidecar.md)** replaces that path for the
  long-term sparse engine: Tantivy in Docker; Go talks HTTP/gRPC only.
- **2026-06 planning correction** (see [progress.md](../progress.md) Chunk 01
  completion notes and roadmap *Open decisions*): the **first live PoC** uses
  **PostgreSQL + pgvector** for dense vectors and **PostgreSQL full-text or fake
  sparse** for lexical tests. QDrant, Turbopuffer, and `context-sparse` remain
  explicit later adapters. Publish a superseding ADR before changing the default
  dense or sparse backend for production-shaped deployments.

## Background drafts (non-normative)

- `.project/.draft/cursor-storage-inventory.md` — Cursor storage role map
- `.project/.draft/vector layer vscdb.md` — session KV vs vector index notes

## Adding an ADR

1. Copy the nearest existing ADR structure.
2. Use the next free number; do not renumber published ADRs.
3. Link related ADRs in **Related** and update this index.
4. Note completion in `.project/progress.md` when the decision gates work.
5. If the ADR changes PoC backend order (e.g. QDrant before pgvector proof),
   mark it **Supersedes** the affected ADR and update the tables in this README.
