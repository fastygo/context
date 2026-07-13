# ADR-0027: Lab Gate Freeze After Chunk 32

Status: Accepted  
Date: 2026-07-13  
Related: [0024](0024-thin-http-service-boundary.md),
[0026](0026-public-api-v1-freeze.md),
[0025](0025-multi-tenant-isolation.md),
[lab-gate.md](../lab-gate.md)

## Context

Chunks 20–31 shipped the Lab-facing surface: HTTP/`contextkit`, inspector,
quotas, readiness, redaction, and in-process background AgentRun jobs. Lab/BFF
needs an explicit “safe to bind” marker without waiting for auth, OpenAPI, or
language adapters.

## Decision

1. **Lab gate passed** means a downstream Lab may depend on documented API v1
   routes and `pkg/contextkit` for the checklist in [lab-gate.md](../lab-gate.md).
2. Compatibility remains additive under ADR-0026. Redaction fields, job routes,
   readiness, and quota objects are **v1-additive** — not a new major.
3. Lab MUST NOT import `github.com/fastygo/context/internal/...`.
4. Core MUST NOT import Lab packages or bake Lab brand names into the engine.
5. The offline smoke `TestLabGateSmoke` is the regression guard for the gate
   path; it must stay green on `go test ./...` without Docker.

## Consequences

### Positive

- Clear freeze point for Lab integration work.
- Gate is testable offline without hosted infra.

### Negative

- Auth, OpenAPI, and durable multi-process jobs remain out of the gate; Lab
  must design around those deferred items.

### Follow-ups

- Membership/ACL when auth lands.
- OpenAPI when the route set stabilizes further.
- Event/scheduled jobs only after measured need.
