# ADR-0037: Public Language Adapter Testkit

Status: Accepted  
Date: 2026-07-14  
Related: [0001](0001-package-boundary-internal-first.md),
[0015](0015-multilingual-linguistic-contracts.md),
stabilization gaps **A1**, **A2**

## Context

External `context-lang-*` repos cannot import `internal/`. The Chunk 18 harness
lived only under `internal/linguistic/harness`, so adapters could not depend on
a published contract test surface.

## Decision

1. Publish `pkg/langcontract` — adapter-facing ports and value types (string IDs,
   public `ByteSpan`).
2. Publish `pkg/langtestkit` — `RunContract`, fixtures, expander maps.
3. Ship reference English adapter `pkg/langtestkit/refen` (`context-lang-en`)
   that passes the public harness.
4. Keep `internal/linguistic` for core indexing/retrieval; mirror thin adapter
   under `internal/linguistic/en` for in-repo harness CI.
5. Dependency direction remains: adapters → public packages; core never imports
   external `context-lang-*` repositories.

## Consequences

### Positive

- Adapter repos can `go get` and gate CI on `langtestkit.RunContract`.
- A1/A2 exit criteria are met without embedding dictionaries in core.

### Negative

- Dual type surfaces (`pkg/langcontract` vs `internal/linguistic`) until a later
  unify ADR; bridges stay explicit.

### Follow-ups

- Optional unify of internal types onto `pkg/langcontract` when two external
  adapters are live.
