# ADR-0032: Index Lifecycle Explain Report

Status: Accepted  
Date: 2026-07-14  
Related: [0021](0021-snapshot-commit-failure-semantics.md),
[0028](0028-source-tombstones.md),
[0026](0026-public-api-v1-freeze.md),
stabilization gap **C1** (lifecycle explain slice)

## Context

`SnapshotStatus` (building|ready|failed|superseded) must stay narrow for commit
semantics (ADR-0021). Consumers still need to know whether search is safe during
rebuild, when `last_failed` is retained, or when sources are tombstoned—without
silent drift or expanding the commit state machine.

## Decision

1. Add `lifecycle.Phase` orthogonal to `SnapshotStatus`: `empty`, `ready`,
   `degraded`, `rebuilding`, `failed`.
2. `lifecycle.Evaluate` emits stable `reasons` codes and `search_available`.
3. Rebuild/retry persist `state.index_op` for the duration of the op; **rebuild
   does not clear or flip `active_snapshot_id`**, so search stays available on
   the prior ready snapshot.
4. Degraded covers retained `last_failed` and/or tombstoned sources while the
   active snapshot remains `ready`.
5. Additive surface: CLI `index-status`, `GET /v1/index`, `contextkit.IndexStatus`.

## Consequences

### Positive

- Lab can show honest index health without archaeology.
- Rebuild/reindex exit criterion for S1 is met: search available or explicitly
  unavailable with reasons.

### Negative

- Dense/sparse backend staleness after snapshot move is not auto-detected; ops
  still use repair when backends are enabled.

### Follow-ups

- Optional stale detection via backend capability probes.
