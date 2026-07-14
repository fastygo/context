# CLI reference (`context-dev`)

```bash
go run ./cmd/context-dev <command> [flags]
```

All mutating commands that touch a workspace need `--data <dir>`. Most also
need `--project <id>` matching the workspace project.

## Commands

| Command | Purpose |
| --- | --- |
| `init-project` | Create workspace + project under `--data` from `--root` corpus |
| `ingest` | Index corpus (optional `--path`); optional dense/FTS via env |
| `search` | Retrieve candidates (`--mode exact\|sparse\|hybrid\|dense\|hybrid-dense`) |
| `context-pack` | Build ContextPack from hybrid search |
| `inspect` | Lab inspector JSON (`--query` and/or `--pack`) |
| `agent-run` | Foreground agent loop |
| `trace` | Load run + events (`--run`) |
| `focus-put` / `focus-get` / `focus-list` | FocusProfile CRUD |
| `job-start` / `job-status` / `job-list` / `job-cancel` | Background AgentRun |
| `metrics` | Workspace counters + readiness + quota |
| `quota` | Soft quota status |
| `ready` | Backend readiness probe |
| `eval` / `eval-history` | Golden suite + JSONL history |
| `repair` | Index rebuild / retry-failed |
| `tombstone-source` | Soft-delete a source (`--source`); chunks leave search/pack |
| `snapshot-export` | Write ready snapshot bundle (`--out`) |
| `snapshot-import` | Verify bundle (`--in`); optional `--activate` |
| `project-export` | Project archive (`--out`; snapshot + focuses) |
| `project-delete` | Wipe project (`--confirm` must match `--project`) |
| `meta-check` | Metadata backend check |
| `proof-run` | Regenerate [`.proofs/`](../.proofs/) artifacts |

## Common flags

| Flag | Used by | Notes |
| --- | --- | --- |
| `--data` | most | Workspace root |
| `--project` | most | Must match workspace `project_id` |
| `--query` | search, pack, agent, inspect, jobs | |
| `--mode` | search | Default hybrid |
| `--focus` | search/pack/agent | FocusProfile id |
| `--owner` | job-start | **Required** for background jobs |
| `--job` | job-status/cancel | |
| `--path` | ingest | File or directory under corpus |
| `--out` | eval / proof-run | Default under `.proofs/` |

## Important env vars

| Variable | Effect |
| --- | --- |
| `CONTEXT_COMPLETER_KIND` | `fake` \| `localecho` \| `http` |
| `CONTEXT_EMBEDDER_KIND` | `fake` \| `local_hash` \| `http` |
| `CONTEXT_ENABLE_DENSE` | `1` upsert/search dense |
| `CONTEXT_SPARSE_KIND` | `postgres_fts` for live FTS |
| `CONTEXT_METADATA_KIND` | `postgres` for durable metadata |
| `CONTEXT_REDACT` | default on; `0` disables |
| `CONTEXT_QUOTA_MAX_*` | Soft project caps |
| `CONTEXT_FAIL_*` | Failure injection |
| `CONTEXT_PG_DSN` | Postgres connection |

Full ops detail: [operations/local-server.md](operations/local-server.md).

## HTTP companion

Long-lived jobs and Lab traffic: `go run ./cmd/context-serve --data …`  
→ [api/v1.md](api/v1.md).
