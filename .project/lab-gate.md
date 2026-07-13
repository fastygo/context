# Lab Gate (Chunk 32)

Status: **passed** (2026-07-13)  
Related: [api-v1.md](api-v1.md), [ADR-0026](decisions/0026-public-api-v1-freeze.md),
[ADR-0027](decisions/0027-lab-gate-freeze.md), [ADR-0024](decisions/0024-thin-http-service-boundary.md)

This gate freezes the Core → Lab/BFF contract. Lab may bind to HTTP +
`pkg/contextkit` only. Context must not import Lab; Lab must not import
`internal/`.

## Consumer contract

| Rule | Detail |
| --- | --- |
| Transport | `cmd/context-serve` HTTP+JSON; optional `pkg/contextkit` |
| Version | `api_version=v1`; header `X-Context-API-Version: v1` |
| Identity | Always send `project_id`; mismatch → permission / 403 |
| Paths | Lab sees `path_key` only — never absolute host paths |
| Truth | Packs, traces, metrics, jobs are Core SoT; model text may be redacted |
| Auth | Optional shared token only (ADR-0024); full multi-tenant auth deferred |

## Checklist (Chunks 20–31)

| Capability | Chunk | Lab use |
| --- | --- | --- |
| Thin HTTP | 20 | Call `/v1/*` |
| Go client | 21 | Import `contextkit` only |
| Metrics / eval history | 22 | Ops panels |
| Index repair | 23 | Rebuild / retry-failed |
| Tenant isolation design | 24 | Bind allowed `project_id` set |
| API v1 freeze | 25 | Pin contract |
| Inspector | 26 | Explain search/pack |
| Completer / Embedder swap | 27 | Config, not Lab code |
| Soft quotas | 28 | Show allow/ask/deny |
| Ready / degraded | 29 | Health + explicit errors |
| Redaction | 30 | Safe model/preview text |
| Background jobs | 31 | Start/status/cancel AgentRun |

## Smoke path (automated)

Offline test `TestLabGateSmoke` exercises:

```text
health → ready → status → search → context-pack → inspect →
metrics → quota → agent-run (localecho) → job-start/status
```

Asserts: `api_version` / version header present; JSON responses contain no
Windows/Unix absolute path prefixes; job reaches `completed` with owner set.

Run:

```bash
go test ./internal/httpserver/ -run TestLabGateSmoke -count=1
```

## Deferred (not blocking Lab)

- Multi-tenant OIDC / membership ACL
- OpenAPI codegen
- `context-lang-*` / full morphology adapters
- Cron / event-triggered jobs, external queues
- Fuzzy/trigram, QDrant, Lab UI in this repository

## Lab responsibility

- Map UI identity → allowed `project_id` (and later `tenant_id`)
- Render inspector / metrics / quota / job status JSON
- Never treat Completer output as source truth
- Keep product/brand names in Lab config, not in Core
