# Architecture Decision Records

Durable **why** decisions for `github.com/fastygo/context`. Implementation
detail lives in code; how-to lives in [`docs/`](../README.md).

**Related**

| Document | Role |
| --- | --- |
| [docs/README.md](../README.md) | User/LLM navigation |
| [`.project/roadmap-context-core.md`](../../.project/roadmap-context-core.md) | Architecture baseline |
| [`.project/future-layer.md`](../../.project/future-layer.md) | Deferred layers |
| **decisions/** (here) | Normative ADRs |

When ADR wording and a planned roadmap disagree on **stack order**, follow the
active roadmap note and publish a superseding ADR if needed.

## Foundation decision gate

Status: **closed** — ADR-0015–0021 (and later Lab-ready ADRs through 0027).

| Topic | ADR |
|-------|-----|
| Multilingual linguistic contracts | [0015](0015-multilingual-linguistic-contracts.md) |
| Lexicographic contracts | [0016](0016-lexicographic-context-contracts.md) |
| PoC backend order | [0017](0017-poc-backend-order.md) |
| Identity and spans | [0018](0018-deterministic-identity-and-spans.md) |
| Retrieval scoring | [0019](0019-phase1-retrieval-scoring.md) |
| ContextPack budget | [0020](0020-contextpack-budget-and-evidence.md) |
| Snapshot commit failure | [0021](0021-snapshot-commit-failure-semantics.md) |

## Index (all)

| ADR | Title | Status |
| --- | --- | --- |
| [0001](0001-package-boundary-internal-first.md) | Package boundary: internal-first | Accepted |
| [0002](0002-metadata-store-progression.md) | Metadata store progression | Accepted |
| [0003](0003-artifact-store-progression.md) | Artifact store progression | Accepted |
| [0004](0004-vector-namespace-abstraction.md) | Vector namespace abstraction | Accepted |
| [0005](0005-model-adapters-fake-first.md) | Model adapters fake-first | Accepted |
| [0006](0006-trace-event-append-only-replay.md) | Trace event append-only replay | Accepted |
| [0007](0007-embedded-kv-intermediate-layers.md) | Embedded KV intermediate layers | Accepted |
| [0008](0008-hybrid-index-architecture.md) | Hybrid index architecture | Accepted |
| [0009](0009-context-sparse-tantivy-sidecar.md) | context-sparse / Tantivy sidecar | Accepted |
| [0010](0010-local-cloud-deployment-parity.md) | Local/cloud deployment parity | Accepted |
| [0011](0011-merkle-manifest-and-snapshot-namespace.md) | Merkle manifest and snapshot namespace | Accepted |
| [0012](0012-index-snapshot-sync-replication.md) | Index snapshot sync / replication | Accepted |
| [0013](0013-context-ref-and-path-alias.md) | ContextRef and path alias | Accepted |
| [0014](0014-storage-role-separation.md) | Storage role separation | Accepted |
| [0015](0015-multilingual-linguistic-contracts.md) | Multilingual linguistic contracts | Accepted |
| [0016](0016-lexicographic-context-contracts.md) | Lexicographic context contracts | Accepted |
| [0017](0017-poc-backend-order.md) | PoC backend order | Accepted |
| [0018](0018-deterministic-identity-and-spans.md) | Deterministic identity and spans | Accepted |
| [0019](0019-phase1-retrieval-scoring.md) | Phase 1 retrieval scoring | Accepted |
| [0020](0020-contextpack-budget-and-evidence.md) | ContextPack budget and evidence | Accepted |
| [0021](0021-snapshot-commit-failure-semantics.md) | Snapshot commit failure semantics | Accepted |
| [0022](0022-structured-artifact-schema-id.md) | Structured artifact schema identity | Accepted |
| [0023](0023-derived-artifact-lineage-temporal-source-metadata.md) | Lineage and temporal source metadata | Accepted |
| [0024](0024-thin-http-service-boundary.md) | Thin HTTP service boundary | Accepted |
| [0025](0025-multi-tenant-isolation.md) | Multi-tenant isolation boundary | Accepted |
| [0026](0026-public-api-v1-freeze.md) | Public API v1 compatibility freeze | Accepted |
| [0027](0027-lab-gate-freeze.md) | Lab gate freeze | Accepted |
| [0028](0028-source-tombstones.md) | Source tombstones for soft delete | Accepted |
| [0029](0029-snapshot-bundle-export-import.md) | Snapshot bundle export/import | Accepted |
| [0030](0030-project-export-delete.md) | Project export and delete retention hooks | Accepted |
| [0031](0031-durable-schedule-port.md) | Durable schedule port for background jobs | Accepted |
| [0032](0032-index-lifecycle-explain.md) | Index lifecycle explain report | Accepted |
| [0033](0033-offset-stable-snippets.md) | Offset-stable snippet / highlight contract | Accepted |
| [0034](0034-tool-side-effect-approval.md) | Tool side-effect approval baseline | Accepted |
| [0035](0035-prompt-injection-fixtures.md) | Prompt-injection threat fixtures | Accepted |
| [0036](0036-intentional-reranker-path.md) | Intentional reranker path | Accepted |
| [0037](0037-public-langtestkit.md) | Public language adapter testkit | Accepted |
| [0038](0038-s3-thin-adapters.md) | S3 thin in-repo adapters | Accepted |
| [0039](0039-s3-adapter-freeze-defer.md) | S3 freeze-defer A6/A8/A9/A10 | Accepted |
| [0040](0040-graph-consumer-projection.md) | Graph remains a consumer projection (C9) | Accepted |
| [0041](0041-query-ast-defer-fts-filters.md) | Query AST deferred; FTS + filters (C10) | Accepted |
| [0042](0042-stabilization-gate.md) | Stabilization Gate closed | Accepted |

## Writing a new ADR

1. Copy the numbering scheme (`00NN-slug.md`).
2. State context, decision, consequences (positive/negative/follow-ups).
3. Link from this index.
4. Prefer updating [docs/](../README.md) how-to when behavior is user-visible.
