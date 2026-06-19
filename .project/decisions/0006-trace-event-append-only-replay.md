# ADR-0006: Trace Events — Append-Only Replay

Status: Accepted  
Date: 2026-06-17  
Related: [0014](0014-storage-role-separation.md)

## Context

Cursor stores per-turn prompt snapshots in SQLite KV (`messageRequestContext`).
The Context core needs an equivalent that is inspectable, project-scoped, and
not conflated with the vector index. Background agents require ownership,
cancellation, and parent/subagent linkage.

## Decision

1. **`AgentRun`:** append-only event log per run (`run_id`, parent, status,
   policy snapshot ref, timestamps).
2. **`ContextPack` snapshot:** persisted per model/tool/subagent step with
   selected evidence refs, budgets, and rejection reasons — equivalent to
   Cursor's per-message context snapshot.
3. **Storage:** Postgres (or SQLite PoC); **never** QDrant or Tantivy.
4. Events are immutable after append; corrections are new events, not overwrites.
5. Session replay is a **product/adapter concern** for chat UI; core exposes
   generic run + pack APIs.

## Consequences

### Positive

- Debuggable retrieval and packing decisions.
- Clear separation from semantic index (ADR-0014).

### Negative

- Storage growth; retention policy deferred to product/tenant config.

### Follow-ups

- Event schema in Plan Chunk 02 domain models; bubble-level mapping optional in
  product layer.
