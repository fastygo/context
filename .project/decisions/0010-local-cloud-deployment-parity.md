# ADR-0010: Local and Cloud Deployment Parity

Status: Accepted  
Date: 2026-06-17  
Related: [0009](0009-context-sparse-tantivy-sidecar.md), [0012](0012-index-snapshot-sync-replication.md)

## Context

The product must support the same retrieval and indexing use case on a developer
PC and in cloud (BuildY BFF, team sync). Divergent code paths ("SQLite FTS locally,
OpenSearch in cloud") break identical search guarantees and testability.

## Decision

1. **One logical service graph** in both environments:

   ```text
   Product UI / CLI
        → context-core (Go)
             → context-sparse (Tantivy)
             → QDrant
             → Postgres (metadata)
             → object store / local artifact dir
   ```

2. **Local:** `docker compose` for QDrant + `context-sparse`; `context-core` as
   native binary or container; data under `~/.context/` or project `.context/`.

3. **Cloud:** same container images and env contract; Postgres + S3/MinIO;
   QDrant single node or cluster.

4. **Configuration differs only by endpoints and credentials**, not by retrieval
   algorithms or manifest schema.

5. **Search contract:** all queries include `project_id`; default to
   `active_snapshot_id` from manifest unless a pinned snapshot is requested for
   replay/eval.

## Consequences

### Positive

- Integration tests run against compose stack locally and in CI.
- Users get offline-capable desktop when indexes are replicated (ADR-0012).

### Negative

- Docker dependency for full stack; document minimal "metadata-only" dev mode
  separately if needed.

### Follow-ups

- `docker-compose.yml` in Context repo (Plan Chunk after storage adapters).
- Health checks: core refuses search if sparse/dense revision ≠ manifest.
