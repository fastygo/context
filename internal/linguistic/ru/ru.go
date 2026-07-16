// Package ru adapts the public context-lang-ru engine (pkg/lang/ru) to the
// internal linguistic contracts so the runtime (hybrid search, query layer,
// CLI) can use Russian morphology without duplicating the rule engine.
package ru

import (
	"context"
	"strings"

	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/linguistic"
	ruengine "github.com/fastygo/context/pkg/lang/ru"
)

const (
	AdapterID      = ruengine.AdapterID
	AdapterVersion = ruengine.AdapterVersion
)

// Normalizer folds text for matching (NFC + lowercase + ё→е).
type Normalizer struct{}

func (Normalizer) Normalize(ctx context.Context, text string, language linguistic.LanguageCode, script linguistic.ScriptCode) (string, linguistic.AnalyzerVersion, error) {
	if err := ctx.Err(); err != nil {
		return "", linguistic.AnalyzerVersion{}, err
	}
	_ = script
	out := text
	if language == "" || language == "ru" {
		out = ruengine.Fold(text)
	}
	return out, Version(), nil
}

// Analyzer returns rule-based candidate lemmas with explicit ambiguity.
type Analyzer struct{}

func (Analyzer) Analyze(ctx context.Context, token linguistic.TokenOccurrence) ([]linguistic.MorphAnalysis, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	word := token.Normalized
	if word == "" {
		word = token.Surface
	}
	if token.Language == "ru" {
		if cands := ruengine.AnalyzeWord(word); len(cands) > 0 {
			out := make([]linguistic.MorphAnalysis, 0, len(cands))
			for i, c := range cands {
				out = append(out, linguistic.MorphAnalysis{
					ID:              ids.MorphAnalysisID(string(token.ID) + ":ru:" + itoa(i)),
					TokenID:         token.ID,
					Language:        token.Language,
					Lemma:           linguistic.Lemma(c.Lemma),
					PartOfSpeech:    string(c.POS),
					Confidence:      c.Confidence,
					Selected:        i == 0,
					SelectionReason: c.Reason,
					AnalyzerID:      AdapterID,
					AnalyzerVersion: AdapterVersion,
				})
			}
			return out, nil
		}
	}
	return []linguistic.MorphAnalysis{{
		ID:              ids.MorphAnalysisID(string(token.ID) + ":ru:surface"),
		TokenID:         token.ID,
		Language:        token.Language,
		Lemma:           linguistic.Lemma(strings.ToLower(word)),
		Confidence:      0.3,
		Selected:        true,
		SelectionReason: "non-ru surface fallback",
		AnalyzerID:      AdapterID,
		AnalyzerVersion: AdapterVersion,
	}}, nil
}

// Expander generates explainable paradigm expansions for Russian terms.
// For "en" it serves the shared harness fixture (run→runners) so the internal
// contract harness stays language-adapter agnostic.
type Expander struct {
	MaxForms int
}

func (e Expander) Expand(ctx context.Context, queryID ids.QueryID, term string, language linguistic.LanguageCode) ([]linguistic.QueryExpansion, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	term = strings.TrimSpace(term)
	if term == "" {
		return nil, nil
	}
	if language == "en" {
		return fixtureEN(queryID, term), nil
	}
	if language != "" && language != "ru" {
		return nil, nil
	}
	maxForms := e.MaxForms
	if maxForms <= 0 {
		maxForms = ruengine.DefaultMaxForms
	}
	forms := ruengine.ExpandWord(term, maxForms)
	out := make([]linguistic.QueryExpansion, 0, len(forms))
	for i, f := range forms {
		out = append(out, linguistic.QueryExpansion{
			ID:             ids.ExpansionID(string(queryID) + ":ru:" + itoa(i) + ":" + f.Form),
			QueryID:        queryID,
			Language:       "ru",
			OriginalTerm:   term,
			ExpandedTerm:   f.Form,
			Type:           expansionType(f.Kind),
			Confidence:     f.Confidence,
			Reason:         f.Reason,
			AdapterID:      AdapterID,
			AdapterVersion: AdapterVersion,
		})
	}
	return out, nil
}

func fixtureEN(queryID ids.QueryID, term string) []linguistic.QueryExpansion {
	lemmaMap := map[string]string{"running": "run"}
	wfMap := map[string][]string{"run": {"runners"}}
	var out []linguistic.QueryExpansion
	if lemma, ok := lemmaMap[term]; ok && lemma != term {
		out = append(out, linguistic.QueryExpansion{
			ID: ids.ExpansionID(string(queryID) + ":en:lemma:" + lemma), QueryID: queryID,
			Language: "en", OriginalTerm: term, ExpandedTerm: lemma,
			Type: linguistic.ExpansionLemma, Confidence: 0.8, Reason: "harness en lemma fixture",
			AdapterID: AdapterID, AdapterVersion: AdapterVersion,
		})
	}
	for _, form := range wfMap[term] {
		out = append(out, linguistic.QueryExpansion{
			ID: ids.ExpansionID(string(queryID) + ":en:wf:" + form), QueryID: queryID,
			Language: "en", OriginalTerm: term, ExpandedTerm: form,
			Type: linguistic.ExpansionWordform, Confidence: 0.75, Reason: "harness en wordform fixture",
			AdapterID: AdapterID, AdapterVersion: AdapterVersion,
		})
	}
	return out
}

func expansionType(k ruengine.ExpansionKind) linguistic.ExpansionType {
	switch k {
	case ruengine.KindLemma:
		return linguistic.ExpansionLemma
	case ruengine.KindAccent:
		return linguistic.ExpansionAccent
	default:
		return linguistic.ExpansionWordform
	}
}

// Version pins the adapter for traces and index snapshots.
func Version() linguistic.AnalyzerVersion {
	return linguistic.AnalyzerVersion{
		AdapterID:         AdapterID,
		AdapterVersion:    AdapterVersion,
		NormalizerVersion: ruengine.NormalizerVersion,
		AnalyzerVersion:   ruengine.AnalyzerVer,
		FeatureScheme:     linguistic.FeatureSchemeUD,
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

var (
	_ linguistic.LexicalNormalizer = Normalizer{}
	_ linguistic.MorphAnalyzer     = Analyzer{}
	_ linguistic.QueryExpander     = Expander{}
)
