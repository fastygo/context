# ADR-0038: S3 Thin In-Repo Adapters

Status: Accepted  
Date: 2026-07-14  
Related: [0015](0015-multilingual-linguistic-contracts.md),
[0016](0016-lexicographic-context-contracts.md),
[0037](0037-public-langtestkit.md),
stabilization gaps **A1**, **A3**, **A4**, **A5**, **A7**

## Context

S3 requires replaceable linguistic, lexicon, document, and event paths — not
wishful interfaces. Full production engines stay external; thin adapters prove
ports.

## Decision

| Gap | Thin adapter | Notes |
| --- | --- | --- |
| A1 | `context-lang-en` via `pkg/langtestkit/refen` + `internal/linguistic/en` | Light lemma + fixture expansion; passes harness |
| A3 | `internal/lexicon/jsonres` | Curated JSON → `ResourceAdapter`; passes lexicon harness |
| A4 | `parse.HTML` (`html-text-v1`) | Tag strip; `ExtractionConfidence=0.9`; Original preserved |
| A5 | `parse.PDF` (`pdf-strings-v1`) | Literal string scrape; **LowConfidence**; Original preserved |
| A7 | `source.NDJSONFiles` | EventAdapter; idempotent batch checksum; temporal filter test |

`Document` gains `ExtractionConfidence` and `LowConfidence` for lossy extractors.
Local file discovery includes `.html`/`.pdf`; NDJSON discovery is a separate
`EventAdapter`.

## Consequences

### Positive

- S3 exit tests are executable offline without vendor SDKs.
- Provenance (Original bytes + parser version + confidence) stays auditable.

### Negative

- PDF extraction is intentionally lossy; production may replace with a richer
  adapter behind the same `Parser` port.

### Follow-ups

- External `context-lang-ru` and TEI lexicon mappers when measured need appears.
