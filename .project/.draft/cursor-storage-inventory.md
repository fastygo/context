# Cursor Storage Inventory → Context Work Scope

Status: draft reference  
Purpose: map where Cursor stores what, and why — to size `github.com/fastygo/context` work.  
Privacy/training modes are out of scope here; this is a **storage topology** note only.

Sources: Cursor docs (agent, plan mode, skills, hooks, enterprise privacy/indexing),
Turbopuffer case study, community reverse-engineering (cursaves, dev.to agent analysis).
Treat `state.vscdb` key families as best-effort until Cursor publishes a schema.

---

## Executive summary

Cursor is not one database. It is **eight storage roles** that must not be collapsed:

| # | Role | Cursor location | Primary purpose |
|---|------|-----------------|-----------------|
| 1 | **Source corpus** | Project files on disk | Ground truth for code/text |
| 2 | **Policy & workflow** | `.cursor/` + `AGENTS.md` in repo | Rules, skills, plans, hooks, MCP |
| 3 | **Session replay** | `state.vscdb` (SQLite KV) | Chat, tool turns, checkpoints, prompt snapshots |
| 4 | **Runtime spill** | `~/.cursor/projects/.../` | Long outputs, transcripts, terminals |
| 5 | **Semantic index** | Remote vector DB (Turbopuffer) | Codebase + @Docs similarity search |
| 6 | **Keyword/regex index** | Remote (+ local ripgrep at query) | Exact symbol/pattern lookup |
| 7 | **Sync manifest** | Client Merkle tree ↔ server | Incremental re-index, index reuse |
| 8 | **Cloud execution state** | Cloud Agent VM (optional) | Clone repo, run tests, PR artifacts |

`@Context` must implement **roles 1–4 and 7–8 (later)** in a project-scoped, auditable way,
plus **5–6** via adapters (QDrant + sparse index). Role 3 maps to **`ContextPack` + `AgentRun` traces**, not to the vector layer.

---

## Layer diagram

```text
                         USER / REPO
                              │
         ┌────────────────────┼────────────────────┐
         │                    │                    │
         v                    v                    v
   [1] Source files     [2] .cursor/ policy   [3] state.vscdb
   git working tree     rules, skills,        session KV:
   .project/, docs       plans, hooks, mcp     bubbles, checkpoints,
                         AGENTS.md             messageRequestContext
         │                    │                    │
         │                    │                    v
         │                    │              [4] ~/.cursor/projects/
         │                    │                  transcripts, agent-tools,
         │                    │                  terminals, mcps/
         │                    │                    │
         └──────────────┬─────┴────────────────────┘
                        │
            indexing & retrieval (agent tools)
                        │
         ┌──────────────┼──────────────┐
         v              v              v
   [5] Vector index  [6] Grep/regex  [7] Merkle manifest
   Turbopuffer       index + rg      hash sync
   per workspace     (server trigram  client↔server
   namespace         + local scan)   diff + simhash copy
         │              │              │
         └──────────────┴──────────────┘
                        │
                        v
              LLM prompt (assembled per turn)
                        │
                        v
              [8] Cloud Agent VM (optional)
                  repo clone, build, artifacts
```

---

## 1. Source corpus (local disk, project-owned)

| What | Where | Format | Used for |
|------|-------|--------|----------|
| Application source | Repo tree | `.go`, `.templ`, `.md`, … | Edits, preview, generation |
| Ignore boundaries | `.gitignore`, `.cursorignore`, `.cursorindexingignore` | glob rules | Scope indexing vs agent read |
| Product memory (BuildY-style) | `.project/`, specs, architecture docs | Markdown | Human + agent orientation (not auto-indexed unless in corpus) |

**Cursor behavior:** local files are **source of truth** for text. Vector search returns coordinates; client reads bytes from disk.

**Context work:**

| Item | Priority | Notes |
|------|----------|-------|
| `Source` + `Artifact` registry | P0 | Register paths, mime, project_id |
| Ignore / focus patterns | P0 | Like `.cursorignore` + `FocusProfile` |
| Content-addressed blobs | P1 | Checksum per artifact version |
| Git-aware versioning | P2 | Commit/branch on `Source` metadata |

---

## 2. Policy & workflow (repo + user home, mostly text)

### 2a. Project-scoped (git-friendly)

| What | Where | Format | Used for |
|------|-------|--------|----------|
| Project rules | `.cursor/rules/*.mdc` | MD + YAML frontmatter | Always / globs / intelligent apply |
| Agent instructions | `AGENTS.md` (root + nested) | Plain MD | Simpler rule alternative |
| Skills | `.cursor/skills/**/SKILL.md` | MD + optional scripts/ | Progressive domain workflows |
| Subagents | `.cursor/agents/*.md` | MD + YAML | Delegation configs |
| Commands | `.cursor/commands/*.md` | MD | Explicit slash workflows |
| Hooks | `.cursor/hooks.json`, `.cursor/hooks/` | JSON + scripts | Agent loop policy (format, gate, audit) |
| MCP | `.cursor/mcp.json` | JSON | External tools |
| Plans (after save) | `.cursor/plans/*.md` | MD + YAML frontmatter | Reviewable implementation plans |
| Cloud env | `.cursor/environment.json` | JSON | Cloud Agent image/snapshot |
| Index local state | `.cursor/codebase.json` | JSON | Client index bookkeeping (not vectors) |

### 2b. User-scoped (not in repo)

| What | Where | Format | Used for |
|------|-------|--------|----------|
| User rules | Cursor Settings → User Rules | text | Global behavior |
| User skills | `~/.cursor/skills/` | SKILL.md trees | Cross-project skills |
| Built-in skills | `~/.cursor/skills-cursor/` | Cursor-managed | Product skills |
| User subagents | `~/.cursor/agents/` | MD | Global agents |
| Global MCP | `~/.cursor/mcp.json` | JSON | Global tools |
| Global hooks | `~/.cursor/hooks.json` | JSON | Global agent policy |
| Plans (default) | `~/.cursor/plans/` | `.md` | Ephemeral until "Save to workspace" |
| CLI permissions | `~/.cursor/cli-config.json` | JSON | Shell allow/deny |
| IDE recency | `~/.cursor/ide_state.json` | JSON | Recently viewed files |

**Cursor behavior:** rules/skills inject **policy** into the agent loop; they are **not** the semantic codebase index (unless the same files also live in the repo as normal sources).

**Context work:**

| Item | Priority | Notes |
|------|----------|-------|
| `PolicySnapshot` from rules files | P0 | Load `.mdc`-like rules + AGENTS.md |
| Skill registry (name, description, paths) | P1 | Progressive load like Cursor skills |
| Plan / spec as `Decision` artifact | P0 | Map Plan mode → `.project/specs/` |
| Hook-equivalent policy hooks in Go | P2 | `preToolUse`, `afterFileEdit` as typed policy |
| MCP as external tool adapter | P2 | Not core; adapter boundary |

---

## 3. Session replay (local SQLite KV)

| What | Where | Format | Used for |
|------|-------|--------|----------|
| All conversation bodies | `%APPDATA%/Cursor/User/globalStorage/state.vscdb` (Win) | SQLite: `ItemTable`, `cursorDiskKV` | Full chat history all projects |
| Workspace sidebar index | `User/workspaceStorage/{hash}/state.vscdb` | SQLite KV | Composer list, selected tabs, workspace binding |
| Workspace path map | `workspaceStorage/{hash}/workspace.json` | JSON | `folder` URI ↔ hash |
| Composer metadata | `composerData:{uuid}` | JSON blob | Headers, usage, codeBlockData |
| Messages | `bubbleId:{composerId}:{bubbleId}` | JSON blob | User/assistant text, tools, reasoning |
| Agent blobs | `agentKv:*` | binary/JSON | Large agent state |
| Prompt snapshot | `messageRequestContext:{composerId}:{messageId}` | JSON blob | **Exact context sent to model on that turn** |
| Checkpoints | `checkpointId:{composerId}:{id}` | JSON blob | Restore diffs per agent turn |
| Dedup content | `composer.content.{hash}` | blob | Content-addressed shared payloads |
| Code suggestion state | `codeBlockDiff:*` | JSON | accepted / rejected |

**Cursor behavior:** UI reads from `state.vscdb`. Summarization affects **model window only**, not disk retention (community observation).

**Context work:**

| Item | Priority | Notes |
|------|----------|-------|
| `AgentRun` trace (append-only) | P0 | Foreground run id, status, parent/subagent |
| `ContextPack` snapshot per model call | P0 | Equivalent of `messageRequestContext` |
| Message / tool call records | P0 | bubble-level events |
| Checkpoint / rollback metadata | P2 | Optional; product layer may own this |
| Global vs workspace session index | P1 | Multi-root / project_id binding |

**Storage engine for Context:** PostgreSQL (or SQLite for PoC), **not** QDrant.

---

## 4. Runtime spill files (local, per workspace)

Path pattern: `~/.cursor/projects/{workspace-slug}/`

| What | Subpath | Format | Used for |
|------|---------|--------|----------|
| Chat export | `agent-transcripts/*.txt` | text | Write-only log; subagents in `*/subagents/*.jsonl` |
| Tool output cache | `agent-tools/*.txt` | text | Large tool results without bloating session DB |
| Terminal capture | `terminals/` | text | Grepable shell history |
| MCP tool sync | `mcps/{server}/` | files | Lazy MCP descriptor load |
| Subagent state | `~/.cursor/subagents/` | files | Background subagent progress |

**Cursor behavior:** **dynamic context discovery** — agent reads slices via grep/read, not full inject.

**Context work:**

| Item | Priority | Notes |
|------|----------|-------|
| Spill `Artifact` type for tool output | P0 | Store path, size, mime, run_id |
| Slice read API (offset/limit/grep) | P0 | Pack builder pulls excerpts only |
| Transcript export (optional) | P2 | JSONL for debug, not SoT |
| Terminal stream adapter | P2 | Browser/BFF shell capture |

---

## 5. Semantic index (remote vector store)

| What | Where | Format | Used for |
|------|-------|--------|----------|
| Code chunk embeddings | Turbopuffer namespace per workspace | float vectors + payload | `@codebase`, agent semantic search |
| Chunk metadata | same | obfuscated path, start/end line, chunk hash | Map hit → local read |
| @Docs embeddings | server-side (shared per URL) | vectors | Library / custom URL docs |
| Embedding model | Cursor infra | code-tuned model | Query + chunk same space |

**Indexing pipeline (codebase):**

```text
open workspace → Merkle scan → changed files → tree-sitter chunks
  → embed on server → store vector + metadata → discard plaintext
query → embed question → ANN → metadata → client reads local file lines
```

**Not stored in vector layer:** chat history, rules-only text (unless part of indexed files), plan files in home dir.

**Context work:**

| Item | Priority | Notes |
|------|----------|-------|
| QDrant adapter (collection per `project_id`) | P0 | PoC target in README |
| Chunk model + payload schema | P0 | span, source_id, checksum, enrichments |
| Pluggable embedder | P0 | Model version in manifest |
| @Docs-style external corpus adapter | P2 | Crawl + index as separate `Source` type |
| Reranker hook | P1 | After ANN merge |

---

## 6. Keyword / regex index (hybrid retrieval)

| What | Where | Format | Used for |
|------|-------|--------|----------|
| Agent grep | local ripgrep + server trigram index | inverted index | Symbols, exact patterns |
| File search | local fuzzy path index | paths | `@file`, open by name |
| List dir | local FS | — | Structure exploration |

**Cursor behavior:** semantic + grep **together**; agent chooses chain (concept → semantic, symbol → grep).

**Context work:**

| Item | Priority | Notes |
|------|----------|-------|
| Sparse/BM25 or trigram index | P0 | Required for LingvY-grade exact lookup |
| Entity / phrase / citation retriever | P1 | Beyond code-oriented grep |
| Retriever planner (multi-path) | P0 | Not single vector query |
| Merge + dedupe + rerank pipeline | P0 | `CandidateSet` → `ContextPack` |

---

## 7. Sync manifest (incremental index)

| What | Where | Format | Used for |
|------|-------|--------|----------|
| Client Merkle tree | client memory + sync | SHA-256 tree | Detect changed files only |
| Server Merkle tree | Cursor infra | hash tree | Index state on server |
| Simhash | server | fingerprint | Copy namespace from similar checkout |
| Content proofs | server (temporary) | Merkle proofs | Safe index reuse across clones |
| Embedding cache | AWS (per chunk hash) | cached vectors | Skip re-embed unchanged chunks |

**Context work:**

| Item | Priority | Notes |
|------|----------|-------|
| `Manifest` with per-artifact checksum | P0 | File-level diff |
| Merkle or rolling hash tree | P1 | Large monorepos |
| Copy-on-write index seed | P2 | Team template projects |
| Embed cache by chunk hash | P1 | Cost/latency |

---

## 8. Cloud execution & attribution (optional / side channels)

| What | Where | Format | Used for |
|------|-------|--------|----------|
| Cloud Agent repo | isolated VM | git clone | Build, test, PR |
| Agent artifacts | cloud | screenshots, video, logs | Handoff / review |
| AI code attribution | `~/.cursor/ai-tracking/ai-code-tracking.db` | SQLite | composer vs tab vs human lines |
| Team rules / memories | Cursor cloud | server store | Cross-session product features |
| Account analytics | Cursor dashboard | metrics | Usage, not full prompt content |

**Context work:**

| Item | Priority | Notes |
|------|----------|-------|
| Background `AgentRun` + cancellation | P2 | Orchestrator, not storage core |
| Attribution / blame metadata | P3 | Product analytics, not retrieval |
| Cloud memories | defer | Use project-scoped memory with provenance instead |

---

## What goes where (quick reference)

| Data type | Session DB | Vector index | Source disk | Policy files |
|-----------|:----------:|:------------:|:-----------:|:------------:|
| Chat messages | yes | no | no | no |
| Prompt snapshot per turn | yes | no | no | no |
| Tool output (full) | partial | no | spill files | no |
| Source code text | snapshot only | no (embed only) | **yes** | no |
| Code embeddings | no | **yes** | no | no |
| Rules / skills text | no | no | yes (if in repo) | **yes** |
| Plans | no* | no | yes after save | yes |
| @Docs pages | no | **yes** | no | no |
| Checkpoints / diffs | yes | no | no | no |

\*Default plans live in `~/.cursor/plans/` until saved to repo.

---

## Context module mapping

| Cursor role | Context type(s) | Package direction |
|-------------|-----------------|-------------------|
| Source corpus | `Project`, `Source`, `Artifact` | `internal/corpus/`, `internal/artifacts/` |
| Policy & workflow | `PolicySnapshot`, `Decision` | `internal/config/` |
| Session replay | `AgentRun`, events, `ContextPack` | `internal/agentruntime/`, `internal/tracing/` |
| Runtime spill | `Artifact` (tool/terminal) | `internal/artifacts/` |
| Semantic index | `Chunk` vectors | `internal/storage/` + QDrant |
| Keyword index | `Chunk` postings | `internal/retrieval/`, `internal/indexing/` |
| Sync manifest | `Manifest` | `internal/indexing/` |
| Graph (imports, citations) | edges | `internal/graph/` (P1+) |
| Cloud run | `AgentRun` background | `internal/agentruntime/` (P2) |

Central object: **`ContextPack`** = selected evidence + policy + budget, versioned and replayable.
Equivalent of Cursor's per-turn assembly (`messageRequestContext`), but **explicit and inspectable**.

---

## Work volume estimate for `@Context`

### P0 — first proof (README hypothesis path)

Must implement analogues of roles **1, 3 (minimal), 5, 6 (minimal), 7 (file checksum)**:

1. Project + source registration, ignore patterns
2. Parse → chunk → manifest (checksum)
3. QDrant ingest + hybrid search stub (dense + keyword)
4. Retrieval plan → candidates → **ContextPack**
5. **AgentRun** trace + **ContextPack** persistence (Postgres or SQLite)
6. Spill artifact for long tool output
7. Load rules/specs as **PolicySnapshot** (`.project/` + config dir)
8. Fake model/tool step + deterministic verifier on source spans

**Rough surface:** ~8–12 Go packages touched, 2 storage backends, 1 CLI (`context-dev`).

### P1 — credible agent orchestrator

- Subagent isolation + summary handoff
- Reranking, recency, graph edges (imports / citations)
- Skill registry + progressive policy load
- Eval harness (retrieval + pack quality)
- Merkle manifest, embed cache

### P2 — product-grade (BuildY / browser / LingvY)

- Multi-tenant `project_id`, index copy-on-write
- Background agents, sandbox policy
- External corpus adapters (docs, web, messaging)
- Morphology / linguistic enrichers (LingvY)
- Full hook/policy engine

### Explicit non-goals for core (match README)

- IDE SQLite `state.vscdb` compatibility
- Turbopuffer / Cursor cloud integration
- Built-in chat UI
- Privacy/training mode toggles (product/tenant concern)

---

## Design rules (from Cursor, for Context)

1. **Never store vectors in the session store.**
2. **Never treat chat history as the retrieval corpus** (LingvY: facts need spans + exact index).
3. **Policy files ≠ indexed corpus** — same repo path can be both, but different pipelines.
4. **Long outputs → artifacts**, not prompt stuffing.
5. **One namespace per project** for vectors + metadata.
6. **Source disk (or artifact store) is text authority**; index holds coordinates + embeddings only.
7. **Every model call gets a replayable `ContextPack`**, not only opaque KV blobs.

---

## References

- [Cursor Agent overview](https://cursor.com/docs/agent/overview)
- [Plan Mode](https://cursor.com/docs/agent/plan-mode)
- [Skills](https://cursor.com/docs/context/skills)
- [Rules](https://cursor.com/docs/context/rules)
- [Hooks](https://cursor.com/docs/agent/hooks)
- [Privacy & Data Governance (indexing flows)](https://cursor.com/docs/enterprise/privacy-and-data-governance)
- [Dynamic context discovery](https://cursor.com/blog/dynamic-context-discovery)
- [Turbopuffer × Cursor](https://turbopuffer.com/customers/cursor)
- [cursaves — how Cursor stores chats](https://github.com/Callum-Ward/cursaves/blob/main/docs/how-cursor-stores-chats.md)
