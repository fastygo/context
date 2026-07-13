# Context API v1 (Lab / BFF)

Status: **frozen** for additive evolution ([ADR-0026](decisions/0026-public-api-v1-freeze.md)).  
Transport: HTTP+JSON (`cmd/context-serve`) and Go client (`pkg/contextkit`).

`api_version` = `v1`. Servers set header `X-Context-API-Version: v1`.

## Endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/health` | Liveness + `api_version` |
| GET | `/v1/status` | Workspace ingest status (no host paths) |
| GET | `/v1/metrics` | Counts + last eval + `quota` + `has_last_failed` |
| GET | `/v1/quota` | Soft project quota status (`allow`/`ask`/`deny`) |
| POST | `/v1/search` | Retrieval candidates |
| POST | `/v1/context-pack` | Build ContextPack |
| POST | `/v1/agent-run` | Fake/swappable agent loop |
| GET | `/v1/trace` | Run + events |
| PUT | `/v1/focus` | Upsert FocusProfile |
| GET | `/v1/focus` | Get FocusProfile |
| GET | `/v1/focuses` | List FocusProfiles |
| POST | `/v1/eval` | Golden suite (+ history append) |
| GET | `/v1/eval/history` | Eval history records |
| POST | `/v1/repair` | Index rebuild / retry-failed |
| POST | `/v1/inspect` | Explain search/pack (Lab inspector) |
| POST | `/v1/ingest` | Ingest by relative `path_key` |

## Soft quotas (Chunk 28)

Env (0/unset = unlimited): `CONTEXT_QUOTA_MAX_CHUNKS`, `CONTEXT_QUOTA_MAX_PACKS`,
`CONTEXT_QUOTA_MAX_RUNS`, optional `CONTEXT_QUOTA_SOFT_ASK_PERCENT` (default 80).

- Soft threshold → decision `ask` (advisory; writes still allowed).
- Hard limit (`used >= max`) → decision `deny`; ingest / context-pack / agent-run
  return permission error (HTTP 403). No billing.

## Compatibility

- Documented JSON fields: do not rename/remove within v1.
- Optional new fields and routes are allowed.
- `project_id` mismatch → permission error (HTTP 403), not silent widen
  ([ADR-0025](decisions/0025-multi-tenant-isolation.md)).

## Client

```go
import "github.com/fastygo/context/pkg/contextkit"

// contextkit.APIVersion == "v1"
cli := &contextkit.Client{BaseURL: "http://127.0.0.1:8080"}
```

Curl examples: [local-server.md](local-server.md).
