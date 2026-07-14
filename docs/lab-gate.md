# Lab Gate

Status: **passed** (2026-07-13)  
Stabilization Gate: **passed** (2026-07-14) — [ADR-0042](decisions/0042-stabilization-gate.md)  
Related: [API v1](api/v1.md), [v1 changelog](api/v1-changelog.md),
[ADR-0026](decisions/0026-public-api-v1-freeze.md),
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

## Stabilization additives (S1–S4, still v1)

Lab may also use these additive routes/fields (see
[v1-changelog.md](api/v1-changelog.md)):

| Area | Surface |
| --- | --- |
| Index health | `GET /v1/index` |
| Tombstones | `POST /v1/sources/tombstone` |
| Snapshot move | `POST /v1/snapshot/export\|import` |
| Project retention | `POST /v1/project/export\|delete` |
| Schedules | `/v1/schedules*` |
| Citations | search candidate optional `snippet` |
| Tool policy | tool_call status `needs_approval` |

Language adapters outside this repo use `pkg/langtestkit` (not `internal/`).

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
go test ./... -count=1
go test ./internal/evals/golden/ ./internal/evals/adversarial/ -count=1
```

## Deferred past Stabilization Gate (frozen)

Reopen only with measured blocker + ADR ([ADR-0042](decisions/0042-stabilization-gate.md)):

- Multi-tenant OIDC / membership ACL
- OpenAPI codegen / gRPC
- QDrant / Turbopuffer / Tantivy `context-sparse`
- Distributed workers / leases / DLQ (beyond single-node scheduler)
- In-core graph store / Query AST (consumer patterns: ADR-0040/0041)
- Object-store ArtifactStore, DOCX, fuzzy/`pg_trgm` in core (ADR-0039)
- OCR / spreadsheet / mailbox / crawler governance
- Billing / cost accounting; Lab UI inside this repository

Thin `context-lang-en`, curated JSON lexicon, HTML/PDF, NDJSON events, and
public langtestkit are **shipped** (S3) — richer engines stay external.

## Lab responsibility

- Map UI identity → allowed `project_id` (and later `tenant_id`)
- Render inspector / metrics / quota / job / index-status JSON
- Never treat Completer output as source truth
- Keep product/brand names in Lab config, not in Core
- Compose boolean/graph UX in Lab using search filters + consumer stores
