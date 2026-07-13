# Lexical Terms For Context Core

Status: draft reference  
Purpose: keep linguistic vocabulary precise for contracts, retrieval, and
Generative UX — without trademark risk in code or APIs.

Related: ADR-0015, ADR-0016, [glossary-en.md](../who-and-why/glossary-en.md),
[generative-ux-from-context.md](./generative-ux-from-context.md).

---

## Why this draft exists

Context Runtime treats language as **contracts + adapters**, not as embedded
dictionaries. Generative UX needs the same discipline: UI is assembled from
discrete units (bricks / screen nodes), the way text is handled via lexemes and
wordforms — **modular composition**, not opaque model prose.

This file is the engineering cheat-sheet. It replaces a long chat transcript.

---

## Core terms (plain)

| Term | Plain meaning | In the engine |
|------|---------------|---------------|
| **Lexis** | Vocabulary of a language or domain (“ocean of words”) | Domain corpus / controlled vocabulary as *sources* |
| **Lexicon** | Working stock actually in use | What retrieval and products operate on |
| **Lexeme** | Word as a meaning unit with all its forms | Discrete search/contract unit |
| **Lemma** | Dictionary / citation form | Normalization key |
| **Wordform** | Concrete surface form (*played*, *plays*) | Exact match in spans |
| **Morphology** | Rules that produce wordforms | Language **adapters**, not core hardcoding |
| **Sense** | One meaning of a lexeme | Lexicographic contract (ADR-0016) |
| **Attestation** | Witnessed quote with span | Evidence that can justify claims |

**Simple chain:**

```text
lexis (domain ocean)
  → lexicon (what this project uses)
    → lexeme (unit)
      → morphology → wordforms
      → senses / attestations → evidence
```

---

## Greek line (etymology only)

| Form | Role | Use in this repo |
|------|------|------------------|
| **legō (λέγω)** | Verb: gather, pick out, speak | Prose/etymology only |
| **lexis (λέξις)** | Noun: chosen word / manner of speech | Conceptual naming |
| **logos (λόγος)** | Sense / reason inside the speech | Do not confuse with lexis |

**Latin look-alike:** Latin *lex* = **law** (legal, legitimate). Unrelated to Greek *lex-*.

Do **not** turn *legō* into code identifiers that read as `lego`.

---

## Morphology vs language type (why adapters matter)

| Language type | Behavior | Engine implication |
|---------------|----------|--------------------|
| Synthetic (e.g. Russian) | Rich endings inside the word | Morphology adapter critical for recall |
| Analytic (e.g. English) | Order + prepositions | Lighter inflection; still need lemma/wordform |
| Mixed (e.g. German) | Articles carry much case work | Adapter encodes language-specific features |

Core stores **feature bundles and analyzer versions**. Dictionaries and taggers
live in versioned language adapters (ADR-0015).

---

## Mapping to Context types

| Linguistic idea | Context / ADR hook |
|-----------------|--------------------|
| Token / lemma / lexeme / wordform | ADR-0015 multilingual contracts |
| Sense / concept / attestation / register | ADR-0016 lexicographic contracts |
| Exact phrase / citation | Hybrid retrieval + source spans |
| “Pick the right unit” | FocusProfile + ContextPack selection |
| Modular UI unit (brick) | Downstream toolkit — analogous *composition*, not a core type named “lexeme-ui” |

---

## Generative UX angle (keep short)

| Text problem | Linguistic analogy | System move |
|---------------|--------------------|-------------|
| Model invents a filter | Empty logos, pretty lexis | Require source/tool evidence for data-bound UI |
| One-off HTML dump | No reusable lexeme | Emit schema/brick tree, not only prose |
| Drift across React/Templ/Twig | Inconsistent wordforms | One brick contract → many runtimes (UI toolkit) |
| Refinement loses prior meaning | Lost lemma | Prior UX-spec artifact re-enters corpus |

---

## Naming and trademark safety

**LEGO®** is a trademark of the LEGO Group. Casual “lego architecture” in APIs
and module names is a risk and is **forbidden** in this repository’s identifiers.

| Allowed | Forbidden |
|---------|-----------|
| *legō* / *lexis* in etymology prose | `lego`, `Lego`, `LEGO` in packages, commands, schemas |
| modular / block / assembly / lexeme | Product names like “DataLego”, “LexiconLego” |
| Comparative mention of LEGO® with ® + disclaimer | Treating LEGO® as a generic engineering term |

Preferred engineering wording: **modular assembly**, **block-based composition**,
**lexeme-based composition**, **brick** (UI toolkit sense only).

Disclaimer when LEGO® appears in comparative prose:

> LEGO® is a trademark of the LEGO Group. This project is not affiliated with,
> endorsed by, or sponsored by the LEGO Group.

---

## Practical defaults for authors

1. In ADRs and Go code, use English linguistic terms: `lexeme`, `lemma`,
   `wordform`, `morphology`, `sense`, `attestation`.  
2. Explain *legō* only when teaching etymology.  
3. For UI composition docs, say **brick** / **modular block**, never “lego”.  
4. Keep dictionary content out of `fastygo/context` core.
