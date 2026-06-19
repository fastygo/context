# Context Core Future Layer Roadmap

Status: deferred architecture backlog  
Scope: production-grade capabilities that should be designed for, but not
implemented before the hypothesis-validation CLI proof.

This document extends `roadmap-context-core.md` and `progress.md`. The current
implementation path should still focus on the proof loop:

```text
ingest -> retrieve -> context pack -> fake model/tool run -> verifier -> trace
```

The layers below are intentionally deferred. They are the gates that turn a
working context engine into a reliable private search system, distributed agent
runtime, and long-lived production platform.

## Deferral Principle

Do not implement these layers early unless a proof-of-concept chunk is blocked
by their absence.

For each future layer:

- Define the contract early.
- Keep the interface narrow.
- Add tests before adding infrastructure.
- Implement the smallest adapter first.
- Promote to distributed or optimized form only after measurement.

## Layer 01: Threat Model And Prompt-Injection Defense

### Why This Matters

The engine will ingest untrusted text from files, web pages, logs, tool outputs,
and user-authored documents. Any source can try to influence agent behavior.
Without an explicit threat model, retrieval can become a prompt-injection
delivery mechanism.

### Future Capabilities

- Threat model document covering source poisoning, prompt injection, cross-project
  leakage, tool exfiltration, web page attacks, and malicious documents.
- Source trust levels: `trusted`, `project`, `external`, `untrusted`,
  `quarantined`.
- Instruction/data separation in context packs.
- Prompt-injection classifier hooks for external content.
- Policy that forbids retrieved source text from overriding system, developer,
  project, or tool policies.
- Quarantine path for suspicious sources.
- Tests with adversarial documents.

### Acceptance Gate

- A malicious source can be retrieved as evidence without gaining tool
  permissions or overriding runtime policy.
- The context pack clearly labels untrusted content.
- Prompt-injection fixtures are part of the regression suite.

## Layer 02: Fine-Grained Access Control

### Why This Matters

Project-level isolation is enough for the first PoC. Production systems need
permissions below the project boundary: source, artifact, chunk, tool output,
agent run, report, and background result.

### Future Capabilities

- ACL model for `Project`, `Source`, `Artifact`, `Chunk`, `ContextPack`,
  `AgentRun`, `ToolCall`, and `Evaluation`.
- Permission filters applied before candidate merge, rerank, model calls, and
  tool execution.
- Access decisions stored in traces.
- Denied-candidate counters for debugging.
- Tenant/project/user/service-account identities.
- Contract tests proving no cross-project or cross-user leakage.

### Acceptance Gate

- A user cannot retrieve, rerank, summarize, or indirectly expose a chunk they
  cannot read.
- Permission checks happen outside the LLM.
- Retrieval traces show which filters were applied.

## Layer 03: Index Lifecycle Management

### Why This Matters

Indexes decay. Sources move, chunks change, embeddings get replaced, parsers are
upgraded, and old vectors become stale. Without lifecycle rules, search results
will become unreliable and hard to debug.

### Future Capabilities

- Index state machine: `new`, `indexing`, `ready`, `degraded`, `stale`,
  `rebuilding`, `failed`, `archived`.
- Tombstones for deleted sources and chunks.
- Stale chunk cleanup.
- Orphan artifact cleanup.
- Parser/chunker/enricher/embedding/sparse/graph version coexistence.
- Zero-downtime reindex with old/new index side by side.
- Backfill jobs.
- Rebuild and repair CLI commands.
- Index health report.

### Acceptance Gate

- Reindexing a project does not make search unavailable.
- Deleted sources cannot appear in new context packs.
- The system can explain which index version produced a result.

## Layer 03A: Snapshot Replication And Index Reuse Hardening

### Why This Matters

ADR-0012 defines `IndexSnapshot` as the sync unit so local and cloud retrieval can
share the same ranking inputs. The first PoC should prove local snapshot commit
and search. Production-grade replication, reuse, and cross-device parity require
additional safety gates.

### Future Capabilities

- Snapshot export/import CLI and service APIs.
- Sparse index bundle hashes in manifest records.
- QDrant point export or snapshot restore validation by `snapshot_id`.
- Local pull command that verifies `source_merkle_root`, `chunk_set_hash`, bundle
  hashes, and engine versions before flipping `active_snapshot_id`.
- Incremental segment sync for sparse bundles.
- Snapshot retention and garbage collection.
- Simhash over chunk multisets for near-duplicate snapshot discovery.
- Copy-on-write seed for project templates or near-identical workspaces.
- Merkle content proofs before any cross-user or cross-team index reuse.

### Acceptance Gate

- Local and cloud search can prove they are using the same snapshot and engine
  versions.
- A corrupted or partial snapshot cannot become active.
- Cross-user reuse is impossible until content proofs and ACL filters are tested.

## Layer 04: Query Language And Advanced Search Semantics

### Why This Matters

Natural language retrieval is not enough for a private search system. Users need
precise search when investigating facts, legal text, scientific sources, logs,
or configuration.

### Future Capabilities

- Query parser for:
  - `AND`, `OR`, `NOT`
  - exact phrases
  - proximity search
  - field filters
  - source filters
  - date/time filters
  - language filters
  - citation filters
  - fuzzy flags
  - wildcard flags
- Query AST with validation.
- Conversion from query AST to retriever calls.
- Explanation of how a query was interpreted.
- Safe fallback when a query cannot be parsed.

### Acceptance Gate

- Advanced queries are deterministic.
- Query parsing is test-covered with golden inputs.
- Query interpretation is visible in retrieval traces.

## Layer 04A: Focus Control And Memory Tiers

### Why This Matters

When a project has too many documents, messages, tasks, logs, and tool outputs,
accuracy drops unless the runtime can focus retrieval before calling a model.
Focus control is a neutral core concern because every downstream product needs
it, regardless of domain.

### Future Capabilities

- `FocusProfile` persistence and inspection.
- Memory tiers:
  - raw artifacts;
  - indexed chunks;
  - active task context;
  - accepted decisions;
  - summaries with source refs;
  - negative assumptions;
  - rejected candidates.
- Focus constraints for:
  - source types;
  - source trust levels;
  - freshness windows;
  - exactness level;
  - citation strictness;
  - token/context budget;
  - allowed tools and subagents;
  - explicitly irrelevant areas.
- Focus inspector for debugging why a context pack used some sources and ignored
  others.
- Focus regression tests for large corpora.

### Acceptance Gate

- Large corpora can be searched through a bounded focus profile.
- The runtime can explain which focus constraints shaped retrieval.
- A model call never receives broad corpus context without an explicit focus
  decision.

## Layer 05: Snippet, Highlighting, And Evidence Presentation

### Why This Matters

Search quality is not only ranking. Users and agents need exact evidence
presentation: where the match happened, why it matched, and what surrounding
context is safe to include.

### Future Capabilities

- Offset-preserving snippets.
- Term highlighting.
- Lemma/stem match highlighting.
- Citation span display.
- Before/after context windows.
- Page, section, heading, paragraph, and line labels.
- Quote-safe extraction for scientific/legal text.
- Snippet checksums tied to source versions.

### Acceptance Gate

- Search results can show exact source snippets without re-tokenizing
  differently from the index.
- Highlighting is stable across repeated runs.
- Snippets preserve provenance.

## Layer 06: Relevance Feedback And Learning Loop

### Why This Matters

High-quality context management improves through traces: which evidence was
selected, ignored, opened, rejected, verified, or corrected. This feedback is
the foundation for better reranking and future custom retrieval models.

### Future Capabilities

- Feedback events:
  - result opened
  - result ignored
  - context pack accepted
  - context pack rejected
  - verifier failure
  - tool result used
  - user correction
  - generated spec accepted
- Offline eval dataset generation from traces.
- Retrieval-quality dashboards.
- Reranker training export.
- Embedding training export.
- Privacy-aware trace retention.

### Acceptance Gate

- Every important user/agent feedback signal is captured as an event.
- Eval datasets can be generated without exposing raw private content by
  default.
- Feedback can improve retrieval without changing source truth.

## Layer 07: Claim, Contradiction, And Citation Graph

### Why This Matters

Scientific, legal, linguistic, and operational corpora contain claims that can
conflict, expire, or depend on source authority. A plain vector index cannot
represent this.

### Future Capabilities

- Claim extraction model.
- Claim-to-source graph.
- Citation graph.
- Contradiction edges.
- Supersession edges.
- Version/edition awareness.
- Trust scoring by source type and verification status.
- Claim verifier for generated summaries and specs.
- "Evidence says" vs "model infers" separation.

### Acceptance Gate

- The engine can represent two conflicting sources without flattening them into
  one answer.
- Generated factual claims can be traced to supporting or conflicting evidence.
- Superseded sources are visible but down-ranked or flagged.

## Layer 08: Distributed Job Control

### Why This Matters

Background indexing, crawling, agent runs, verification, and monitoring need a
real distributed execution model before production scale.

### Future Capabilities

- Job queue adapter.
- Leases and heartbeats.
- Idempotency keys.
- Retry policy.
- Dead-letter queue.
- Cancellation.
- Timeout budgets.
- Resource quotas.
- Worker capabilities.
- Shard ownership for large projects.
- Backpressure when model, vector, metadata, or artifact stores are degraded.

### Acceptance Gate

- A worker crash does not corrupt an index or agent run.
- Duplicate jobs do not produce duplicate side effects.
- Long-running jobs can be cancelled and inspected.

## Layer 09: Tool Sandbox And Side-Effect Model

### Why This Matters

Distributed agents become dangerous when tools can affect external systems.
Permission labels are not enough; the runtime needs side-effect semantics.

### Future Capabilities

- Tool side-effect classes:
  - read-only
  - local write
  - reversible write
  - irreversible write
  - external network
  - billing-affecting
  - user-visible
  - credential-affecting
  - admin-affecting
- Dry-run mode.
- Compensation/rollback hooks.
- Approval checkpoints.
- Network allowlists.
- Filesystem sandbox policy.
- Environment-variable and secret access policy.
- Tool causal graph.

### Acceptance Gate

- The runtime can explain what side effects a run may perform before it starts.
- Risky tools cannot run without policy approval.
- Tool outputs cannot silently escalate permissions.

## Layer 10: Operational SLOs And Capacity Planning

### Why This Matters

Production reliability needs explicit budgets. Without SLOs, every optimization
and incident response becomes subjective.

### Future Capabilities

- SLOs for:
  - ingest latency
  - indexing freshness
  - search p50/p95/p99
  - context pack build latency
  - agent startup latency
  - background job completion
  - QDrant availability
  - metadata store availability
  - artifact store availability
- RPO/RTO targets.
- Capacity model by documents, chunks, embeddings, projects, tenants, and
  background jobs.
- Load tests.
- Soak tests.
- Cost model.

### Acceptance Gate

- The team can say whether the system is healthy.
- Performance regressions are measurable.
- Scaling decisions are driven by observed bottlenecks.

## Layer 11: Crawler Governance And Web Capture Safety

### Why This Matters

Web search and crawling introduce legal, operational, and security risks. Broad
crawling should not be part of the first proof.

### Future Capabilities

- Explicit URL capture before broad crawling.
- robots.txt policy.
- Rate limits.
- Per-host budgets.
- User-agent policy.
- Content-type allowlist.
- Maximum page size.
- Redirect limits.
- Archive snapshots with timestamps.
- Malware and script stripping.
- Source trust classification.

### Acceptance Gate

- External content cannot overwhelm the system.
- Web captures are reproducible and source-stamped.
- Crawling policy is auditable.

## Layer 12: Multi-Modal And Binary Document Pipeline

### Why This Matters

Private corpora often include PDFs, DOCX, scans, spreadsheets, archives, email,
and images. These formats should be adapters, not core assumptions.

### Future Capabilities

- PDF text extraction.
- DOCX extraction.
- Spreadsheet extraction.
- Email and mailbox extraction.
- Archive traversal with depth limits.
- OCR adapter.
- Table extraction.
- Figure/image metadata extraction.
- Binary fingerprinting.
- Extraction confidence scores.

### Acceptance Gate

- Extracted text has provenance back to file, page, section, table, or image.
- Low-confidence extraction is flagged.
- Binary parsers cannot escape sandbox/resource limits.

## Layer 13: Privacy, Encryption, And Retention

### Why This Matters

Context systems often store sensitive source material, traces, prompts, model
outputs, and tool results. Retention must be explicit.

### Future Capabilities

- Retention policies per artifact type.
- Encryption-at-rest adapter support.
- Key management boundary.
- PII and secret detection.
- Redacted traces for support.
- Raw trace access policy.
- Export/delete project data.
- Audit trail for data access.

### Acceptance Gate

- Sensitive data has retention and access policy.
- Support/debug views do not expose raw secrets by default.
- Project deletion removes or tombstones all expected data.

## Layer 14: Multi-Tenant And Team Governance

### Why This Matters

Single-user local PoC does not need governance. Teams and hosted deployments do.

### Future Capabilities

- Tenant model.
- Team/project roles.
- Service accounts.
- Team-level policies.
- Required rules.
- Shared adapters.
- Shared model/provider budgets.
- Cross-project search policy.
- Admin audit views.

### Acceptance Gate

- Team policies can restrict tools, models, sources, and background runs.
- Project-level settings cannot weaken enforced tenant policy.
- Cross-project retrieval is explicit and audited.

## Layer 15: API Stability, SDK, And Compatibility

### Why This Matters

The core should eventually support downstream products, custom companions,
tools, and adapters without forks.

### Future Capabilities

- Stable `pkg/contextkit` API.
- Dockerized `context-core` service for BFF/API consumers.
- HTTP/gRPC API contract for ingest, search, context pack, agent run, trace, and
  snapshot inspection.
- Thin client SDKs generated from or aligned with the service contract.
- Adapter contract tests.
- Tool SDK.
- Companion configuration format.
- Lab-facing JSON/DTO compatibility tests using proof artifacts from
  `.project/proof/`.
- Scenario plugin contracts for source adapters, graph projections, tool packs,
  rule packs, and methodology packs.
- DSL schemas for FocusProfile, RetrievalPlan, ContextPackTemplate, ToolPolicy,
  SourceAdapterConfig, and AgentRunPolicy.
- Migration policy.
- Semantic versioning policy.
- Deprecation policy.
- Example neutral applications.

### Acceptance Gate

- Third-party adapters can be tested without private internals.
- Public APIs are versioned and documented.
- Breaking changes are intentional and announced.
- A downstream BFF or lab shell can consume the service/SDK without importing
  internal packages.

## Layer 15A: Lab-Driven UX/DX/DSL Workbench

### Why This Matters

The CLI proves the engine. A browser lab proves whether humans and product BFFs
can understand and operate it. This is a downstream concern, but it should feed
neutral contract improvements back into the core.

### Future Capabilities

- UX fixture screens for project corpus, source list, search results, snippets,
  FocusProfile, ContextPack, AgentRun, and trace timeline.
- DX dashboard for QDrant, `context-sparse`, PostgreSQL, active snapshot,
  source/chunk counts, and integration test status.
- BFF adapter that calls the Context service or consumes `context-dev` JSON
  during local development.
- DSL workbench for editing/visualizing FocusProfile, RetrievalPlan,
  ContextPackTemplate, ToolPolicy, SourceAdapterConfig, and AgentRunPolicy.
- Contract tests proving Lab fixture JSON and Context CLI/API JSON stay
  compatible.

### Acceptance Gate

- Lab can demonstrate ingest -> search -> context pack -> fake agent -> trace
  without importing Context internals.
- Any Lab-discovered requirement is translated into a neutral Context contract
  or explicitly kept in Lab.
- UX/DX/DSL screens do not change core semantics without an ADR.

## Layer 16: Production Review Gates

These gates should be required before a hosted or paid system depends on the
engine.

### Security Gate

- Threat model reviewed.
- Prompt-injection tests exist.
- Cross-project leakage tests pass.
- Secret redaction tests pass.
- Tool permission tests pass.

### Reliability Gate

- Index rebuild tested.
- Store outage tested.
- Worker crash tested.
- Agent cancellation tested.
- Backup/restore tested.

### Retrieval Quality Gate

- Golden retrieval suite passes.
- Citation accuracy threshold defined.
- Unsupported-claim rate tracked.
- Ranking regressions visible.

### Operations Gate

- SLOs defined.
- Metrics emitted.
- Logs structured.
- Traces replayable.
- Runbooks written.

## Future Layer Sequencing

Recommended order after the current proof loop:

1. Threat model and prompt-injection fixtures.
2. Fine-grained retrieval ACL.
3. Index lifecycle management.
4. Snapshot replication and index reuse hardening.
5. Focus control and memory tiers.
6. Query language and snippet/highlighting engine.
7. Relevance feedback events.
8. Distributed job control.
9. Tool sandbox and side-effect model.
10. Operational SLOs and capacity tests.
11. Claim/contradiction graph for scientific/legal corpora.
12. Crawler governance.
13. Privacy/encryption/retention.
14. Multi-tenant/team governance.
15. Binary/multi-modal adapters.
16. Stable SDK and ecosystem contracts.
17. Lab-driven UX/DX/DSL workbench hardening.

## What Must Not Move Into The First PoC

- Broad web crawling.
- Cross-user index reuse.
- Simhash copy-on-write seeding and Merkle content proofs.
- Incremental sparse segment sync beyond simple snapshot export/import.
- Custom embedding-model training.
- Multi-tenant billing.
- Full sandboxed code execution.
- Marketplace/plugin ecosystem.
- Dockerized `context-core` service and public SDKs before CLI contracts settle.
- Lab-specific UX, widgets, BFF routes, or DSL screens inside Context core.
- Messaging catalogs, calendar/Gantt products, CRM workflows, dashboards, or
  methodology runtimes inside the neutral core.
- Distributed worker orchestration.
- Complex binary document extraction.
- Production-grade query language.
- Claim contradiction graph.

The first PoC should remain small enough to debug line by line. These future
layers should guide interfaces and tests, not inflate the first implementation.
