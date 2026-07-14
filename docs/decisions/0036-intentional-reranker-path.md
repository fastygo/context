# ADR-0036: Intentional Reranker Path

Status: Accepted  
Date: 2026-07-14  
Related: [0005](0005-model-adapters-fake-first.md),
[0019](0019-phase1-retrieval-scoring.md),
stabilization gap **C11**

## Context

`models.Reranker` and `retrieval.Reranker` existed but hybrid search never
called them. The path must be intentional (even as no-op/weighted).

## Decision

1. `hybrid.Engine.Reranker` is optional post-`DedupAndMerge`; when set, emit a
   trace payload with `phase=rerank` and pre/post counts.
2. `rerank.Identity` is the documented intentional no-op; CLI search wires it.
3. `rerank.WeightedResort` re-sorts by `MergedScore`; `rerank.ModelAdapter`
   bridges `models.Reranker` passage scores onto candidates.
4. Deterministic weighted merge remains the ranking baseline; model rerank is
   additive.

## Consequences

### Positive

- S2 exit: reranker path is wired and tested, not an unused interface.
- Cross-encoder adapters can plug in without changing merge math.

### Negative

- Identity adds a no-op stage on CLI search (cheap; keeps the hook live).

### Follow-ups

- Export pre/post order in Lab metrics when consumers need debug UI.
