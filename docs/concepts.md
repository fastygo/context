# Concepts

## What this is

A **context operating core**: deterministic project memory, hybrid retrieval,
source-backed ContextPacks, typed tool/model steps, verification, and
replayable traces.

It is **not** a chat app, RAG framework wrapper, or Lab UI.

Pipeline (normative):

```text
deterministic project memory
 → precise indexing
 → hybrid retrieval
 → source-backed ContextPack
 → typed tool/model/subagent step
 → verification
 → replayable trace
```

Lab Gate and Stabilization Gate are **passed** — integrate via HTTP /
`pkg/contextkit` only ([lab-gate.md](lab-gate.md), ADR-0042).

## Isolation

- **Project** is the isolation boundary for indexes, packs, runs, jobs, and
  schedules.
- Callers always pass `project_id`. Mismatch → permission error (HTTP 403).
- Optional `TenantID` exists for future ACL/billing; single-process serve is
  still one workspace (`--data`).

## Evidence and truth

- Chunks carry spans, checksums, and version pins.
- ContextPack selects budgeted evidence with explainable scores.
- Derived structured outputs may carry `ArtifactLineage` (many inputs) without
  relying on an `AgentRun` trace alone.
- Event/log sources may carry neutral `TemporalRange` metadata; corpus events
  are not runtime `tracing.Event` records.
- Completer/LLM output may summarize; it is not source truth.
- Redaction (`CONTEXT_REDACT`, default on) strips secrets/PII from
  `model_text` and inspect `surface_preview` — not from the corpus index.

## Modes of work

| Mode | Meaning |
| --- | --- |
| Foreground `agent-run` | Synchronous pack → model → tool → verify |
| Background `jobs` | Same AgentRun path in-process with owner + cancel |
| Schedules | Durable `once_at` / `interval` / `event` → job registry |
| Search modes | `exact`, `sparse`, `hybrid`, `dense`, `hybrid-dense`, `query` |

`mode=query` enables the operator layer (phrases, AND/OR/NOT, morphology
`~term`, `lang:`) — see [search-operators.md](search-operators.md) (ADR-0043).
Lexicographic `TimePeriod` is not event/log time.

## Adapters (config, not hardcoding)

| Role | Env examples |
| --- | --- |
| Embedder | `CONTEXT_EMBEDDER_KIND=fake\|local_hash\|http` |
| Completer | `CONTEXT_COMPLETER_KIND=fake\|localecho\|http` |
| Metadata | `CONTEXT_METADATA_KIND=memory\|postgres` |
| Sparse | default memory/fake; `CONTEXT_SPARSE_KIND=postgres_fts` |
| Dense | `CONTEXT_ENABLE_DENSE=1` + pgvector |
| Language | registry `en` / `ru` (`pkg/lang/ru`); external via `pkg/langtestkit` |

## Related

- [CLI](cli.md) · [API v1](api/v1.md) · [ADRs](decisions/README.md)
- Brand-neutral rule: product names belong in Lab/config, not in core packages.
