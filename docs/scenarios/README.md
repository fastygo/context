# Scenarios

Copy-paste recipes for common integrations. Each scenario lists CLI and HTTP
(and `contextkit` where useful).

| Scenario | Outcome |
| --- | --- |
| [Ingest → search → pack](ingest-search-pack.md) | Evidence-backed ContextPack |
| [Agent run + trace](agent-run.md) | Completer + tool + verify + trace |
| [Background jobs](background-jobs.md) | Async AgentRun with cancel |
| [Lab / BFF](lab-bff.md) | Bind without importing `internal/` |
| [Ops](ops.md) | Quotas, readiness, repair, metrics |

Assumes a workspace created as in [getting-started.md](../getting-started.md)
(`--data` / `--project`).
