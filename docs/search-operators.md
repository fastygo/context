# Operator search and morphology (mode=query)

Status: shipped ([ADR-0043](decisions/0043-ru-adapter-operator-query-layer.md)).
For plain modes and filters see [search-power-user.md](search-power-user.md).

Mode `query` adds a small deterministic operator language on top of the
existing retrieval paths. There is no hidden rewrite: the parsed tree,
every accepted/rejected morphology expansion, and per-leaf match counts are
returned in `query_explain` and recorded as trace events.

## Operators

| Syntax | Meaning |
| --- | --- |
| `память проект` | implicit AND; each term matches on **token boundaries** |
| `a OR b`, `a \| b` | union |
| `a NOT b`, `a -b` | exclusion (needs at least one positive term) |
| `( … )` | grouping |
| `"память проекта"` | literal phrase (substring, case-sensitive) |
| `~дорога` | morphology-expanded term |
| `~"железная дорога"` | lemma-sequence phrase: matches inflected word combinations |
| `lang:ru` | selects the language adapter for this query |

Token-boundary matters: in mode `query` the term `дом` does **not** match
`Домашний` (in `exact`/`hybrid` substring modes it would).

## Morphology

Language adapters are selected per query: in-query `lang:` wins, then the
`lang` request field / `--lang` flag, then `CONTEXT_LANG`. In-repo adapters:

| Tag | Adapter | Behavior |
| --- | --- | --- |
| `ru` | `context-lang-ru` (`pkg/lang/ru`) | Rule-based declension/conjugation paradigms, explicit ambiguity, ё→е fold, exception fixture |
| `en` | `context-lang-en` | Light lemma + fixture expansion |

With `lang:ru`, the query `дорога` also matches `дороги`, `дороге`,
`дорогу`, `дорогой`… and `~"железная дорога"` matches «вдоль железной
дороги». Expansions are bounded, explainable (type/reason/confidence), and
can be suppressed via reject lists.

When no adapter matches the language, matching degrades to case/NFC-folded
token equality and `~` markers add nothing — never an error.

## Surfaces

```bash
# CLI
go run ./cmd/context-dev search --data "$DATA" --project demo \
  --mode query --lang ru --query '~"железная дорога" -чат'

# HTTP (additive v1)
curl -s -X POST http://127.0.0.1:8080/v1/search -d '{
  "project_id": "demo",
  "query": "дорога -чат lang:ru",
  "mode": "query"
}'
```

`pkg/contextkit`: set `SearchRequest.Mode="query"` (+ optional `Lang`); read
`SearchResult.QueryExplain`.

## Explain shape

```json
{
  "query": "дорога -чат",
  "canonical": "(AND дорога (NOT чат))",
  "language": "ru",
  "adapter_id": "context-lang-ru",
  "leaves": [
    {
      "kind": "term",
      "text": "дорога",
      "expansions": ["дороги", "дороге", "дорогу", "дорогой"],
      "retrievers": ["term", "sparse"],
      "matches": 2
    }
  ]
}
```

## Semantics and limits

- Scoring reuses ADR-0019 contracts; new reason codes: `token_term`,
  `morph_phrase`. Sparse/FTS only reinforces token hits — it cannot add
  chunks the deterministic layer did not match.
- Field/facet filtering stays in `RetrievalFilters` (no field syntax in the
  grammar).
- Quoted phrases are literal by design; use `~"…"` for inflection-aware
  phrases.
- Rule-based Russian morphology over-generates on exceptional stems; golden
  precision guards (`eval-golden-v3`) and reject lists control false
  expansions. Dictionary-backed `context-lang-ru` can replace the rules
  behind the same registry entry.
- Pure-negative queries (`-чат`, `NOT чат`) are validation errors.
