# GRACE Methodology Plugin Note

Status: deferred research plugin  
Scope: future methodology adapter for contract-first AI engineering practices
inspired by Vladimir Ivanov's GRACE/PCAM work.

Reference channel: https://t.me/turboproject

## Purpose

This note preserves GRACE as a future plugin candidate for the context core. It
must not become a hard dependency, core brand, or mandatory implementation
style. The core should remain neutral and expose generic primitives that can
support GRACE-like workflows through adapters, contracts, rules, skills, and
configuration.

## Working Understanding

Based on preliminary public references, GRACE can be treated as a methodology
for AI-assisted engineering that combines:

- contract-first development;
- semantic markup for LLM navigation;
- project knowledge graphs;
- verification plans;
- operational packets;
- governed autonomy for agents.

In this repository, these ideas map naturally to existing neutral concepts:

| GRACE-like idea | Context core concept |
| --- | --- |
| Module contract | `Decision`, future `Contract` artifact |
| Semantic markup | `Chunk` metadata, source spans, graph edges |
| Knowledge graph | `graph` package and source/artifact relations |
| Verification plan | `Evaluation`, verifier requirements |
| Operational packet | `ContextPack` template |
| Governed autonomy | `PolicySnapshot`, `AgentRun`, tool permissions |

## Research Tasks For Later

Before implementing anything, perform a dedicated English-source review and
methodology analysis:

1. Read primary GRACE/PCAM materials from Vladimir Ivanov and the TurboProject
   channel.
2. Separate confirmed methodology from community/plugin interpretations.
3. Identify which parts are implementation-independent and which are tied to a
   specific AI coding tool.
4. Compare GRACE concepts against `roadmap-context-core.md`,
   `progress.md`, and `future-layer.md`.
5. Draft an ADR for whether GRACE should be supported as:
   - documentation convention;
   - skill/rule pack;
   - contract artifact schema;
   - graph projection;
   - full plugin adapter.
6. Keep core names generic even if the plugin uses GRACE terminology.

## Possible Plugin Shape

A future plugin may provide:

- contract templates;
- verification-plan templates;
- context-pack templates;
- graph export/import conventions;
- semantic chunk annotation rules;
- agent policy presets for governed autonomy;
- review gates aligned with contract-driven development.

## Non-Goals

- Do not add GRACE-specific terminology to core package names.
- Do not require XML, semantic block comments, or any specific marker format in
  the base engine.
- Do not make GRACE the only supported methodology.
- Do not implement this before the proof-of-concept context loop works through
  CLI, indexing, retrieval, context packs, fake model/tool execution, verifier,
  and trace replay.

## Acceptance Gate

This plugin becomes actionable only after:

- the PoC loop is working;
- core `ContextPack`, `Decision`, `Evaluation`, `AgentRun`, and `ToolCall`
  primitives exist;
- a detailed GRACE/PCAM research note is written;
- an ADR confirms the minimal integration shape.
