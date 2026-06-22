# Lexicon Resource Plugin Roadmap

Status: deferred plugin roadmap  
Scope: dictionary, thesaurus, controlled vocabulary, historical lexicon,
regional vocabulary, slang, community lexicon, and corpus-attestation resources.

## Purpose

Language adapters explain how forms behave. Lexicon resource adapters explain
what forms mean, where they are attested, which authority supports them, and how
they map to concepts or controlled vocabularies.

`fastygo/context` should define neutral contracts for senses, concepts,
attestations, variants, registers, regions, time periods, and source authority.
It should not embed dictionary content, thesaurus content, historical lexicons,
slang lists, or community vocabularies in the core.

```text
TokenOccurrence
  -> WordForm
  -> Lemma
  -> Lexeme
  -> Sense
  -> Concept
  -> Attestation
  -> SourceSpan
  -> ContextPackEvidence
```

## Resource Types

Future adapters may support:

- TEI dictionaries.
- TEI Lex-0 compatible dictionaries.
- SKOS/SKOS-XL concept schemes.
- ISO 25964-style thesauri.
- Authority lists.
- Glossaries and terminology bases.
- Historical dictionaries.
- Regional and dialect dictionaries.
- Slang and community lexicons.
- Corpus-derived attestation stores.
- Manually curated project vocabularies.

## Core Contract Surface

Adapters should map resources to neutral contracts:

- `Sense`
- `Concept`
- `Attestation`
- `Variant`
- `MultiwordExpression`
- `Register`
- `DialectRegion`
- `TimePeriod`
- `LexiconSource`
- source authority descriptor
- resource license descriptor

These contracts should remain usable even when the source format is TEI, SKOS,
CSV, JSON, XML, SQL, RDF, or a custom corpus export.

## TEI Path

TEI dictionary import/export should preserve:

- entry identifiers;
- headwords;
- homographs;
- senses and subsenses;
- definitions;
- examples;
- citations;
- etymology where available;
- orthographic variants;
- source references;
- edition/version metadata;
- language and region tags.

TEI import must not flatten all senses into one lemma. Dictionary examples and
citations should become `Attestation` records with source spans or source
references wherever possible.

## SKOS And Thesaurus Path

SKOS/ISO 25964 compatibility should preserve:

- concept schemes;
- preferred labels;
- alternate labels;
- hidden labels;
- broader/narrower/related relations;
- exact/close/broad/narrow mappings;
- scope notes;
- source notes;
- language tags;
- scheme/version metadata.

SKOS labels are not the same thing as lemmas. Concept mappings must remain
separate from language adapter morphology.

## Attestation Model

Every attestation should preserve:

- original quote or excerpt;
- source id;
- source span or explicit source-location reference;
- page/section/paragraph when available;
- attestation date or source date;
- region/dialect/community metadata when available;
- register metadata when available;
- linked lexeme, sense, concept, or variant;
- source authority;
- confidence;
- extraction/import version.

Generated wordforms are not attestations unless they are witnessed in a source.

## Historical And Regional Lexicons

Support should be designed for:

- diachronic search across time periods;
- historical spelling and orthographic reforms;
- archaic and obsolete forms;
- regional labels;
- dialect and ethnogroup metadata;
- slang and community-specific usage;
- competing authorities and conflicting definitions;
- edition-aware dictionary entries.

Retrieval should be able to ask:

- "What did this word mean in this period?"
- "Where is this form attested?"
- "Is this regional or general usage?"
- "Which authority supports this sense?"
- "Which concept does this term map to in this domain?"

## Versioning And Governance

Every resource import must record:

- adapter id;
- adapter version;
- source resource id;
- source resource version;
- import timestamp;
- license;
- source authority;
- transformation rules;
- checksum or content hash;
- schema/profile version.

Resource licensing must be checked before data is used for indexing,
redistribution, eval fixtures, or model-training exports.

## Contract Tests

Future `context-lang-testkit` or a dedicated lexicon resource testkit should
prove:

- original source text is preserved;
- senses are not collapsed into lemmas;
- concepts are not collapsed into labels;
- attestations have source spans or explicit source-location references;
- historical/region/register filters are deterministic;
- resource versions change snapshot identity when relevant;
- conflicting senses can coexist;
- concept mappings remain explainable;
- license metadata is present before export or training use.

## Integration With Context Core

Lexicon resources may influence:

- chunk metadata;
- source authority scoring;
- exact and lexical retrieval;
- sense/concept filters;
- query expansion;
- snippets and highlights;
- citation extraction;
- `ContextPack` evidence selection;
- verifier requirements;
- eval dataset generation.

Every influence must be traceable and replayable.

## Non-Goals

- No dictionary content inside `fastygo/context`.
- No thesaurus or controlled vocabulary as core data.
- No generated wordform treated as witnessed usage.
- No source-less attestation.
- No license-less resource import.
- No flattening senses into lemmas.
- No flattening concepts into labels.
- No historical or regional normalization without provenance.

## Acceptance Gate

This plugin roadmap becomes actionable only after:

- core contracts for `Sense`, `Concept`, `Attestation`, `Variant`,
  `MultiwordExpression`, `Register`, `DialectRegion`, `TimePeriod`, and
  `LexiconSource` exist;
- the PoC can export proof JSON with original text and derived lexical metadata;
- language adapter boundaries are stable enough to avoid coupling dictionary
  importers to morphology implementations;
- an ADR confirms TEI/SKOS/resource adapter boundaries;
- licensing and resource-governance review rules are documented.
