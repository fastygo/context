# Architecture Decision Records

Durable decisions for `github.com/fastygo/context`. Each ADR captures **why**
a boundary exists; implementation detail lives in code and
`.project/roadmap-context-core.md`.

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

## Supersedes notes

- Plan Chunk 01 originally suggested an in-memory sparse retriever first.
  **[0009](0009-context-sparse-tantivy-sidecar.md)** replaces that path: Tantivy
  runs in Docker from the start; Go talks HTTP/gRPC only.

## Background drafts (non-normative)

- `.project/.draft/cursor-storage-inventory.md` — Cursor storage role map
- `.project/.draft/vector layer vscdb.md` — session KV vs vector index notes

## Adding an ADR

1. Copy the nearest existing ADR structure.
2. Use the next free number; do not renumber published ADRs.
3. Link related ADRs in **Related** and update this index.
4. Note completion in `.project/progress.md` when the decision gates work.
