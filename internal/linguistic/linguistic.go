// Package linguistic defines language-neutral contracts and morphology ports (ADR-0015).
package linguistic

import (
	"context"
	"fmt"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
)

// LanguageCode is a BCP 47 language tag.
type LanguageCode string

func (c LanguageCode) Validate() error {
	if c == "" {
		return fmt.Errorf("language_code: empty")
	}
	return nil
}

// ScriptCode is an ISO 15924 script tag.
type ScriptCode string

func (c ScriptCode) Validate() error {
	if c == "" {
		return fmt.Errorf("script_code: empty")
	}
	return nil
}

// FeatureScheme names a portable morphology feature vocabulary.
type FeatureScheme string

const (
	FeatureSchemeUD       FeatureScheme = "UD"
	FeatureSchemeUniMorph FeatureScheme = "UniMorph"
	FeatureSchemeAdapter  FeatureScheme = "adapter"
)

// AnalyzerVersion pins reproducible linguistic processing versions.
type AnalyzerVersion struct {
	AdapterID          string
	AdapterVersion     string
	NormalizerVersion  string
	TokenizerVersion   string
	AnalyzerVersion    string
	GeneratorVersion   string
	DictionaryVersion  DictionaryVersion
	FeatureScheme      FeatureScheme
	RawFeatureScheme   string
}

func (v AnalyzerVersion) Validate() error {
	if v.AdapterID == "" {
		return fmt.Errorf("analyzer_version: adapter_id empty")
	}
	return nil
}

// DictionaryVersion identifies a lexicon/dictionary resource revision used by an adapter.
type DictionaryVersion string

// Lemma is a canonical citation form; not a sense or concept.
type Lemma string

// WordForm is a surface or generated form with optional features.
type WordForm struct {
	Form     string
	Lemma    Lemma
	LexemeID ids.LexemeID
	Features MorphFeatureSet
}

func (w WordForm) Validate() error {
	if w.Form == "" {
		return fmt.Errorf("wordform: empty form")
	}
	return nil
}

// MorphFeatureSet is a portable feature bundle.
type MorphFeatureSet struct {
	Scheme     FeatureScheme
	Features   map[string]string
	RawScheme  string
	RawFeatures map[string]string
}

// TokenOccurrence is an offset-preserving token in a source span.
type TokenOccurrence struct {
	ID                ids.TokenID
	ProjectID         ids.ProjectID
	SourceID          ids.SourceID
	ChunkID           ids.ChunkID
	Language          LanguageCode
	Script            ScriptCode
	Surface           string
	Normalized        string
	Span              foundation.ByteSpan
	TokenizerVersion  string
	NormalizerVersion string
}

func (t TokenOccurrence) Validate() error {
	if err := t.ID.Validate(); err != nil {
		return err
	}
	if err := t.ProjectID.Validate(); err != nil {
		return err
	}
	if err := t.SourceID.Validate(); err != nil {
		return err
	}
	if err := t.ChunkID.Validate(); err != nil {
		return err
	}
	if err := t.Language.Validate(); err != nil {
		return err
	}
	if t.Surface == "" {
		return fmt.Errorf("token surface: empty")
	}
	return t.Span.Validate()
}

// MorphAnalysis is one candidate analysis of a token; ambiguity is explicit.
type MorphAnalysis struct {
	ID                ids.MorphAnalysisID
	TokenID           ids.TokenID
	Language          LanguageCode
	Lemma             Lemma
	LexemeID          ids.LexemeID
	PartOfSpeech      string
	Features          MorphFeatureSet
	Confidence        float64
	Selected          bool
	SelectionReason   string
	AnalyzerID        string
	AnalyzerVersion   string
	DictionaryVersion DictionaryVersion
}

func (m MorphAnalysis) Validate() error {
	if err := m.ID.Validate(); err != nil {
		return err
	}
	if err := m.TokenID.Validate(); err != nil {
		return err
	}
	if err := m.Language.Validate(); err != nil {
		return err
	}
	if m.AnalyzerID == "" {
		return fmt.Errorf("morph_analysis: analyzer_id empty")
	}
	return nil
}

// ExpansionType classifies explainable query expansions (ADR-0015).
type ExpansionType string

const (
	ExpansionLemma           ExpansionType = "lemma"
	ExpansionWordform        ExpansionType = "wordform"
	ExpansionCompound        ExpansionType = "compound"
	ExpansionAccent          ExpansionType = "accent"
	ExpansionFuzzy           ExpansionType = "fuzzy"
	ExpansionSynonym         ExpansionType = "synonym"
	ExpansionTransliteration ExpansionType = "transliteration"
)

func (t ExpansionType) Validate() error {
	switch t {
	case ExpansionLemma, ExpansionWordform, ExpansionCompound, ExpansionAccent,
		ExpansionFuzzy, ExpansionSynonym, ExpansionTransliteration:
		return nil
	case "":
		return fmt.Errorf("expansion_type: empty")
	default:
		return fmt.Errorf("expansion_type: unknown %q", t)
	}
}

// QueryExpansion is an explainable lexical or morphology-driven expansion.
type QueryExpansion struct {
	ID             ids.ExpansionID
	QueryID        ids.QueryID
	Language       LanguageCode
	OriginalTerm   string
	ExpandedTerm   string
	Type           ExpansionType
	Features       MorphFeatureSet
	Confidence     float64
	Reason         string
	AdapterID      string
	AdapterVersion string
}

func (q QueryExpansion) Validate() error {
	if err := q.ID.Validate(); err != nil {
		return err
	}
	if err := q.QueryID.Validate(); err != nil {
		return err
	}
	if q.OriginalTerm == "" || q.ExpandedTerm == "" {
		return fmt.Errorf("query_expansion: terms required")
	}
	if err := q.Type.Validate(); err != nil {
		return err
	}
	if q.AdapterID == "" {
		return fmt.Errorf("query_expansion: adapter_id empty")
	}
	return nil
}

// CapabilityMissing is reported when an adapter cannot fulfill a request.
const CapabilityMissing = "capability_missing"

// MorphAnalyzer analyzes tokens into candidate morphologies.
type MorphAnalyzer interface {
	Analyze(ctx context.Context, token TokenOccurrence) ([]MorphAnalysis, error)
}

// MorphGenerator generates wordforms for a lexeme and feature bundle.
type MorphGenerator interface {
	Generate(ctx context.Context, language LanguageCode, lexemeID ids.LexemeID, features MorphFeatureSet) ([]WordForm, error)
}

// LexicalNormalizer normalizes text without replacing original surface spans.
type LexicalNormalizer interface {
	Normalize(ctx context.Context, text string, language LanguageCode, script ScriptCode) (normalized string, version AnalyzerVersion, err error)
}

// QueryExpander produces explainable expansions; never silent rewrites of meaning.
type QueryExpander interface {
	Expand(ctx context.Context, queryID ids.QueryID, term string, language LanguageCode) ([]QueryExpansion, error)
}
