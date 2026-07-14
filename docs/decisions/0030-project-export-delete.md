# ADR-0030: Project Export And Delete Retention Hooks

Status: Accepted  
Date: 2026-07-14  
Related: [0028](0028-source-tombstones.md),
[0029](0029-snapshot-bundle-export-import.md),
[0026](0026-public-api-v1-freeze.md),
stabilization gap **C7**

## Context

Long-lived corpora need an explicit governance boundary: export project evidence
for portability/compliance, and delete project data so search cannot resurrect
withdrawn corpora. Full retention policies and encryption remain future-layer
L13; Stabilization Gate needs minimal hooks now.

## Decision

1. **Export** writes `project-archive-v1`: sealed snapshot bundle (ADR-0029) plus
   FocusProfiles. No host `CorpusRoot`, no runs/traces (minimize secret surface).
2. **Delete** requires `confirm_project_id == project_id`.
3. Delete order:
   1. tombstone all workspace sources (hide evidence if later steps fail);
   2. `MetadataStore.DeleteProject` (Postgres CASCADE / memory purge);
   3. `ArtifactStore.DeleteProject` for localfs project tree;
   4. remove workspace `state.json` so search cannot load the project.
4. Additive surfaces: CLI `project-export` / `project-delete`, HTTP
   `POST /v1/project/export|delete`, `contextkit` helpers.
5. Dense/FTS backend rows are best-effort outside this ADR; ops may rebuild or
   GC vectors by `project_id` separately when those backends are enabled.

## Consequences

### Positive

- Governance path without waiting for full retention policy engine.
- Confirm latch reduces accidental wipe.

### Negative

- Single-project workspace model: delete clears the whole `--data` state file.
- Vector/FTS orphans possible until backend GC.

### Follow-ups

- Per-artifact retention TTLs (L13).
- Multi-project metadata host with selective workspace files.
- Dense/sparse delete-by-project in adapters.
