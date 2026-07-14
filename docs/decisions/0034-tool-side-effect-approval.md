# ADR-0034: Tool Side-Effect Approval Baseline

Status: Accepted  
Date: 2026-07-14  
Related: [0020](0020-contextpack-budget-and-evidence.md),
stabilization gap **C6**

## Context

Write and external tools are unsafe if a missing policy rule silently allows
execution. `SideEffectClass` already exists on `ToolSchema` but was unused in
`policy/eval`.

## Decision

1. Explicit policy rules still win for any tool name (including `*`).
2. When no rule matches, `write` and `external` side effects decide **`ask`**
   — even if `Engine.Default` is `allow`.
3. Orchestrator maps `ask` to tool status `needs_approval` and **does not**
   execute the tool.
4. Read/none tools keep prior default (deny unless rule/Default allows).

## Consequences

### Positive

- Approval required for non-read classes without an allow rule.
- Explicit allow remains available for trusted deployments.

### Negative

- Consumers must handle `needs_approval` in UI/CLI; no in-core human prompt.

### Follow-ups

- Approval grant records in traces when a product layer confirms `ask`.
