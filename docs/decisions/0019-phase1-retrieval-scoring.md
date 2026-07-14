# ADR-0019: Phase-1 Retrieval Scoring

Status: Accepted
Date: 2026-07-11
Related: [0008](0008-hybrid-index-architecture.md),
[0015](0015-multilingual-linguistic-contracts.md),
[0016](0016-lexicographic-context-contracts.md),
[0020](0020-contextpack-budget-and-evidence.md)

## Context

Hybrid retrieval needs a deterministic merge before any model-based reranker.
Without a written scoring policy, candidate order will drift between adapters and
tests will not catch regressions.

## Decision

### 1. Phase-1 scope

Phase 1 uses **deterministic weighted merge only**. Model-based reranking, LLM
query rewriting, and learned cross-encoders are deferred to phase 2 behind a
`Reranker` port (fake no-op until then).

### 2. Candidate identity and dedup

Primary dedup key:

```text
source_id + span_start + span_end + text_checksum
```

If `chunk_id` is present and spans match, merge into one candidate and **preserve
all contributing retriever scores** as a list of `ScoreContribution` records.

Do not drop a contribution silently when merging.

### 3. Per-retriever score normalization

Each retriever emits a raw score plus an explanation. Before merge, normalize
within the current result set for that retriever call:

```text
norm = 0 if max_raw == min_raw
norm = (raw - min_raw) / (max_raw - min_raw) otherwise
```

Normalized scores are in `[0, 1]`. Exact lookup may emit `{0, 1}` only.
Retrievers must not assume global corpus statistics in phase 1 unless the
adapter documents it and pins the version on the snapshot.

### 4. ScoreContribution shape

```text
retriever_id        # exact | sparse | dense | lemma | sense | ...
raw_score
normalized_score
weight              # from RetrievalPlan
explanation         # structured reason codes + human string
snapshot_id
project_id
expansion_ids[]     # if matched via QueryExpansion
sense_id / concept_id / attestation_id  # optional
analyzer_version / embed_version        # when applicable
```

Explanation reason codes (phase 1 minimum): `exact_phrase`, `exact_span`,
`sparse_term`, `dense_similarity`, `lemma_match`, `wordform_expand`,
`sense_filter`, `concept_filter`, `attestation_filter`, `trust_boost`,
`recency_boost`, `citation_boost`.

### 5. Merge formula

```text
merged = sum(weight_i * normalized_score_i) + sum(boost_j)
```

Default weights (overridable by `RetrievalPlan`, must be traced):

| Retriever family | Default weight |
|------------------|----------------|
| exact | 1.00 |
| sparse / FTS | 0.75 |
| lemma / wordform expand | 0.70 |
| dense | 0.55 |
| sense / concept filter hit | 0.20 (additive boost cap) |

Default boosts (additive, each capped, all traced):

| Boost | Cap |
|-------|-----|
| exact phrase match | +0.25 |
| citation / attestation present on factual tasks | +0.15 |
| trust_level >= required | +0.10 |
| recency within FocusProfile window | +0.10 |

Boosts cannot promote a candidate that failed ACL/project filters. Filters run
**before** merge.

### 6. Stable tie-breaking

Sort by:

1. `merged_score` descending
2. retriever priority rank (exact > sparse > lemma > dense > other)
3. `chunk_id` ascending
4. `source_id` ascending
5. `span_start` ascending

Equal inputs must yield equal order in tests.

### 7. Rejection recording

Candidates removed by policy, trust, budget pre-filter, or duplicate dominance
are retained in a rejected list with reason codes when they were in the top-N
raw pool (N from `RetrievalPlan`, default 50). See ADR-0020 for pack-level
rejection.

## Consequences

### Positive

- Unit tests can golden-file merge order without models.
- Expansions and sense hits remain explainable.

### Negative

- Min-max normalization is set-local; absolute scores are not comparable across
  queries (acceptable for phase 1).
- Default weights will need tuning after Chunk 12.

### Follow-ups

- ~~Phase-2 `Reranker` adapter with mandatory trace of pre/post order.~~
  Closed by [ADR-0036](0036-intentional-reranker-path.md) (Identity / ModelAdapter).
- Evaluation harness comparing exact-only vs hybrid on fixture corpora.
