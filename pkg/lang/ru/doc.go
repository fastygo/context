// Package ru is the in-repo Russian linguistic adapter (context-lang-ru).
//
// It is a rule-based morphology engine: no bundled dictionaries, no network
// resources. It analyzes surface wordforms into explicit candidate lemmas
// (ambiguity is preserved, never collapsed) and generates paradigm wordforms
// for explainable query expansion. Coverage is intentionally heuristic:
// declension/conjugation ending tables plus a tiny curated exception fixture.
// Rich dictionary-backed morphology (OpenCorpora-scale) belongs in external
// context-lang-* repositories per the language adapter plugin roadmap; this
// package proves the contract path with useful real-world recall.
//
// Guarantees:
//   - Original surfaces and spans are never mutated (langcontract).
//   - Every expansion carries type, reason, confidence, and adapter pins.
//   - Matching is case-folded and ё→е folded ("yo fold"); generated forms are
//     emitted in the folded normal form. The fold direction is one-way and
//     recorded in the normalizer version.
//   - Analysis returns multiple candidates ordered deterministically.
package ru
