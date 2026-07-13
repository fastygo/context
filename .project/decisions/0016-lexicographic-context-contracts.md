# ADR-0016: Lexicographic Context Contracts

Status: Accepted
Date: 2026-07-11
Related: [0015](0015-multilingual-linguistic-contracts.md),
[0020](0020-contextpack-budget-and-evidence.md)
Plugin roadmap: [../plugins/lexicon-resources.md](../plugins/lexicon-resources.md)

## Context

Lexeme and morphology answer "which form." Context also needs "which meaning,
where, when, in which register, and according to which evidence." Dictionary
content, TEI/SKOS importers, and historical/regional lexicons must stay outside
the neutral core while Chunk 02 still needs stable domain types.

## Decision

### 1. Evidence layer, not NLP preprocessing

```text
TokenOccurrence
  -> WordForm / Lemma / Lexeme
  -> Sense
  -> Concept
  -> Attestation
  -> SourceSpan
  -> ContextPackEvidence
```

Language adapters may hint senses; lexicon resource adapters own authority,
licensing, and import/export. Core never embeds dictionary or thesaurus data.

### 2. Core contract types

| Type | Meaning |
|------|---------|
| `Sense` | One meaning of a lexeme; never collapsed into lemma |
| `Concept` | Language-independent or domain concept with labels/relations |
| `Attestation` | Witnessed use in a source with quote and provenance |
| `Variant` | Non-canonical form meaningful for retrieval or history |
| `MultiwordExpression` | Lexical unit spanning multiple tokens |
| `Register` | Usage layer (formal, slang, legal, …) as metadata reference |
| `DialectRegion` | Geographic/community usage boundary as metadata reference |
| `TimePeriod` | Date range, era, or orthography period as metadata reference |
| `LexiconSource` | Dictionary, corpus, glossary, authority list, community lexicon |

`Register`, `DialectRegion`, and `TimePeriod` are metadata-rich references, not
hardcoded closed enums when vocabularies are corpus- or product-specific.

### 3. Required fields (minimum)

**Sense:** `id`, `project_id`, `lexeme_id`, `language`, `definition` (optional
when only labels exist), `concept_id` (optional), `register`, `region`,
`time_period`, `lexicon_source_id`, `source_authority`, `confidence`,
`license_ref`, `metadata`.

**Concept:** `id`, `project_id`, `preferred_label`, `labels`, `concept_scheme`,
`broader`/`narrower`/`related`, exact/close/broad/narrow matches,
`lexicon_source_id`, `license_ref`, `metadata`.

**Attestation:** `id`, `project_id`, `source_id`, `chunk_id` (when indexed),
`span_start`, `span_end`, `quote`, `language`, optional links to
`lexeme_id`/`sense_id`/`concept_id`/`variant_id`, `attested_at`, `region`,
`register`, `source_authority`, `confidence`, `import_version`, `metadata`.

**Variant:** `id`, `project_id`, `canonical_ref`, `variant`, `variant_type`
(`orthographic`, `historical`, `regional`, `slang`, `spelling`, `script`,
`transliteration`), `language`, `script`, `region`, `time_period`, `source_id`,
`confidence`.

**MultiwordExpression:** `id`, `project_id`, `surface`, `normalized`, `language`,
`token_ids` or span, optional `lexeme_id`/`sense_id`, `expression_type`,
`analyzer_version`, `confidence`.

**LexiconSource:** `id`, `kind`, `title`, `authority`, `license`, `version`,
`language_scope`, `uri` (optional), `metadata`.

### 4. Authority and licensing

1. Every lexicographic claim used in retrieval or a `ContextPack` must cite a
   `LexiconSource` or be marked `inference` / `adapter_hint`.
2. Generated wordforms are **not** attestations unless witnessed in a source.
3. SKOS labels are not lemmas; concept mappings stay separate from morphology.
4. TEI/SKOS/CSV/RDF importers live in resource adapters; core stores only mapped
   contracts plus provenance and license metadata.
5. License metadata must be preserved when resource text or definitions are
   stored or emitted in packs.

### 5. Retrieval filters (ports)

Retrieval may filter or boost by: `sense_id`, `concept_id`, `attestation_id`,
`register`, `dialect_region`, `time_period`, `lexicon_source`, `source_authority`.

Filters are explainable constraints. They do not replace original source text or
override project policy.

### 6. Resource adapter boundary

```text
LexiconResourceAdapter
  Import / Lookup / MapToContracts / LicenseMetadata

Core
  stores Sense, Concept, Attestation, …
  never imports TEI, SKOS, or dictionary SDKs in domain packages
```

PoC may use fixture lexicon rows in tests. Production TEI/SKOS/dictionary
adapters are deferred until after the CLI proof.

## Consequences

### Positive

- Chunk 02 can model senses and attestations without resource importers.
- Historical, regional, and slang evidence share one provenance shape.
- ContextPack evidence classes can distinguish source text vs sense claims.

### Negative

- Full dictionary interoperability is deferred; fixtures only in PoC.
- Authority conflicts between sources need later policy (future layer).

### Follow-ups

- Contract tests for TEI and SKOS mappers after PoC.
- Conflict-resolution policy when two lexicon sources disagree.
