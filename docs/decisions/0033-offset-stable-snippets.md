# ADR-0033: Offset-Stable Snippet / Highlight Contract

Status: Accepted  
Date: 2026-07-14  
Related: [0018](0018-deterministic-identity-and-spans.md),
[0019](0019-phase1-retrieval-scoring.md),
stabilization gap **C4**

## Context

Consumers need citations, not only chunk blobs. Snippet offsets must stay
stable when unchanged chunk bytes are re-indexed.

## Decision

1. Add `retrieval.Snippet` on `Candidate` with absolute half-open byte spans
   into **chunk text** (UTF-8), plus `chunk_checksum` and highlight spans.
2. Builders live in `internal/retrieval/snippet`: phrase find matches exact
   retrieval (case-sensitive substring); window defaults 40/40 bytes.
3. CLI/HTTP search attaches snippets after merge via `snippet.Attach`.
4. Snippets are presentation derived from chunk text — not a separate indexed
   artifact.

## Consequences

### Positive

- Re-index of identical bytes yields identical `chunk_span` / highlights.
- Provenance stays tied to chunk checksum.

### Negative

- Phrase match is first-hit only; multi-hit highlighting is deferred.

### Follow-ups

- Lemma/stem highlight adapters when language plugins need them.
