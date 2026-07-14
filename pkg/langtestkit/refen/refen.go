// Package refen is a reference English linguistic adapter for langtestkit CI (A1/A2).
// It is intentionally thin: light lemma map + fixture wordform expansion.
// Production morphology stays in external context-lang-* repositories.
package refen

import (
	"context"
	"strings"
	"unicode"

	"github.com/fastygo/context/pkg/langcontract"
	"github.com/fastygo/context/pkg/langtestkit"
)

const (
	AdapterID      = "context-lang-en"
	AdapterVersion = "en-v1"
)

// Normalizer applies NFC-ish whitespace collapse without replacing surfaces.
type Normalizer struct{}

func (Normalizer) Normalize(ctx context.Context, text string, language langcontract.LanguageCode, script langcontract.ScriptCode) (string, langcontract.AnalyzerVersion, error) {
	if err := ctx.Err(); err != nil {
		return "", langcontract.AnalyzerVersion{}, err
	}
	_ = language
	_ = script
	out := strings.Join(strings.Fields(text), " ")
	return out, langcontract.AnalyzerVersion{
		AdapterID:         AdapterID,
		AdapterVersion:    AdapterVersion,
		NormalizerVersion: "fields-v1",
		FeatureScheme:     langcontract.FeatureSchemeUD,
	}, nil
}

// Analyzer returns a light English lemma guess; never mutates the token.
type Analyzer struct{}

func (Analyzer) Analyze(ctx context.Context, token langcontract.TokenOccurrence) ([]langcontract.MorphAnalysis, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	lemma := lightLemma(token.Normalized)
	return []langcontract.MorphAnalysis{{
		ID:              token.ID + ":en",
		TokenID:         token.ID,
		Language:        token.Language,
		Lemma:           lemma,
		Confidence:      0.7,
		Selected:        true,
		SelectionReason: "en_light_lemma",
		AnalyzerID:      AdapterID,
		AnalyzerVersion: AdapterVersion,
	}}, nil
}

// Expander uses fixture maps (run→runners) required by langtestkit.
type Expander struct {
	LemmaMap    map[string]string
	WordformMap map[string][]string
}

// NewExpander returns an expander seeded with DefaultExpanderMaps.
func NewExpander() Expander {
	lemma, wf := langtestkit.DefaultExpanderMaps()
	return Expander{LemmaMap: lemma, WordformMap: wf}
}

func (e Expander) Expand(ctx context.Context, queryID string, term string, language langcontract.LanguageCode) ([]langcontract.QueryExpansion, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	term = strings.TrimSpace(term)
	var out []langcontract.QueryExpansion
	if lemma, ok := e.LemmaMap[term]; ok && lemma != term {
		out = append(out, langcontract.QueryExpansion{
			ID: queryID + ":lemma:" + lemma, QueryID: queryID, Language: language,
			OriginalTerm: term, ExpandedTerm: lemma, Type: langcontract.ExpansionLemma,
			Confidence: 0.8, Reason: "en lemma map",
			AdapterID: AdapterID, AdapterVersion: AdapterVersion,
		})
	}
	if forms, ok := e.WordformMap[term]; ok {
		for _, form := range forms {
			out = append(out, langcontract.QueryExpansion{
				ID: queryID + ":wf:" + form, QueryID: queryID, Language: language,
				OriginalTerm: term, ExpandedTerm: form, Type: langcontract.ExpansionWordform,
				Confidence: 0.75, Reason: "en wordform map",
				AdapterID: AdapterID, AdapterVersion: AdapterVersion,
			})
		}
	}
	return out, nil
}

// Ports returns the three ports for langtestkit.RunContract.
func Ports() langtestkit.Ports {
	return langtestkit.Ports{
		Normalizer: Normalizer{},
		Analyzer:   Analyzer{},
		Expander:   NewExpander(),
	}
}

func lightLemma(surface string) string {
	s := strings.ToLower(strings.TrimSpace(surface))
	if s == "" {
		return s
	}
	// Tiny closed set for harness fixtures; not a full stemmer.
	switch s {
	case "runners", "running", "ran":
		return "run"
	case "beguny", "бегуны":
		return "бежать"
	}
	if strings.HasSuffix(s, "ies") && len(s) > 4 {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(s, "s") && len(s) > 3 && !strings.HasSuffix(s, "ss") {
		r := []rune(s)
		if unicode.IsLetter(r[len(r)-2]) {
			return s[:len(s)-1]
		}
	}
	return s
}
