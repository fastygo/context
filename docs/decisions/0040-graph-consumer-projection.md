# ADR-0040: Graph Remains a Consumer Projection (C9)

Status: Accepted  
Date: 2026-07-14  
Related: [0001](0001-package-boundary-internal-first.md),
stabilization gap **C9**, future-layer graph notes

## Context

S4 requires either a Postgres edge store + bounded traversal used by the
retrieval planner, **or** an explicit forever-defer ADR. Core today has only
`graph.NodeRef` / `EdgeRef` stubs and an unused `RetrievalFilters.GraphNodeID`
extension point. No measured deployment is blocked on citation/co-occurrence
edges inside the engine.

## Decision

1. **Forever-defer** in-core graph storage, traversal, and planner integration.
2. Core keeps **identity stubs only**: `graph.NodeRef`, `graph.EdgeRef`, and
   optional `RetrievalFilters.GraphNodeID` as a reserved filter field.
3. `MatchesFilters` **ignores** `GraphNodeID` until a superseding ADR implements
   enforcement (setting it must not silently change recall today).
4. **Consumer-side pattern:** products that need citation, reply, or
   co-occurrence graphs own an edge store (or projection table) keyed by
   `project_id` + source/chunk IDs from Context Runtime APIs. They may filter
   candidate `chunk_id`s before or after `POST /v1/search`, then pack with
   ordinary evidence. Core remains the provenance/retrieval engine, not a
   knowledge-graph database.

## Consequences

### Positive

- Stops reopening “do we have a graph?” without measured need.
- Prevents forked half-schemas inside `internal/` while consumers stay free to
  model edges.

### Negative

- No bounded hop traversal in retrieval plans.
- Products must not import `internal/graph` for storage — use public IDs only.

### Follow-ups

- Reopen only with: measured blocker, edge schema ADR, and one store adapter
  plus filter enforcement tests.
