# ADR-0041: Query AST Deferred — FTS + API Filters (C10)

Status: Accepted — **partially superseded by
[ADR-0043](0043-ru-adapter-operator-query-layer.md)** (2026-07-16): the reopen
condition in Follow-ups was met by the lexical-search product mandate; a
minimal operator layer shipped with golden tests and trace payloads. Field
filters and the no-general-DSL stance remain in force.  
Date: 2026-07-14  
Related: [0008](0008-hybrid-index-architecture.md),
[0009](0009-context-sparse-tantivy-sidecar.md),
[0039](0039-s3-adapter-freeze-defer.md) (A10 fuzzy),
stabilization gap **C10**, future-layer L04

## Context

S4 requires a Query AST subset (phrase / AND-OR-NOT / field filters) **or** an
explicit forever-defer ADR naming the supported power-user path. Building a
general boolean DSL in core would expand domain language without a measured
blocker and overlaps Postgres FTS / consumer query UIs.

## Decision

1. **Forever-defer** a first-class Query AST / boolean DSL inside Context Runtime.
2. **Supported power-user path** until measured need:
   - **Exact phrase:** `exact` retriever (case-sensitive substring / quoted
     intent at the caller).
   - **Sparse/FTS:** `postgresfts` (or fake sparse offline); callers may pass
     backend-native query text at the sparse adapter boundary only.
   - **Field / facet filters:** `RetrievalFilters` (language, sense, concept,
     attestation, register, temporal range, trust via pack focus, etc.).
   - **Hybrid:** compose strategies via `RetrievalPlan.Strategies` + optional
     morph expansion — not a boolean parse tree.
3. **Consumer-side pattern:** Lab/BFF parse rich query UX into (a) one or more
   search calls with modes/filters, and/or (b) Postgres FTS operators in an
   ops-owned sparse path. Do not push AND/OR/NOT AST into core packs or traces
   as a required type.
4. Fuzzy/trigram remains out of core per ADR-0039 (`pg_trgm` recipe outside).

## Consequences

### Positive

- Deterministic retrieval stays explainable without a second query language.
- Power users keep Postgres FTS and API filters without waiting on a DSL.

### Negative

- No golden Query AST suite in core; boolean composition is a consumer concern.
- Cross-backend boolean parity is explicitly not promised.

### Follow-ups

- Reopen only if API filters + FTS cannot express a measured product query
  without forking the engine — then ship a minimal AST with golden tests and
  trace payloads.
