# ADR-0014: Storage Role Separation

Status: Accepted  
Date: 2026-06-17  
Related: [0006](0006-trace-event-append-only-replay.md), [0008](0008-hybrid-index-architecture.md), [0013](0013-context-ref-and-path-alias.md)

## Context

Analysis of Cursor storage (see `.project/.draft/cursor-storage-inventory.md`)
identified eight roles that must not be collapsed: source corpus, policy/workflow,
session replay, runtime spill, semantic index, keyword index, sync manifest, and
optional cloud execution. Mixing them causes audit failure, token bloat, and
incorrect retrieval scope.

## Decision

Map each role to Context core storage explicitly:

| Role | Cursor analogue | Context store | Never |
|------|-----------------|---------------|-------|
| Source corpus | repo files | Artifact store + git adapter | In vector DB |
| Policy/workflow | `.cursor/rules`, skills | PolicySnapshot files + metadata | Indexed as RAG corpus by default |
| Session replay | `state.vscdb` | Postgres/SQLite trace (ADR-0006) | QDrant |
| Runtime spill | agent-tools, terminals | Artifact store + slice API | Full body in prompt |
| Semantic index | Turbopuffer | QDrant via VectorNamespace | Chat history |
| Keyword/sparse | trigram + ripgrep | Tantivy sidecar + local grep adapter | Session DB |
| Sync manifest | Merkle client/server | Manifest chain + Merkle (ADR-0011) | SQLite KV hack |
| Cloud execution | Cloud Agent VM | AgentRun background (P2) | Core retrieval SoT |

### Design rules (normative)

1. Never store vectors in the session store.
2. Never treat chat history as the retrieval corpus.
3. Policy files ≠ indexed corpus unless explicitly ingested as sources.
4. Long outputs → artifacts; packs pull excerpts only.
5. Source/artifact store is text authority; indexes hold coordinates + embeddings.
6. Every model call gets a replayable `ContextPack`.

## Consequences

### Positive

- Clear package boundaries for `internal/indexing`, `internal/agentruntime`,
  `internal/artifacts`, `internal/retrieval`.
- LingvY-grade exact lookup stays on sparse/exact path, not session memory.

### Negative

- More moving parts than a single SQLite database; acceptable for scalability
  and audit requirements.

### Follow-ups

- Keep inventory draft updated when new Cursor behaviors are discovered; ADRs
  change only when Context policy changes.
