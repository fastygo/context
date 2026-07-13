# ADR-0025: Multi-Tenant Isolation Boundary

Status: Accepted  
Date: 2026-07-13  
Related: [0001](0001-package-boundary-internal-first.md),
[0004](0004-vector-namespace-abstraction.md),
[0011](0011-merkle-manifest-and-snapshot-namespace.md),
[0021](0021-snapshot-commit-failure-semantics.md),
[0024](0024-thin-http-service-boundary.md)

## Context

Phase 3 requires a multi-tenant isolation **design** before production auth,
quotas, or billing. ADR-0011 already maps `ProjectID` to “tenant ACL, quota,
policy,” which conflates two layers: the **tenant** (billing/ACL root) and the
**project** (index, artifact, and run boundary). Today `context-serve` binds one
process-local `--data` workspace; Lab/BFF may host many projects later.

## Decision

### 1. Hierarchy

| Layer | ID | Role |
|-------|-----|------|
| Tenant | `TenantID` (optional on `Project`) | Outer ACL, quota, retention, billing root |
| Project | `ProjectID` (required) | Index, artifact, pack, run, and search boundary |
| Snapshot | `SnapshotID` | Immutable index generation inside a project |

1. A tenant may own many projects. A project belongs to at most one tenant.
2. Until auth lands, `TenantID` may be empty; isolation still requires
   `ProjectID` on every storage and retrieval operation.
3. Do **not** map `Project` to a person. Users/service accounts are identity
   subjects in a later ACL layer (future-layer Layer 02).

### 2. Hard isolation rules (now)

1. Every metadata, vector, sparse, artifact, pack, run, and trace read/write is
   keyed by `project_id` (and `snapshot_id` where applicable).
2. Search, context-pack, agent-run, repair, and metrics must reject a request
   `project_id` that does not match the bound workspace/project
   (`permission` / `validation` — never silently widen).
3. Cross-project retrieval is **forbidden** in core APIs. Any future
   cross-project feature requires an explicit audited API and a superseding ADR.
4. Shared vector collections (ADR-0004) remain allowed only with mandatory
   payload/`WHERE` filters on `project_id` + `snapshot_id`.

### 3. Auth and quotas (deferred)

1. Multi-tenant **authentication** (OIDC, API keys per tenant, membership) stays
   out of Chunk 24. Optional process shared-secret (ADR-0024) is not tenancy.
2. **Quotas** (chunks, embeds, runs) are designed as tenant- or project-scoped
   counters read from metrics/ops; enforcement hooks land in a later chunk.
   Soft design: deny or `ask` when over quota **outside** the model
   (`policy` package).
3. BFF/Lab binds caller → allowed `project_id` set; Context trusts the
   `project_id` on the wire only after that binding exists.

### 4. Process topology

1. `context-serve --data` remains a **single-workspace** process for local/dev.
2. Multi-tenant hosting runs one process per tenant/project **or** a future
   router that selects a store namespace per `project_id` — not an in-process
   merge of corpora.
3. `pkg/contextkit` clients always send `project_id`; servers never infer it
   from host paths.

## Consequences

### Positive

- Clear split between tenant (policy/billing) and project (index isolation).
- Existing `project_id` filters become the normative isolation contract.
- Auth and quota work can proceed without rewriting retrieval ports.

### Negative

- Optional `TenantID` is unused until a consumer sets it.
- Single-process serve does not multiplex tenants; operators must not point one
  `--data` at mixed-tenant corpora.

### Follow-ups

- Membership/ACL checks when auth lands (future-layer Layer 02).
- Quota enforcement + soft limits using ops metrics.
- Contract tests remain mandatory for any new store/retriever adapter.

## Contract tests (Chunk 24)

Adapters and in-process indexes must prove:

1. Chunks written under `project_a` are invisible to `project_b` list/get/search.
2. Request `project_id` mismatch against the bound workspace returns an error,
   not another project’s data.
