# ADR-0001: Package Boundary — Internal First

Status: Accepted  
Date: 2026-06-17  
Related: [0006](0006-trace-event-append-only-replay.md)

## Context

The module is an MIT-licensed library core used by multiple downstream products
(BuildY browser assistant, LingvY, CLI tools). Public API surface must stay
stable and brand-neutral. Premature export of half-formed packages creates
breaking-change pressure before retrieval and indexing contracts are proven.

## Decision

1. Implement under `internal/` until interfaces survive at least one PoC CLI loop
   and integration tests.
2. Promote a curated surface to `pkg/contextkit/` (or similar) only after:
   - domain types (`Project`, `Chunk`, `ContextPack`, `AgentRun`) stabilize;
   - storage and retriever adapters have two working implementations (e.g. memory
     + Postgres/QDrant);
   - trace replay is demonstrated end-to-end.
3. Keep product identity, companions, and scenario-specific naming out of core
   packages, comments, and examples.

## Consequences

### Positive

- Freedom to refactor indexing and retrieval internals during PoC.
- Clear signal when an API is intentionally public.

### Negative

- Downstream repos cannot import unexported packages; they use CLI or future
  `contextkit` only.

### Follow-ups

- Define the first `contextkit` export list in Plan Chunk 02 after domain models
  land.
