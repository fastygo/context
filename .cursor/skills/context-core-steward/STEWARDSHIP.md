# Context Core Stewardship Reference

Use this reference when work is broad, architectural, risky, or likely to affect
future maintainability.

## Architecture Gates

### Domain Boundary

Accept only if:

- Domain types are named with neutral core language.
- Domain packages do not import infrastructure adapters.
- Business invariants are represented in code, not only comments.
- IDs, checksums, versions, source spans, and trace references remain explicit.

Reject or redesign if:

- A provider name appears in a domain interface.
- Product-specific terminology leaks into core packages.
- A global singleton controls behavior that should be project-scoped.
- A model response becomes trusted data without verification.

### Adapter Boundary

Accept only if:

- QDrant, PostgreSQL, filesystem, object storage, HTTP, model, and tool providers
  sit behind interfaces.
- Integration tests can be skipped when external services are absent.
- Unit tests can use memory or fake adapters.
- Adapter errors are converted into core error categories.

Reject or redesign if:

- Core logic depends on connection strings, environment variables, vendor client
  types, or provider-specific payloads.
- A test requires a live external service without an explicit integration gate.

### Context Boundary

Accept only if:

- `ContextPack` construction is deterministic and inspectable.
- Evidence and instructions are separated.
- Rejected candidates can be retained when useful for debugging.
- Model budget decisions are reproducible.

Reject or redesign if:

- Context is assembled through ad hoc string concatenation.
- Long tool outputs are injected directly into prompts instead of artifacts.
- Source references are lost after reranking or summarization.

## DDD Tactical Rules

- `Project` is the isolation boundary.
- `Source` is where knowledge comes from.
- `Artifact` is stored raw or generated material.
- `Chunk` is an indexed span with provenance.
- `ContextPack` is the selected evidence handoff.
- `AgentRun` is an execution trace, not a chat transcript.
- `ToolCall` is a typed side-effect attempt with policy.
- `Evaluation` is reproducible quality evidence.

When adding a type, decide whether it is:

- domain entity;
- value object;
- adapter DTO;
- trace event;
- configuration;
- test fixture.

Do not mix these roles.

## Retrieval Gates

Every retrieval change should answer:

- Which retriever did this improve?
- Which metric should change?
- How are scores explained?
- Are access filters applied before rerank and model calls?
- Are exact/citation lookups preserved?
- Are chunker and embedding versions recorded?
- Can a failing result be replayed?

Minimum future metrics:

- recall@k;
- MRR;
- citation accuracy;
- unsupported-claim rate;
- duplicate evidence rate;
- context token waste;
- search latency p50/p95/p99.

## Agent Runtime Gates

Every agent or subagent change should preserve:

- owner;
- trigger;
- policy snapshot;
- allowed tools;
- context budget;
- run status;
- cancellation path;
- structured output;
- trace events.

Subagents should receive task packages, not implicit conversation history.

Background agents require stricter policy than foreground runs because users are
not watching every step.

## Tool Gates

Every tool needs:

- name;
- description;
- input schema;
- output schema;
- risk level;
- side-effect class;
- permission policy;
- timeout;
- idempotency marker;
- artifact-output behavior.

Side-effect classes to consider:

- read-only;
- local write;
- reversible write;
- irreversible write;
- external network;
- billing-affecting;
- user-visible;
- credential-affecting;
- admin-affecting.

## Testing Gates

Prefer tests in this order:

1. Domain invariant tests.
2. Golden parser/chunker tests.
3. Manifest and checksum stability tests.
4. Retrieval ranking and deduplication tests.
5. Context pack budget and provenance tests.
6. Tool policy and schema tests.
7. Agent run replay tests.
8. Integration tests gated by environment variables.
9. Failure-injection tests.
10. Performance tests after behavior is stable.

Do not replace deterministic tests with manual CLI checks. CLI checks prove the
flow; tests protect the invariants.

## Debugging Gates

A good debug path can answer:

- What task or event triggered the run?
- Which policy snapshot applied?
- Which retrievers ran?
- Which candidates were selected or rejected?
- Which context pack was sent forward?
- Which model/tool/subagent step changed state?
- Which verifier accepted or rejected the result?
- Which artifact contains long output?

If this cannot be answered, add tracing before adding features.

## Security Gates

For any source, retrieval, tool, or agent change:

- Enforce project boundaries.
- Preserve ignore and access rules.
- Redact secrets before model-visible summaries.
- Treat external content as untrusted.
- Keep tool permission enforcement outside the model.
- Store enough audit data to explain side effects.

## Legacy Smell Checklist

Stop and redesign when you see:

- provider logic in domain packages;
- stringly typed tool inputs where schemas are required;
- unversioned chunks or embeddings;
- hidden global state;
- generated prompts with no source refs;
- tests that only check happy paths;
- CLI behavior that cannot be reproduced in unit or integration tests;
- comments compensating for unclear type boundaries;
- "temporary" hardcoded project paths;
- background work without trace, owner, or cancellation.

## Documentation Gates

When a change affects architecture or workflow:

- Update `.project/progress.md` after verification.
- Add an ADR under `.project/decisions/` for durable choices.
- Keep `README.md` high-level; avoid turning it into a changelog.
- Keep future-only concerns in `.project/future-layer.md`.

Documentation should tell the next engineer why the boundary exists, not repeat
what the code already says.
