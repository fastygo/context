# Language Adapter Plugin Roadmap

Status: deferred plugin roadmap  
Scope: external language adapter repositories for multilingual lexeme,
morphology, lexical retrieval, query expansion, snippets, and citation support.

## Purpose

`fastygo/context` must stay multilingual by contract and neutral by design. It
should define stable language contracts, provenance, indexing, retrieval,
`ContextPack`, and trace semantics. It must not become a Russian, German,
Spanish, French, Hindi, Indic, or English morphology engine.

Language-specific complexity belongs in external adapters that implement core
contracts and pass shared compatibility tests.

```text
fastygo/context
  -> language-neutral contracts
  -> source spans, snapshots, retrieval, ContextPack, traces
  -> no language-specific dictionaries or grammar rules

context-lang-*
  -> normalization
  -> tokenization
  -> lexeme and wordform analysis
  -> morphology generation
  -> query expansion
  -> language-specific eval fixtures
```

## Repository Naming

Planned repositories:

- `github.com/fastygo/context-lang-testkit`
- `github.com/fastygo/context-lang-en`
- `github.com/fastygo/context-lang-ru`
- `github.com/fastygo/context-lang-de`
- `github.com/fastygo/context-lang-es`
- `github.com/fastygo/context-lang-fr`
- `github.com/fastygo/context-lang-hi`
- `github.com/fastygo/context-lang-indic`

The exact list may grow, but the contract direction must not change:

```text
context-lang-* -> context contracts
context -> no context-lang-* imports
```

## Core Contract Surface

Adapters should implement only stable contracts exposed by the core after the
PoC proves them:

- `LanguageCode`
- `ScriptCode`
- `TokenOccurrence`
- `TokenSpan`
- `LexemeID`
- `Lemma`
- `WordForm`
- `MorphFeatureSet`
- `MorphAnalysis`
- `MorphAnalyzer`
- `MorphGenerator`
- `LexicalNormalizer`
- `QueryExpander`
- adapter capability descriptor

Core contracts should support Universal Dependencies and UniMorph as portable
feature schemes. Adapter-specific raw schemes such as OpenCorpora may be carried
as raw metadata, but they must not become core enums.

## Adapter Capabilities

Each adapter declares which features it supports:

- Unicode normalization.
- Script detection.
- Tokenization with offset preservation.
- Lemmatization.
- Lexeme lookup.
- Wordform generation.
- Morphology analysis.
- Ambiguity scoring.
- Query expansion.
- Compound splitting.
- Transliteration.
- Stop-word policy.
- Accent/diacritic policy.
- Snippet/highlight metadata.

Unsupported capabilities must be explicit. A missing capability is better than a
silent approximation.

## Versioning Requirements

Every adapter result must carry enough version data to make retrieval
reproducible:

- adapter id;
- adapter version;
- normalizer version;
- tokenizer version;
- analyzer version;
- generator version;
- dictionary version;
- feature scheme;
- raw feature scheme when applicable.

`IndexSnapshot`, retrieval traces, proof artifacts, and `ContextPack` evidence
must preserve these versions when language processing affects a result.

## Shared Contract Tests

`context-lang-testkit` should provide tests that every official adapter must
pass:

- normalization is deterministic;
- token offsets preserve source spans;
- snippets can be reconstructed without re-tokenizing differently;
- analyzer output is stable for fixture inputs;
- ambiguity is explicit, not silently collapsed;
- selected analyses include reasons;
- generated wordforms round-trip where the language allows it;
- query expansion is explainable and can be disabled;
- expansions do not cross project, source, trust, or permission boundaries;
- feature sets serialize and deserialize without losing scheme information;
- adapter version changes can force a new `IndexSnapshot`.

## Language Notes

### English

Primary role: baseline adapter and simplest contract proof.

Focus:

- light lemmatization;
- part-of-speech tagging when available;
- plural and verb-form handling;
- phrase/exact search stability;
- low-risk query expansion.

Non-goal: do not treat English as the universal language model for all adapters.

### Russian

Primary role: rich morphology stress test.

Focus:

- OpenCorpora-compatible raw tags where useful;
- lemma and lexeme lookup;
- full inflectional paradigms;
- case, number, gender, animacy, aspect, tense, person, mood, voice;
- `ё/е` normalization policy;
- ambiguity resolution with multiple candidate analyses;
- query expansion by controlled wordform generation;
- legal/scientific citation safety.

Non-goal: do not move Russian grammemes into core enums.

### German

Primary role: compounds and agreement stress test.

Focus:

- noun capitalization;
- case, gender, number;
- adjective agreement;
- compound splitting;
- separable verbs;
- umlaut/orthographic variants;
- phrase and proximity search across compound parts.

Non-goal: do not make compound splitting mandatory for every language.

### Spanish

Primary role: verb morphology and accent policy.

Focus:

- rich verb conjugation;
- tense, mood, person, number;
- gender and number agreement;
- accent/diacritic handling;
- clitic handling;
- lemma-aware search for inflected verbs.

Non-goal: do not fold accents unless the adapter records the expansion reason.

### French

Primary role: elision, contraction, and silent morphology.

Focus:

- elision;
- contractions;
- accents;
- agreement;
- verb conjugation;
- apostrophe tokenization;
- snippet stability for surface text.

Non-goal: do not normalize away apostrophes or accents without traceable
provenance.

### Hindi

Primary role: Devanagari and Indo-Aryan morphology proof.

Focus:

- Devanagari Unicode normalization;
- tokenization and segmentation;
- postpositions;
- oblique case;
- gender and number;
- compound verbs;
- transliteration boundaries when enabled.

Non-goal: do not force Latin-script assumptions onto Hindi.

### Indic Language Family

Primary role: broader script and segmentation adapter family.

Focus:

- script-specific Unicode normalization;
- transliteration boundaries;
- language identification;
- segmentation;
- dictionary resource policy;
- per-language capability declaration;
- shared fixtures across multiple Indic scripts where appropriate.

Non-goal: do not collapse all Indic languages into one morphology model.

## Release Lifecycle

Adapter lifecycle states:

- `experimental`: contract still moving; fixtures may be small.
- `supported`: contract tests pass and versioning policy is documented.
- `deprecated`: adapter is still usable but should not be chosen for new
  projects.
- `retired`: adapter is no longer maintained; old snapshots remain replayable if
  artifacts and versions are available.

Official support requires:

- documented capability descriptor;
- contract test pass;
- language-specific golden fixtures;
- versioning policy;
- compatibility note for `IndexSnapshot` and `ContextPack`;
- security and license review of dictionaries/resources.

## Integration With Context Core

The core should discover adapters through configuration or registry wiring owned
by the consuming application. It should not import language repositories.

Language processing may influence:

- chunk metadata;
- lexical/sparse indexes;
- query expansion;
- candidate ranking;
- snippets and highlighting;
- citation extraction;
- `ContextPack` evidence selection;
- verifier checks for factual claims.

Every influence must be traceable.

## Non-Goals

- No language-specific dictionaries inside `fastygo/context`.
- No Russian, German, Spanish, French, Hindi, or Indic grammar enums in core.
- No adapter-specific raw tag scheme as the only stored representation.
- No silent query expansion.
- No morphology result without source offsets.
- No official adapter without contract tests.
- No product or brand-specific language workflows in these adapters.

## Acceptance Gate

This plugin roadmap becomes actionable only after:

- core domain contracts for token spans, morphology features, analyses, and
  query expansion are defined;
- the PoC loop works with no-op/simple language adapters;
- `context-lang-testkit` requirements are stable enough to prevent adapter drift;
- an ADR confirms the public language adapter boundary;
- at least one downstream consumer can inspect language metadata in proof JSON
  without importing adapter internals.
