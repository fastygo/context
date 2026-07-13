# ADR-0015: Multilingual Linguistic Contracts

Status: Accepted
Date: 2026-07-11
Related: [0008](0008-hybrid-index-architecture.md), [0011](0011-merkle-manifest-and-snapshot-namespace.md),
[0016](0016-lexicographic-context-contracts.md)
Plugin roadmap: [../plugins/language-adapters.md](../plugins/language-adapters.md)

## Context

The core must support multilingual retrieval, snippets, citations, and
morphology-aware query expansion without embedding language-specific
dictionaries or grammar rules. Domain models for Chunk 02 need stable,
language-neutral contracts and a clear `context-lang-*` adapter boundary.

## Decision

### 1. Core owns contracts; adapters own language behavior

```text
fastygo/context
  -> LanguageCode, ScriptCode, TokenOccurrence, LexemeID, Lemma, WordForm
  -> MorphFeatureSet, MorphAnalysis, QueryExpansion, AnalyzerVersion
  -> MorphAnalyzer, MorphGenerator, LexicalNormalizer, QueryExpander ports
  -> no language dictionaries, paradigms, or grammar tables

context-lang-*
  -> implements ports
  -> owns raw tags, dictionaries, expansion heuristics, eval fixtures
  -> never imported by core domain packages
```

Direction of dependency is one-way: adapters depend on core contracts; core
never depends on a language adapter repository.

### 2. Identity and value types (language-neutral)

| Type | Meaning |
|------|---------|
| `LanguageCode` | BCP 47 language tag string (e.g. `en`, `ru`, `hi`) |
| `ScriptCode` | ISO 15924 script tag (e.g. `Latn`, `Cyrl`, `Deva`) |
| `LexemeID` | Opaque adapter-stable lexeme reference within `(language, dictionary_version)` |
| `Lemma` | Canonical citation form string; not a sense and not a concept |
| `WordForm` | Surface or generated form paired with optional `MorphFeatureSet` |
| `TokenOccurrence` | Offset-preserving occurrence of surface text in a source |
| `MorphFeatureSet` | Portable feature bundle; preferred schemes `UD` or `UniMorph` |
| `MorphAnalysis` | One candidate analysis of a token; ambiguity is explicit |
| `QueryExpansion` | Explainable expansion of a query term; never silent rewrite |
| `AnalyzerVersion` | Composite of adapter/normalizer/tokenizer/analyzer/dictionary versions |

Typed string IDs are acceptable until two real adapters prove a richer ID shape.

### 3. TokenOccurrence invariants

Required fields: `id`, `project_id`, `source_id`, `chunk_id`, `language`,
`script`, `surface`, `normalized`, `span_start`, `span_end`,
`tokenizer_version`, `normalizer_version`.

Rules:

1. `surface` is original source text for the span; normalization never replaces it.
2. Spans follow [ADR-0018](0018-deterministic-identity-and-spans.md) (byte offsets
   canonical).
3. A token may have zero or more `MorphAnalysis` rows; none are selected by
   default unless an explicit selection reason is recorded.
4. Missing language capability must be explicit (`capability_missing`), not a
   silent approximation.

### 4. Morphology interfaces (ports only)

```text
MorphAnalyzer.Analyze(token) -> []MorphAnalysis
MorphGenerator.Generate(lexeme, features) -> []WordForm
LexicalNormalizer.Normalize(text, language, script) -> normalized + version
QueryExpander.Expand(term, language, policy) -> []QueryExpansion
```

Every result must carry `adapter_id`, relevant component versions, and
`feature_scheme`. Adapter-specific raw schemes (e.g. OpenCorpora) may appear only
in `raw_feature_scheme` / `raw_features`; they are not core enums.

### 5. QueryExpansion reasons

Allowed `expansion_type` values for phase 1:

`lemma`, `wordform`, `compound`, `accent`, `fuzzy`, `synonym`, `transliteration`

Each expansion records `original_term`, `expanded_term`, `confidence`, `reason`,
adapter versions. Expansions are candidates for retrieval; they are not source
truth and must be rejectable in traces without mutating artifacts.

### 6. Versioning and snapshots

When linguistic processing affects indexing or retrieval, `IndexSnapshot` and
trace events must record at least: `adapter_id`, `analyzer_version`,
`tokenizer_version`, `normalizer_version`, `dictionary_version`,
`feature_scheme`. Bumping any of these versions that change indexed tokens or
expansions requires a new snapshot per ADR-0011.

### 7. PoC fixtures

Chunks 02–08 may ship no-op or fixture analyzers inside the core test tree that
implement the ports with deterministic fake data. Production language behavior
belongs in `context-lang-*` after the CLI proof.

## Consequences

### Positive

- Chunk 02 can define linguistic structs without importing language repos.
- Ambiguity, expansions, and versions remain explainable in traces.
- Universal Dependencies / UniMorph stay portable across adapters.

### Negative

- Real morphology quality is deferred; first proof uses fakes/simple hooks.
- Feature-scheme mapping work moves to adapters and contract tests.

### Follow-ups

- Shared adapter testkit (`context-lang-testkit`) after ports stabilize.
- Capability descriptor schema when the first real adapter lands.
