# ADR-0031: Durable Schedule Port For Background Jobs

Status: Accepted  
Date: 2026-07-14  
Related: [0024](0024-thin-http-service-boundary.md),
[0026](0026-public-api-v1-freeze.md),
stabilization gap **C8**

## Context

In-process `jobs.Registry` (Chunk 31) loses pending/running work on process
exit. Stabilization needs a **schedule port** so triggers survive restart
without implementing distributed workers/leases/DLQ (future-layer L08).

## Decision

1. `scheduler.Store` is the replaceable port. First adapter: `FileStore` under
   `ops/schedules/*.json`.
2. Schedule kinds: `once_at`, `interval`, `event`.
3. Time-based due schedules are fired by `scheduler.Tick` → enqueues
   `jobs.Registry.Start`. Event schedules fire via `FireEvent`.
4. **Durability model:** schedule definitions and `next_run_at` survive restart;
   mid-flight AgentRun does **not**. After restart, call Tick (CLI/HTTP or
   `context-serve` startup) to enqueue overdue work as **new** jobs.
5. Owner + query required on every schedule (same policy as background jobs).
6. Additive API: `PUT/GET/DELETE /v1/schedules`, `POST /v1/schedules/tick`,
   `POST /v1/schedules/fire`; CLI `schedule-*`.

## Consequences

### Positive

- Cron/file-style automation without core distributed queue.
- Clear boundary for future queue/lease adapters behind the same port.

### Negative

- Not multi-worker safe (single-node file store; Tick is best-effort).
- Lost in-flight runs still mark as failed on job Open (Chunk 31 behavior).

### Follow-ups

- Distributed job control (L08): leases, DLQ, heartbeats.
- Optional periodic ticker interval config beyond serve-start Tick.
