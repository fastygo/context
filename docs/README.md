# Context documentation

User-facing docs for `github.com/fastygo/context` — a **brand-neutral context
operating core** (project memory → retrieval → ContextPack → tools/agents →
verification → replayable traces).

This tree is the primary guide for mid-level engineers and for LLMs helping
with integration. Planning-only material lives under [`.project/`](../.project/).

## Start here

| If you need… | Go to |
| --- | --- |
| First local run | [Getting started](getting-started.md) |
| What the engine is (and is not) | [Concepts](concepts.md) |
| End-to-end recipes | [Scenarios](scenarios/README.md) |
| CLI commands | [CLI reference](cli.md) |
| Search operators + morphology (`mode=query`, `lang:ru`) | [search-operators.md](search-operators.md) |
| Plain modes + filters | [search-power-user.md](search-power-user.md) |
| HTTP + `contextkit` | [API v1](api/v1.md) |
| Postgres / dense / FTS / env | [Local operations](operations/local-server.md) |
| Lab/BFF freeze checklist | [Lab gate](lab-gate.md) |
| Ops runbook (ingest/rebuild/restore/degraded) | [operations/runbook.md](operations/runbook.md) |
| API v1 additive changelog | [api/v1-changelog.md](api/v1-changelog.md) |
| Why a boundary exists | [ADRs / decisions](decisions/README.md) |
| Proof artifacts (JSON) | [`.proofs/`](../.proofs/README.md) |

## Capabilities map (today)

```text
ingest (corpus → chunks + optional dense/FTS)
  → search (exact | sparse | hybrid | dense* | query†)
  → context-pack / inspect
  → agent-run (foreground)  OR  jobs (background AgentRun)
  → trace / metrics / quota / ready
```

\* Dense needs Postgres/pgvector (`CONTEXT_ENABLE_DENSE=1`).  
† Operator layer with morphology (`"phrase"`, AND/OR/NOT, `~term`, `lang:ru`) —
see [search-operators.md](search-operators.md) (ADR-0043).

| Surface | Entry |
| --- | --- |
| CLI | `go run ./cmd/context-dev …` — see [cli.md](cli.md) |
| HTTP | `go run ./cmd/context-serve --data …` — see [api/v1.md](api/v1.md) |
| Go client | `github.com/fastygo/context/pkg/contextkit` |
| Language adapter harness | `pkg/langtestkit` + `pkg/langcontract` |
| Russian morphology engine | `pkg/lang/ru` (`context-lang-ru`, rule-based) |

**Do not** import `internal/` from Lab or products. **Do not** treat model text
as source truth (redaction applies to Lab-visible surfaces).

## Scenario index

1. [Ingest → search → pack](scenarios/ingest-search-pack.md)
2. [Agent run + trace](scenarios/agent-run.md)
3. [Background jobs](scenarios/background-jobs.md)
4. [Lab / BFF consumer](scenarios/lab-bff.md)
5. [Ops: quotas, readiness, repair](scenarios/ops.md)

## Planning vs shipped docs

| Location | Contents |
| --- | --- |
| **`docs/`** (here) | How to use and integrate the shipped core |
| **`.project/`** | Plugins, drafts, future roadmaps only |
| **`.proofs/`** | Measured PoC / eval JSON artifacts |

Deferred work (auth, OpenAPI, richer lang engines, fuzzy): see
[`.project/future-layer.md`](../.project/future-layer.md) and
[`.project/adapters-backlog.md`](../.project/adapters-backlog.md).
Thin S3 adapters + public langtestkit are shipped (ADR-0037–0039).
