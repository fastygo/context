# ADR-0035: Prompt-Injection Threat Fixtures

Status: Accepted  
Date: 2026-07-14  
Related: [0020](0020-contextpack-budget-and-evidence.md),
[0034](0034-tool-side-effect-approval.md),
stabilization gap **C5**

## Context

Untrusted sources can carry instruction-like text. Defense is pack/policy
separation, not a production classifier in core.

## Decision

1. Ship `internal/evals/adversarial` fixtures (grant-tools / override-policy /
   quarantined).
2. Regression: adversarial surfaces cannot appear in pack `Instructions`,
   cannot enter evidence as `instruction`/`policy` class, quarantined stays
   rejected, and `PolicySnapshot` tool decisions are unchanged by pack text.
3. Heuristic `LooksLikeInstructionInjection` is test/docs only.

## Consequences

### Positive

- S2 exit test is executable offline.
- Trust levels + evidence classes remain the primary control plane.

### Negative

- No ML prompt-injection detector in core (deferred to adapters).

### Follow-ups

- Optional classifier hook behind a narrow port when measured need appears.
