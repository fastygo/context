# ADR-0026: Public API v1 Compatibility Freeze

Status: Accepted  
Date: 2026-07-13  
Related: [0001](0001-package-boundary-internal-first.md),
[0024](0024-thin-http-service-boundary.md),
[0025](0025-multi-tenant-isolation.md)

## Context

Lab/BFF consumers now call Context through HTTP JSON (`cmd/context-serve`) and
`pkg/contextkit` (Chunks 20–21). Without an explicit compatibility rule,
additive refactors can silently break fixtures. OpenAPI codegen and multi-tenant
auth remain deferred; labs still need a pinable `api_version`.

## Decision

1. The supported public surface is **API v1**:
   - HTTP routes under `/v1/...` plus `/health`
   - `pkg/contextkit` types and methods that mirror those routes
   - CLI JSON field names that already match the HTTP DTOs
2. `api_version` is the string `v1`. Servers advertise it on:
   - `GET /health` body field `api_version`
   - response header `X-Context-API-Version: v1` for `/health` and `/v1/*`
3. Compatibility rules for v1:
   - **Breaking:** rename/remove a documented request/response field; change
     status-code meaning for a documented error; remove a `/v1` route; change
     `project_id` isolation semantics.
   - **Allowed without major bump:** add optional JSON fields; add new `/v1`
     routes; add optional query params with safe defaults.
   - Breaking changes require a new major (`v2` routes and a superseding ADR).
4. Clients SHOULD send `Accept: application/json`. `pkg/contextkit` records the
   response `X-Context-API-Version` when present; mismatch with
   `contextkit.APIVersion` is reported as a soft warning via
   `Client.LastAPIVersion` (hard fail optional later).
5. Domain ports under `internal/` are **not** part of API v1.

## Consequences

### Positive

- Labs can pin `v1` and detect drift.
- Clear gate for Chunks 26–29 work without renaming the contract mid-flight.

### Negative

- Some early CLI-only fields may lag documentation until listed in
  `.project/api-v1.md`.

### Follow-ups

- Chunk 26 inspector endpoint must land under `/v1` without breaking v1.
- OpenAPI document generation remains optional after two real consumers.
