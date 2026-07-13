# ADR-0024: Thin HTTP Service Boundary

Status: Accepted  
Date: 2026-07-13  
Related: [0001](0001-package-boundary-internal-first.md),
[0013](0013-context-ref-and-path-alias.md),
[0017](0017-poc-backend-order.md)

## Context

Phase 2 exit requires Lab/BFF consumers to operate Context without importing
`internal/`. The CLI already exposes stable JSON DTOs (`search`, `context-pack`,
`agent-run`, `trace`, `focus-*`, `eval`). A network boundary must mirror those
contracts without inventing a parallel domain model or a full SDK.

gRPC remains optional; no accepted ADR requires it for the first service cut.

## Decision

1. Prefer **HTTP + JSON** as the first public service surface
   (`cmd/context-serve` + `internal/httpserver`).
2. Endpoints mirror proven CLI operations: health, workspace status, search,
   context-pack, agent-run, trace, focus put/get/list, eval report.
3. Request/response field names align with existing CLI DTOs
   (`project_id`, `query`, `mode`, `focus_id`, `context_pack`, `run`, `events`,
   `report`, …).
4. The process owns `--data` (and optional corpus root for ingest). Clients do
   **not** send absolute host filesystem paths. Ingest accepts optional
   `path_key` relative to the configured corpus root (ADR-0013). Status and
   other responses omit absolute `corpus_root` / host paths.
5. Auth is deferred for multi-tenant use. Optional local shared secret via
   `Authorization: Bearer <token>` or `X-Context-Token` when
   `CONTEXT_SERVE_TOKEN` / `--token` is set.
6. Server wires the same stores/config env as `context-dev` (metadata, sparse,
   dense, embedder). No Lab imports; no QDrant in this chunk.
7. Promote a curated `pkg/` client later only after this contract survives
   consumer use (ADR-0001 still applies).

## Consequences

### Positive

- Lab/BFF can call Context over the network with curl or any HTTP client.
- One DTO vocabulary for CLI and HTTP reduces dual-maintenance risk.
- Offline `go test` stays green via `httptest` + in-memory/local workspace.

### Negative

- HTTP surface is still process-local workspace oriented (one `--data` root),
  not multi-tenant SaaS.
- Full SDK, OpenAPI generation, and gRPC are deferred.

### Follow-ups

- OpenAPI or protobuf only when a second consumer proves the shape.
- Multi-tenant auth and project routing after Chunk 20.
- Optional gRPC adapter behind the same application handlers if needed.
