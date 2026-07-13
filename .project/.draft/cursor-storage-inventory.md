# Storage Roles Inventory → Context Work Scope

Status: draft reference  
Purpose: map storage roles (inspired by modern coding-agent layouts) onto
`github.com/fastygo/context`, and show what Generative UX needs from those roles.  
Privacy/training modes are out of scope. This is a **storage topology** note.

Sources: public Cursor docs (agent, plan mode, skills, hooks, indexing),
Turbopuffer case study, community reverse-engineering. Treat IDE KV key families
as best-effort until vendors publish schemas.

Related: [generative-ux-from-context.md](./generative-ux-from-context.md),
`../roadmap-context-core.md`, `../progress.md`, ADR-0017 (PoC backend order).

---

## Executive summary

A serious agent stack is not one database. It is **eight storage roles** that
must stay separate:

| # | Role | Typical location (reference) | Primary purpose |
|---|------|------------------------------|-----------------|
| 1 | **Source corpus** | Project files on disk | Ground truth for code/text/data |
| 2 | **Policy & workflow** | Repo config (rules, skills, plans) | Behavior constraints |
| 3 | **Session replay** | Local session KV / DB | Turns, tool calls, prompt snapshots |
| 4 | **Runtime spill** | Per-workspace spill files | Long tool/terminal outputs |
| 5 | **Semantic index** | Vector store | Similarity recall |
| 6 | **Keyword / exact index** | Sparse index + local scan | Symbols, phrases, exact facts |
| 7 | **Sync manifest** | Client/server hash trees | Incremental re-index |
| 8 | **Remote execution** (optional) | Isolated VM / workers | Build, test, preview artifacts |

`fastygo/context` must cover **1–4 and 7** in a project-scoped, auditable way,
plus **5–6** via adapters. Role 3 maps to **`ContextPack` + `AgentRun` traces**,
not to the vector layer.

**PoC backend order (ADR-0017):** filesystem artifacts → memory then PostgreSQL
metadata → `VectorStore` port with **PostgreSQL + pgvector first** → QDrant later.
Do not treat a remote vendor vector DB as the PoC default.

**Generative UX note:** forming a UX/screen/brick spec is a **structured
artifact + tool/model step** over roles 1–4. It is not “stuff the chat into a
vector DB.” See the bridge draft.

---

## Layer diagram

```text
                         USER / REPO
                              │
         ┌────────────────────┼────────────────────┐
         │                    │                    │
         v                    v                    v
   [1] Source files     [2] Policy / rules    [3] Session store
   working tree         skills, plans,         AgentRun + ContextPack
   .project/, docs      tool policies          snapshots, tool events
         │                    │                    │
         │                    │                    v
         │                    │              [4] Spill artifacts
         │                    │                  long tool/terminal blobs
         │                    │                    │
         └──────────────┬─────┴────────────────────┘
                        │
            indexing & retrieval
                        │
         ┌──────────────┼──────────────┐
         v              v              v
   [5] Vector index  [6] Sparse/exact [7] Manifest
   (adapter)         (adapter)        checksum / Merkle
         │              │              │
         └──────────────┴──────────────┘
                        │
                        v
              ContextPack → model/tool step → verifier → trace
                        │
                        v
              [8] Remote workers (optional)
                  preview build, sandbox, handoff artifacts
```

---

## 1. Source corpus

| What | Where | Used for |
|------|-------|----------|
| Application / content sources | Repo or registered paths | Edits, preview, generation |
| Ignore boundaries | ignore globs + FocusProfile | Scope indexing vs agent read |
| Product memory | `.project/`, specs, ADRs | Human + agent orientation |

**Rule:** local/registered bytes are **text authority**. Indexes hold coordinates
and embeddings, not the sole copy of truth.

**Context work:**

| Item | Priority | Notes |
|------|----------|-------|
| `Source` + `Artifact` registry | P0 | Paths, mime, project_id, checksum |
| Ignore / focus patterns | P0 | FocusProfile + corpus filters |
| Content-addressed blobs | P1 | Versioned artifact store |
| Git-aware metadata | P2 | Commit/branch on Source |

**For Generative UX:** catalog schemas, domain rules, brick catalogs, and prior
screen specs must be registerable sources/artifacts so refinements cite them.

---

## 2. Policy & workflow

| What | Examples | Used for |
|------|----------|----------|
| Project rules | `.mdc`-like rules, `AGENTS.md` | Always / glob / intelligent apply |
| Skills | `SKILL.md` trees | Progressive domain workflows |
| Plans / decisions | saved plan markdown, ADRs | Reviewable intent |
| Hooks / tool policy | JSON + scripts | Gate tool use |
| External tools | MCP-like adapters | Side systems |

**Rule:** policy injects **control**; it is not the semantic corpus (unless the
same file is also indexed as a normal source).

**Context work:**

| Item | Priority | Notes |
|------|----------|-------|
| `PolicySnapshot` | P0 | Frozen into ContextPack.policy_refs |
| Plan / spec as Decision or Artifact | P0 | Saved intent with checksum |
| Skill registry | P1 | Progressive load |
| Hook-equivalent typed policy | P2 | preToolUse / afterEdit style gates |
| External tool adapter | P2 | Boundary only; not core product |

**For Generative UX:** FocusProfile + ToolPolicy decide which bricks, data
queries, and emit tools are allowed on a refinement turn.

---

## 3. Session replay

Reference systems store chat bubbles, tool turns, and **exact prompt snapshots**
per model call in a local KV/DB.

**Context equivalent (do this, do not clone IDE schemas):**

| Item | Priority | Notes |
|------|----------|-------|
| `AgentRun` append-only trace | P0 | Status, parent/subagent |
| `ContextPack` snapshot per model call | P0 | Inspectable assembly |
| Message / ToolCall events | P0 | Structured, not opaque blobs |
| Checkpoint / rollback metadata | P2 | Often product-owned |
| Multi-root / project binding | P1 | project_id on every run |

**Storage:** PostgreSQL (SQLite acceptable only as a thin local experiment).  
**Not** the vector store.

**For Generative UX:** each refinement turn stores (a) ContextPack that fed the
draft, (b) structured UX-spec tool output as an Artifact, (c) verifier result.

---

## 4. Runtime spill

Long tool/terminal outputs should not bloat the session store.

| Item | Priority | Notes |
|------|----------|-------|
| Spill `Artifact` for tool output | P0 | path/size/mime/run_id |
| Slice read (offset/limit/grep) | P0 | Pack builder pulls excerpts |
| Transcript export | P2 | Debug only, not source of truth |
| Terminal stream adapter | P2 | Downstream shell capture |

**For Generative UX:** codegen logs, preview HTML, and large JSON specs live as
spill/artifacts; ContextPack keeps spans and summaries, not megabyte dumps.

---

## 5. Semantic index

| Item | Priority | Notes |
|------|----------|-------|
| `VectorStore` port | P0 | Hide backend |
| PostgreSQL + pgvector adapter | P0 | PoC first (ADR-0017) |
| QDrant / others | P1+ | After live stack proves the port |
| Chunk payload schema | P0 | span, source_id, checksum |
| Pluggable embedder | P0 | Model version in manifest |
| Reranker hook | P1 | After candidate merge |
| External docs corpus | P2 | Separate Source type |

**Not stored here:** chat history, rules-only text (unless also indexed files),
UX-spec JSON as the only copy of truth.

---

## 6. Keyword / exact index

| Item | Priority | Notes |
|------|----------|-------|
| Sparse / BM25 or trigram path | P0 | Exact facts, symbols, phrases |
| Retriever planner (multi-path) | P0 | Not single vector query |
| Merge + dedupe + rerank | P0 | CandidateSet → ContextPack |
| Entity / citation retriever | P1 | Beyond code grep |
| Morphology-aware path | P1 | Via language adapters |

**For Generative UX:** filters like “works with carrier N” need exact/metadata
retrieval against project data sources, not embedding luck.

---

## 7. Sync manifest

| Item | Priority | Notes |
|------|----------|-------|
| Manifest + per-artifact checksum | P0 | File-level diff |
| Merkle / rolling hash | P1 | Large corpora |
| Embed cache by chunk hash | P1 | Cost/latency |
| Copy-on-write index seed | P2 | Team templates |

---

## 8. Remote execution (optional)

| Item | Priority | Notes |
|------|----------|-------|
| Background `AgentRun` + cancel | P2 | Orchestrator, not storage core |
| Preview/build worker artifacts | P2 | Downstream builder owns sandbox |
| Attribution analytics | P3 | Product concern |

Core must not assume cloud IDE semantics.

---

## What goes where

| Data type | Session / trace | Vector index | Source / artifact store | Policy |
|-----------|:---------------:|:------------:|:-----------------------:|:------:|
| Chat / turn text | yes | no | no | no |
| ContextPack per turn | yes | no | optional export | refs |
| Tool output (full) | partial | no | **spill/artifact** | no |
| Source text | snapshot only | no | **yes** | no |
| Embeddings | no | **yes** | no | no |
| Rules / skills | no | no | yes if in corpus | **yes** |
| UX / screen / brick **spec JSON** | event + checksum | no | **yes (artifact)** | schema policy |
| Checkpoints / diffs | yes | no | no | no |

---

## Context module mapping

| Role | Context types | Direction |
|------|---------------|-----------|
| Source corpus | `Project`, `Source`, `Artifact` | `internal/corpus`, `internal/artifacts` |
| Policy | `PolicySnapshot`, `FocusProfile` | `internal/config` / focus |
| Session replay | `AgentRun`, events, `ContextPack` | `internal/agentruntime`, tracing |
| Spill | `Artifact` (tool/terminal) | `internal/artifacts` |
| Semantic index | `Chunk` vectors | `VectorStore` adapters |
| Keyword index | postings / sparse | `internal/retrieval`, indexing |
| Sync | `Manifest` | `internal/indexing` |
| Structured UX draft | Artifact + tool I/O schema | **consumer schema**; core stores/traces |

Central object: **`ContextPack`** — selected evidence + policy + budget,
replayable. Equivalent of a per-turn prompt snapshot, but explicit.

Structured UX specs are **not** ContextPacks. They are **outputs** (tool results /
artifacts) produced *after* a pack, then optionally re-ingested as sources for
the next refinement.

---

## Work volume (aligned with progress.md)

### P0 — hypothesis CLI proof (Chunks 02–12)

Roles **1, 3 (minimal), 5–6 (minimal), 7 (checksum)**:

1. Project + source registration, ignore/focus  
2. Parse → chunk → manifest  
3. Hybrid retrieve (exact/sparse + dense via port)  
4. ContextPack builder + verifier  
5. AgentRun + tool registry + fake model  
6. Spill artifact for long tool output  
7. PolicySnapshot from project config  
8. `context-dev` CLI loop  

### P1 — credible orchestrator

Subagents, rerank, skills, eval harness, Merkle, richer lexical path.

### P2 — downstream products (builders, mesh, rich linguistics)

Multi-tenant, background agents, external corpora, full morphology adapters,
preview sandboxes — **outside** or on top of core.

### Explicit non-goals for core

- IDE session-DB compatibility  
- Vendor cloud index lock-in  
- Built-in chat or Generative UI product shell  
- Brick/DOM codegen (belongs to UI toolkit / builder)  
- Privacy/training product toggles  

---

## Design rules

1. Never store vectors in the session/trace store.  
2. Never treat chat history as the retrieval corpus for facts.  
3. Policy files ≠ indexed corpus (pipelines differ even if path overlaps).  
4. Long outputs → artifacts, not prompt stuffing.  
5. One index namespace per project.  
6. Source/artifact store is text authority; index holds coordinates + embeddings.  
7. Every model call gets a replayable `ContextPack`.  
8. Every Generative UX draft gets a **schema-versioned artifact**
   (`artifact_type=structured` + `schema_id`, ADR-0022).

---

## References

- [Cursor Agent overview](https://cursor.com/docs/agent/overview)
- [Plan Mode](https://cursor.com/docs/agent/plan-mode)
- [Skills](https://cursor.com/docs/context/skills)
- [Rules](https://cursor.com/docs/context/rules)
- [Hooks](https://cursor.com/docs/agent/hooks)
- [Privacy & Data Governance](https://cursor.com/docs/enterprise/privacy-and-data-governance)
- [Dynamic context discovery](https://cursor.com/blog/dynamic-context-discovery)
- [Turbopuffer × Cursor](https://turbopuffer.com/customers/cursor)
- [cursaves — how Cursor stores chats](https://github.com/Callum-Ward/cursaves/blob/main/docs/how-cursor-stores-chats.md)
