# ADR-0039: S3 Freeze-Defer (A6, A8, A9, A10)

Status: Accepted  
Date: 2026-07-14  
Related: [0003](0003-artifact-store-progression.md),
[0005](0005-model-adapters-fake-first.md),
[0017](0017-poc-backend-order.md),
[0038](0038-s3-thin-adapters.md),
stabilization gaps **A6**, **A8**, **A9**, **A10**

## Context

S3 allows implement **or** freeze-defer for A8–A10. A6 (DOCX) is optional when
PDF ships. These items must not reopen casually after Stabilization Gate.

## Decision

| Gap | Decision | Rationale |
| --- | --- | --- |
| **A6** DOCX | **Freeze-defer** | PDF adapter (A5) covers binary document path; DOCX waits for measured need |
| **A8** Object-store ArtifactStore | **Freeze-defer: localfs-only until measured** | ADR-0003 progression; no multi-node artifact volume yet |
| **A9** Provider Completer/Embedder | **HTTP JSON is the supported production adapter** | `models/httpjson` already ships; vendor SDKs stay out of core |
| **A10** Fuzzy/trigram | **Freeze-defer: Postgres `pg_trgm` recipe outside core** | Sparse FTS remains default; typo path is ops/SQL until measured blocker |

## Consequences

### Positive

- S3 closes without expanding domain language or vendor lock-in.
- Clear reopen condition: measured blocker + superseding ADR.

### Negative

- DOCX and object-store remain unavailable in-core until reopened.

### Follow-ups

- None until a deployment is blocked.
