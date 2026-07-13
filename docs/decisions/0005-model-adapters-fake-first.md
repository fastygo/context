# ADR-0005: Model Adapters — Fake First

Status: Accepted  
Date: 2026-06-17  
Related: [0006](0006-trace-event-append-only-replay.md)

## Context

PoC must prove indexing → retrieval → context pack → agent step → verification
without requiring live LLM or embedding API keys in CI. Real providers differ in
latency, cost, and failure modes and belong behind interfaces.

## Decision

1. Define adapter interfaces: `Embedder`, `LLM`, `Reranker` (minimal surface).
2. Ship deterministic **fake** implementations first:
   - hash-based pseudo-embeddings for stable ANN tests;
   - templated LLM responses driven by context pack checksum;
   - fixed rerank ordering from merged scores.
3. Real providers (OpenAI-compatible, local Ollama, etc.) plug in via config;
   never import provider SDKs from domain packages.
4. Every model call records `model_id`, adapter version, and input pack hash in
   the trace (ADR-0006).

## Consequences

### Positive

- CI and local dev run without network.
- Replay/debug remains deterministic until real models are enabled.

### Negative

- Fake embeddings do not validate semantic quality; eval harness needed when
  switching to real embedders.

### Follow-ups

- Verification step in PoC uses source span checks, not model judgment alone.
