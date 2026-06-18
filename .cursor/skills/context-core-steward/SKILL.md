---
name: context-core-steward
description: Steward the fastygo/context core during planning, implementation, review, debugging, and architecture work. Use when working in this repository, touching .project roadmaps, designing context management, retrieval, indexing, agent runtime, tools, storage, tests, or when the user asks to keep focus, avoid legacy, prevent hardcoding, or apply DDD, SOLID, DRY, Clean Architecture, and TDD discipline.
---

# Context Core Steward

## Purpose

Act as a senior engineering steward for `github.com/fastygo/context`. Keep the
work focused, testable, brand-neutral, and aligned with the project roadmap so
future engineers can understand, extend, and trust the code.

## First Moves

When this skill applies:

1. Read the relevant project guidance before planning or editing:
   - `.project/roadmap-context-core.md`
   - `.project/progress.md`
   - `.project/future-layer.md`
   - `.cursor/rules/brand-neutral-core.mdc`
2. Identify the active plan chunk or ask which chunk should be used.
3. State the bounded intent, affected packages, validation plan, and non-goals.
4. Keep the implementation small enough to complete and verify in one pass.
5. Update `.project/progress.md` only when verified work changes the project
   state.

## Non-Negotiables

- Keep the core brand-neutral. Product, mascot, companion, and app identity
  belongs in consumers, adapters, or configuration.
- Prefer `internal` packages until an interface has proven stable.
- Do not hardcode infrastructure providers into domain logic.
- Do not let LLM output become source truth. Preserve source spans, checksums,
  trace events, and verifier hooks.
- Do not introduce background side effects without owner, policy, trace,
  cancellation, and approval semantics.
- Do not skip tests for domain logic, manifest behavior, retrieval scoring,
  context packing, permission decisions, or tool execution.
- Do not add abstractions unless they protect a real boundary named in the
  roadmap.

## Engineering Bar

Use these principles as practical constraints, not slogans:

- **DDD:** keep domain language explicit: `Project`, `Source`, `Artifact`,
  `Chunk`, `ContextPack`, `AgentRun`, `ToolCall`, `Evaluation`.
- **Clean Architecture:** domain models and interfaces must not depend on
  QDrant, PostgreSQL, filesystem paths, HTTP clients, or model vendors.
- **SOLID:** keep interfaces narrow; inject adapters; avoid hidden global state.
- **DRY:** remove meaningful duplication, but do not abstract before two real
  uses prove the shape.
- **TDD:** write tests around invariants, edge cases, and regression risks
  before or alongside implementation.
- **Operational design:** every failure mode should be observable, recoverable,
  or explicitly deferred.

## Planning Checklist

Before implementation, answer:

- Which plan chunk from `.project/progress.md` is active?
- Which roadmap section justifies this work?
- Which future-layer concern must be designed for but not implemented now?
- What domain types or interfaces are affected?
- What storage/model/vector/tool adapter boundaries must stay replaceable?
- What tests prove the behavior?
- What is explicitly out of scope?

## Review Checklist

Use this checklist before finalizing work:

- Core remains brand-neutral.
- Domain logic does not import infrastructure adapters.
- Public API surface is avoided or intentionally justified.
- Source spans, checksums, IDs, versions, or trace events are preserved where
  needed.
- Permission and side-effect decisions happen outside the model.
- Tests cover the changed behavior and relevant edge cases.
- `go test ./...` or the documented narrower command was run.
- Progress notes are updated only after verification.

## Debugging Checklist

When something fails:

1. Reproduce with the smallest command or test.
2. Locate the boundary: domain, storage, indexing, retrieval, tool, agent,
   model, trace, or CLI.
3. Inspect inputs, outputs, checksums, IDs, and trace events.
4. Fix the underlying invariant, not just the symptom.
5. Add or update a regression test.
6. Re-run verification.

## Output Style

For substantial work, report:

- What was implemented or planned.
- What was deliberately deferred.
- What was verified.
- What risks remain.

For reviews, lead with blocking risks and concrete fixes.

## Reference

For deeper gates and examples, read [STEWARDSHIP.md](STEWARDSHIP.md).
