# Chunk 14 — Postgres FTS lexical limits

Recorded against live compose Postgres (`CONTEXT_SPARSE_KIND=postgres_fts`).

| Dimension | Fake term-overlap | Postgres FTS (`simple`) | Gate |
| --- | --- | --- | --- |
| Offline / CI without DB | Yes | Skip unless `CONTEXT_PG_DSN` | Keep fake default |
| Project + snapshot isolation | Index-scoped | Server `WHERE` + PK | OK |
| Morphology / stemming | None | None (`simple`) | Needs `context-lang-*` or lang-specific FTS configs |
| Phrase / proximity | Substring-ish overlap | `plainto_tsquery` AND semantics | Needs richer query DSL or `context-sparse` |
| Ranking | Token overlap count | `ts_rank_cd` | BM25-style weights → later |
| Language filter | Client index | Client index (`SupportsMetadataFilter=false`) | Same as pgvector PoC |

Trigger for Tantivy/`context-sparse`: measured failure on morphology, phrase
search, or ranking quality that FTS cannot close without forking domain ports.
