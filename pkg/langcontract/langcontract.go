// Package langcontract is the public, language-neutral adapter surface (ADR-0037 / A2).
// External context-lang-* repositories depend on this package — never on internal/.
package langcontract

import (
	"context"
	"fmt"
)

// LanguageCode is a BCP 47 language tag.
type LanguageCode string

// ScriptCode is an ISO 15924 script tag.
type ScriptCode string

// FeatureScheme names a portable morphology feature vocabulary.
type FeatureScheme string

const (
	FeatureSchemeUD       FeatureScheme = "UD"
	FeatureSchemeUniMorph FeatureScheme = "UniMorph"
	FeatureSchemeAdapter  FeatureScheme = "adapter"
)

// ByteSpan is a half-open byte range [Start, End) into UTF-8 text.
type ByteSpan struct {
	Start uint64 `json:"start"`
	End   uint64 `json:"end"`
}

func (s ByteSpan) Validate() error {
	if s.Start >= s.End {
		return fmt.Errorf("byte span: start %d must be < end %d", s.Start, s.End)
	}
	return nil
}

// AnalyzerVersion pins reproducible linguistic processing versions.
type AnalyzerVersion struct {
	AdapterID         string        `json:"adapter_id"`
	AdapterVersion    string        `json:"adapter_version"`
	NormalizerVersion string        `json:"normalizer_version,omitempty"`
	TokenizerVersion  string        `json:"tokenizer_version,omitempty"`
	AnalyzerVersion   string        `json:"analyzer_version,omitempty"`
	DictionaryVersion string        `json:"dictionary_version,omitempty"`
	FeatureScheme     FeatureScheme `json:"feature_scheme,omitempty"`
}

func (v AnalyzerVersion) Validate() error {
	if v.AdapterID == "" {
		return fmt.Errorf("analyzer_version: adapter_id empty")
	}
	return nil
}

// TokenOccurrence is an offset-preserving token in a source span.
type TokenOccurrence struct {
	ID                string       `json:"id"`
	ProjectID         string       `json:"project_id"`
	SourceID          string       `json:"source_id"`
	ChunkID           string       `json:"chunk_id"`
	Language          LanguageCode `json:"language"`
	Script            ScriptCode   `json:"script"`
	Surface           string       `json:"surface"`
	Normalized        string       `json:"normalized"`
	Span              ByteSpan     `json:"span"`
	TokenizerVersion  string       `json:"tokenizer_version"`
	NormalizerVersion string       `json:"normalizer_version"`
}

// MorphAnalysis is one candidate analysis; ambiguity is explicit.
type MorphAnalysis struct {
	ID              string       `json:"id"`
	TokenID         string       `json:"token_id"`
	Language        LanguageCode `json:"language"`
	Lemma           string       `json:"lemma"`
	LexemeID        string       `json:"lexeme_id,omitempty"`
	Confidence      float64      `json:"confidence"`
	Selected        bool         `json:"selected"`
	SelectionReason string       `json:"selection_reason,omitempty"`
	AnalyzerID      string       `json:"analyzer_id"`
	AnalyzerVersion string       `json:"analyzer_version"`
}

func (m MorphAnalysis) Validate() error {
	if m.ID == "" || m.TokenID == "" {
		return fmt.Errorf("morph_analysis: id and token_id required")
	}
	if m.Language == "" {
		return fmt.Errorf("morph_analysis: language empty")
	}
	if m.AnalyzerID == "" || m.AnalyzerVersion == "" {
		return fmt.Errorf("morph_analysis: analyzer pin required")
	}
	return nil
}

// ExpansionType classifies explainable query expansions.
type ExpansionType string

const (
	ExpansionLemma    ExpansionType = "lemma"
	ExpansionWordform ExpansionType = "wordform"
)

// QueryExpansion is an explainable expansion candidate.
type QueryExpansion struct {
	ID             string         `json:"id"`
	QueryID        string         `json:"query_id"`
	Language       LanguageCode   `json:"language"`
	OriginalTerm   string         `json:"original_term"`
	ExpandedTerm   string         `json:"expanded_term"`
	Type           ExpansionType  `json:"type"`
	Confidence     float64        `json:"confidence"`
	Reason         string         `json:"reason"`
	AdapterID      string         `json:"adapter_id"`
	AdapterVersion string         `json:"adapter_version"`
}

func (q QueryExpansion) Validate() error {
	if q.ID == "" || q.QueryID == "" {
		return fmt.Errorf("query_expansion: ids required")
	}
	if q.OriginalTerm == "" || q.ExpandedTerm == "" {
		return fmt.Errorf("query_expansion: terms required")
	}
	if q.AdapterID == "" || q.AdapterVersion == "" {
		return fmt.Errorf("query_expansion: adapter pin required")
	}
	return nil
}

// MorphAnalyzer analyzes tokens into candidate morphologies.
type MorphAnalyzer interface {
	Analyze(ctx context.Context, token TokenOccurrence) ([]MorphAnalysis, error)
}

// LexicalNormalizer normalizes text without replacing original surface spans.
type LexicalNormalizer interface {
	Normalize(ctx context.Context, text string, language LanguageCode, script ScriptCode) (normalized string, version AnalyzerVersion, err error)
}

// QueryExpander produces explainable expansions.
type QueryExpander interface {
	Expand(ctx context.Context, queryID string, term string, language LanguageCode) ([]QueryExpansion, error)
}
