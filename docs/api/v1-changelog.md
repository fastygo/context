# API v1 additive changelog

Status: living log under [ADR-0026](../decisions/0026-public-api-v1-freeze.md).  
Breaking changes require a new major (`v2`). This file only records **additive**
surface since Lab Gate (2026-07-13).

## Lab Gate baseline (Chunks 20–31)

See [lab-gate.md](../lab-gate.md) and [api/v1.md](v1.md) endpoint table for the
frozen Lab smoke path (`health` … `jobs`).

## Stabilization additives (S1–S4)

| When | Surface | ADR |
| --- | --- | --- |
| S1 / C1 | `POST /v1/sources/tombstone`, CLI `tombstone-source` | [0028](../decisions/0028-source-tombstones.md) |
| S1 / C1 | `GET /v1/index`, CLI `index-status` (`phase`, `search_available`) | [0032](../decisions/0032-index-lifecycle-explain.md) |
| S1 / C2 | `POST /v1/snapshot/export\|import` | [0029](../decisions/0029-snapshot-bundle-export-import.md) |
| S1 / C7 | `POST /v1/project/export\|delete` | [0030](../decisions/0030-project-export-delete.md) |
| S1 / C8 | `PUT/GET/DELETE /v1/schedules*`, `tick`, `fire` | [0031](../decisions/0031-durable-schedule-port.md) |
| S2 / C4 | Search candidate optional `snippet` object | [0033](../decisions/0033-offset-stable-snippets.md) |
| S2 / C6 | Tool status `needs_approval` on `ask` (agent-run tool_call) | [0034](../decisions/0034-tool-side-effect-approval.md) |
| S3 | No new HTTP routes; public `pkg/langtestkit` + thin adapters | [0037](../decisions/0037-public-langtestkit.md)–[0039](../decisions/0039-s3-adapter-freeze-defer.md) |
| S4 | No new HTTP routes; graph/query forever-defer docs | [0040](../decisions/0040-graph-consumer-projection.md)–[0041](../decisions/0041-query-ast-defer-fts-filters.md) |

## Post-S5 additives

| When | Surface | ADR |
| --- | --- | --- |
| 2026-07-16 | `POST /v1/search`: `mode:"query"` (operators), optional request `lang`, optional response `query_explain`; CLI `--mode query --lang`; `contextkit` `SearchRequest.Lang` / `SearchResult.QueryExplain` | [0043](../decisions/0043-ru-adapter-operator-query-layer.md) |
| 2026-07-16 | In-repo language adapters: `pkg/lang/ru` (`context-lang-ru`), registry `en`/`ru`; hybrid mode honors expansion language | [0043](../decisions/0043-ru-adapter-operator-query-layer.md) |

## Client packages

| Package | Role |
| --- | --- |
| `pkg/contextkit` | HTTP Lab/BFF client (must not import `internal/`) |
| `pkg/langcontract` / `pkg/langtestkit` | External language-adapter contracts |

## Compatibility rules

1. New fields are optional / omitempty.
2. New routes are additive under `/v1/`.
3. Enum expansions document unknown-value handling at the consumer.
4. Removals or semantic flips require `v2` + migration note.
