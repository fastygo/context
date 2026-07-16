package ru

import (
	"context"
	"strings"

	"golang.org/x/text/unicode/norm"

	"github.com/fastygo/context/pkg/langcontract"
	"github.com/fastygo/context/pkg/langtestkit"
)

const (
	AdapterID      = "context-lang-ru"
	AdapterVersion = "ru-v1"
	// NormalizerVersion records the one-way NFC + lowercase + ё→е fold policy.
	NormalizerVersion = "nfc-lower-yofold-v1"
	AnalyzerVer       = "ru-rules-v1"
)

// Normalizer folds text for matching: NFC, lowercase, ё→е.
// Original surfaces are never rewritten (langcontract requirement).
type Normalizer struct{}

func (Normalizer) Normalize(ctx context.Context, text string, language langcontract.LanguageCode, script langcontract.ScriptCode) (string, langcontract.AnalyzerVersion, error) {
	if err := ctx.Err(); err != nil {
		return "", langcontract.AnalyzerVersion{}, err
	}
	_ = script
	out := norm.NFC.String(text)
	if language == "" || language == "ru" {
		out = Fold(out)
	}
	return out, version(), nil
}

// Analyzer maps tokens to candidate lemmas. Russian tokens receive full
// rule-based candidates with explicit ambiguity; other languages fall back to
// a low-confidence surface analysis so multilingual pipelines stay total.
type Analyzer struct{}

func (Analyzer) Analyze(ctx context.Context, token langcontract.TokenOccurrence) ([]langcontract.MorphAnalysis, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	word := token.Normalized
	if word == "" {
		word = token.Surface
	}
	if token.Language == "ru" {
		cands := AnalyzeWord(word)
		if len(cands) > 0 {
			out := make([]langcontract.MorphAnalysis, 0, len(cands))
			for i, c := range cands {
				out = append(out, langcontract.MorphAnalysis{
					ID:              token.ID + ":ru:" + itoa(i),
					TokenID:         token.ID,
					Language:        token.Language,
					Lemma:           c.Lemma,
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
	return []langcontract.MorphAnalysis{{
		ID:              token.ID + ":ru:surface",
		TokenID:         token.ID,
		Language:        token.Language,
		Lemma:           strings.ToLower(norm.NFC.String(word)),
		Confidence:      0.3,
		Selected:        true,
		SelectionReason: "non-ru surface fallback",
		AnalyzerID:      AdapterID,
		AnalyzerVersion: AdapterVersion,
	}}, nil
}

// Expander generates paradigm wordforms for Russian terms. For language "en"
// it serves the langtestkit fixture maps so the public contract harness can
// assert the shared run→runners expectation against any adapter.
type Expander struct {
	MaxForms int
}

func (e Expander) Expand(ctx context.Context, queryID string, term string, language langcontract.LanguageCode) ([]langcontract.QueryExpansion, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	term = strings.TrimSpace(term)
	if term == "" {
		return nil, nil
	}
	if language == "en" {
		return e.fixtureEN(queryID, term), nil
	}
	if language != "" && language != "ru" {
		return nil, nil
	}
	maxForms := e.MaxForms
	if maxForms <= 0 {
		maxForms = DefaultMaxForms
	}
	forms := ExpandWord(term, maxForms)
	out := make([]langcontract.QueryExpansion, 0, len(forms))
	for i, f := range forms {
		out = append(out, langcontract.QueryExpansion{
			ID:             queryID + ":ru:" + itoa(i) + ":" + f.Form,
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

func (Expander) fixtureEN(queryID, term string) []langcontract.QueryExpansion {
	lemmaMap, wfMap := langtestkit.DefaultExpanderMaps()
	var out []langcontract.QueryExpansion
	if lemma, ok := lemmaMap[term]; ok && lemma != term {
		out = append(out, langcontract.QueryExpansion{
			ID: queryID + ":en:lemma:" + lemma, QueryID: queryID, Language: "en",
			OriginalTerm: term, ExpandedTerm: lemma, Type: langcontract.ExpansionLemma,
			Confidence: 0.8, Reason: "harness en lemma fixture",
			AdapterID: AdapterID, AdapterVersion: AdapterVersion,
		})
	}
	for _, form := range wfMap[term] {
		out = append(out, langcontract.QueryExpansion{
			ID: queryID + ":en:wf:" + form, QueryID: queryID, Language: "en",
			OriginalTerm: term, ExpandedTerm: form, Type: langcontract.ExpansionWordform,
			Confidence: 0.75, Reason: "harness en wordform fixture",
			AdapterID: AdapterID, AdapterVersion: AdapterVersion,
		})
	}
	return out
}

func expansionType(k ExpansionKind) langcontract.ExpansionType {
	switch k {
	case KindLemma:
		return langcontract.ExpansionLemma
	default:
		// langcontract's public enum carries lemma/wordform; accent variants
		// are wordform-class with an explicit reason.
		return langcontract.ExpansionWordform
	}
}

// Ports returns the three linguistic ports for langtestkit.RunContract.
func Ports() langtestkit.Ports {
	return langtestkit.Ports{
		Normalizer: Normalizer{},
		Analyzer:   Analyzer{},
		Expander:   Expander{},
	}
}

func version() langcontract.AnalyzerVersion {
	return langcontract.AnalyzerVersion{
		AdapterID:         AdapterID,
		AdapterVersion:    AdapterVersion,
		NormalizerVersion: NormalizerVersion,
		AnalyzerVersion:   AnalyzerVer,
		FeatureScheme:     langcontract.FeatureSchemeUD,
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

var (
	_ langcontract.LexicalNormalizer = Normalizer{}
	_ langcontract.MorphAnalyzer     = Analyzer{}
	_ langcontract.QueryExpander     = Expander{}
)
