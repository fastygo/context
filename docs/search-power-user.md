# Power-user search (filters + operator layer)

Status: supported path after S4 ([ADR-0041](decisions/0041-query-ast-defer-fts-filters.md));
extended by the minimal operator layer
([ADR-0043](decisions/0043-ru-adapter-operator-query-layer.md), mode `query`).

For operators (`"phrase"`, AND/OR/NOT, `~` morphology, `lang:`) see
[search-operators.md](search-operators.md). This page covers plain modes and
filters.

## Modes

| Mode | Behavior |
| --- | --- |
| `exact` | Case-sensitive phrase / substring over chunk text |
| `sparse` | Fake sparse offline; `postgresfts` when configured |
| `dense` / `hybrid-dense` | Requires pgvector |
| `hybrid` | Exact + sparse (+ dense if enabled); morph expansion with `lang`/`CONTEXT_LANG` |
| `query` | Operator layer with explain ([search-operators.md](search-operators.md)) |

CLI: `context-dev search --mode …`  
HTTP: `POST /v1/search` with `mode`.

## Filters (API / plan)

Use `RetrievalFilters` / focus constraints — not a DSL:

- language, sense, concept, attestation
- register / region / time_period / lexicon authority
- temporal half-open range (event corpora)
- trust via FocusProfile when packing

`GraphNodeID` is reserved and currently ignored ([ADR-0040](decisions/0040-graph-consumer-projection.md)).

## Consumer boolean UX

If the product UI needs `AND` / `OR` / `NOT`:

1. Parse in Lab/BFF.
2. Issue one or more `/v1/search` calls with concrete modes + filters.
3. Intersect/union `chunk_id`s in the consumer.
4. Or pass backend-native FTS syntax only into an ops-owned sparse path.

## Graph / edges

Citation and co-occurrence graphs stay **consumer projections** keyed by
public IDs from search/pack responses ([ADR-0040](decisions/0040-graph-consumer-projection.md)).
