# ADR-0042: Stabilization Gate Closed

Status: Accepted  
Date: 2026-07-14  
Related: [0026](0026-public-api-v1-freeze.md),
[0027](0027-lab-gate-freeze.md),
[stabilization-roadmap](../../.project/stabilization-roadmap.md) **S5**

## Context

Lab Gate froze the Lab/BFF contract. Stabilization Gates S0–S4 closed durability,
evidence/safety, thin adapters, and graph/query decisions. S5 is the “don’t
touch” checklist so core only reopens for measured blockers.

## Decision

1. **Stabilization Gate passed** (2026-07-14): S1–S4 exit criteria green;
   offline `go test ./...`, Lab smoke, and golden/adversarial suites green;
   runbooks and additive API changelog published.
2. Default answer to feature requests that expand domain language:
   **adapter or downstream consumer**.
3. Reopen core only with: measured blocker + superseding ADR + tests.
4. Section D items in the stabilization roadmap remain frozen-deferred (owner:
   core steward) until that bar is met.
5. Consumers integrate via HTTP/`pkg/contextkit`/`pkg/langtestkit` only — never
   `internal/`.

## Consequences

### Positive

- Clear calm period for Lab and product work on a stable core.
- Prevents casual reopen of graph AST, object-store, OIDC, etc.

### Negative

- Some power features stay consumer-side by design (ADR-0040/0041).

### Follow-ups

- Phase 4 commercial envelope (billing, team OIDC, COW) remains after S5 unless
  a deployment blocks without them.
