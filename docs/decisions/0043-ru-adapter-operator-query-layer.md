# ADR-0043: In-Repo context-lang-ru + Minimal Operator Query Layer

Status: Accepted  
Date: 2026-07-16  
Related: [0015](0015-multilingual-linguistic-contracts.md),
[0037](0037-public-langtestkit.md), [0038](0038-s3-thin-adapters.md),
[0041](0041-query-ast-defer-fts-filters.md) (partially superseded),
future-layer **L04**, **L05A**

## Context

Product priority shifted to quality lexical search: finding lexemes and
multiword expressions (словосочетания) across inflected Russian text, with
explicit operators. ADR-0041 forever-deferred a core Query AST but named its
own reopen condition: *"Reopen only if API filters + FTS cannot express a
measured product query without forking the engine — then ship a minimal AST
with golden tests and trace payloads."* That condition is met:

- Postgres FTS `simple` config performs no Russian stemming; `plainto_tsquery`
  cannot match `дороги` for query `дорога` (documented in
  `.proofs/14-sparse-fts-limits.md`).
- Exact retrieval is substring-based: `дом` false-matches `Домашний`, and no
  filter expresses token-boundary or lemma-sequence phrase intent.
- Consumer-side AND/OR/NOT composition cannot see morphology expansions, so it
  cannot intersect lexeme-level result sets without duplicating an analyzer.

## Decision

### 1. `context-lang-ru` ships in-repo (thin-but-real)

Following the ADR-0038 precedent (`context-lang-en`), a rule-based Russian
morphology engine lands as an adapter, not as core domain language:

- `pkg/lang/ru` — public engine + `langcontract` adapter. Declension /
  conjugation ending tables, explicit multi-candidate analysis (ambiguity is
  never collapsed), paradigm generation for query expansion, one-way ё→е fold
  (`nfc-lower-yofold-v1`), spelling rule after velars/hushers, and a tiny
  curated exception fixture (бежать, идти, человек, ребёнок, год). No bundled
  dictionaries, no network resources. Passes public `langtestkit.RunContract`.
- `internal/linguistic/ru` — wrapper onto internal linguistic ports.
- `internal/linguistic/registry` — language tag → in-repo adapter ports
  (`en`, `ru`); unknown tags degrade to no expansion.

Dictionary-scale morphology (OpenCorpora etc.) remains external per the
language-adapters plugin roadmap; this adapter proves the contract path with
useful real recall and stays replaceable behind the same ports.

### 2. Minimal operator query layer (`internal/retrieval/querylang`)

A deterministic parser + executor compiled onto existing retrieval paths:

| Operator | Semantics |
| --- | --- |
| `word word` | implicit AND (token-boundary matching, not substring) |
| `OR` / `\|`, `AND`, `NOT` / `-`, `( )` | boolean set combination on chunk sets |
| `"phrase"` | literal substring (exact retriever semantics) |
| `~word` | morphology-expanded term (explainable expansions) |
| `~"phrase"` | consecutive lemma-sequence match (inflected словосочетания) |
| `lang:ru` | selects the language adapter for this query |

Constraints that keep ADR-0041's spirit:

- Leaves reuse existing retrievers and score contracts (ADR-0019 reasons:
  `token_term`, `morph_phrase` added to the enum); no second ranking model.
- Every interpretation is visible: canonical tree, per-leaf accepted/rejected
  expansions, match counts (`query_explain`), plus trace events
  (`layer=querylang`).
- Sparse/FTS signal only reinforces token hits; it cannot widen the
  deterministic operator result set.
- Field/facet filtering remains `RetrievalFilters` — the operator grammar does
  not grow field syntax.
- Pure negation queries are rejected; OR operands must be positive.

### 3. Surfaces (additive v1)

- CLI: `context-dev search --mode query [--lang ru]`; `CONTEXT_LANG` env.
- HTTP: `POST /v1/search` accepts `mode:"query"` and optional `lang`;
  response gains optional `query_explain`.
- `pkg/contextkit`: `SearchRequest.Lang`, `SearchResult.QueryExplain`.
- Hybrid mode also honors the language adapter for expansion (fixes the
  hardcoded `"en"` expansion language in the hybrid engine).

### 4. Golden gates

`eval-golden-v3` adds cases: RU inflection recall, irregular verb recall,
AND/NOT with morphology, lemma-sequence phrase, token-boundary precision
(`дом` must not match `Домашний`), ё→е accent reach, OR/grouping determinism,
with `forbid_chunk_ids` precision guards.

## Consequences

### Positive

- Lexeme- and collocation-level search works for inflected Russian today,
  offline, with full explainability and adapter pins.
- Operator semantics are deterministic and covered by golden precision guards.
- The language adapter path is proven end-to-end for external `context-lang-*`
  repositories (registry + harness + runtime wiring).

### Negative

- Rule-based morphology over-generates for exceptional stems (stem
  alternations, mobile vowels); `RejectExp` and confidence floors mitigate,
  dictionary adapters fix properly.
- Operator term matching scans the in-memory chunk index client-side; large
  corpora will need a posting-list or FTS pushdown behind the same leaf
  contract (measured blocker → follow-up ADR).

### Follow-ups

- External `context-lang-ru` (dictionary-backed) may replace the in-repo rules
  behind the same registry entry; `pkg/lang/ru` then becomes the reference.
- Proximity (`NEAR/n`) and field syntax stay out until a measured product
  query requires them.
