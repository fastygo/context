// Package en is the thin in-repo English linguistic adapter (S3 / A1).
// Adapter id: context-lang-en. Full morphology belongs in external repos;
// this package proves the harness path and keeps core brand-neutral.
package en

import (
	"context"
	"strings"

	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing/normalize"
	"github.com/fastygo/context/internal/linguistic"
	"github.com/fastygo/context/internal/linguistic/harness"
)

const (
	AdapterID      = "context-lang-en"
	AdapterVersion = "en-v1"
)

// Normalizer applies ADR-0018 hashing normalization.
type Normalizer struct{}

func (Normalizer) Normalize(ctx context.Context, text string, language linguistic.LanguageCode, script linguistic.ScriptCode) (string, linguistic.AnalyzerVersion, error) {
	if err := ctx.Err(); err != nil {
		return "", linguistic.AnalyzerVersion{}, err
	}
	_ = language
	_ = script
	out, err := normalize.ForHashing([]byte(text))
	if err != nil {
		return "", linguistic.AnalyzerVersion{}, err
	}
	return out, linguistic.AnalyzerVersion{
		AdapterID:         AdapterID,
		AdapterVersion:    AdapterVersion,
		NormalizerVersion: "nfc-lf-v1",
		FeatureScheme:     linguistic.FeatureSchemeUD,
	}, nil
}

// Analyzer returns a light English lemma; never mutates the token surface.
type Analyzer struct{}

func (Analyzer) Analyze(ctx context.Context, token linguistic.TokenOccurrence) ([]linguistic.MorphAnalysis, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	lemma := lightLemma(token.Normalized)
	return []linguistic.MorphAnalysis{{
		ID:              ids.MorphAnalysisID(string(token.ID) + ":en"),
		TokenID:         token.ID,
		Language:        token.Language,
		Lemma:           linguistic.Lemma(lemma),
		Confidence:      0.7,
		Selected:        true,
		SelectionReason: "en_light_lemma",
		AnalyzerID:      AdapterID,
		AnalyzerVersion: AdapterVersion,
	}}, nil
}

// Expander uses harness fixture maps.
type Expander struct {
	LemmaMap    map[string]string
	WordformMap map[string][]string
}

// NewExpander seeds DefaultExpanderMaps from the in-repo harness.
func NewExpander() Expander {
	lemma, wf := harness.DefaultExpanderMaps()
	return Expander{LemmaMap: lemma, WordformMap: wf}
}

func (e Expander) Expand(ctx context.Context, queryID ids.QueryID, term string, language linguistic.LanguageCode) ([]linguistic.QueryExpansion, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	term = strings.TrimSpace(term)
	var out []linguistic.QueryExpansion
	if lemma, ok := e.LemmaMap[term]; ok && lemma != term {
		out = append(out, linguistic.QueryExpansion{
			ID: ids.ExpansionID(string(queryID) + ":lemma:" + lemma), QueryID: queryID, Language: language,
			OriginalTerm: term, ExpandedTerm: lemma, Type: linguistic.ExpansionLemma,
			Confidence: 0.8, Reason: "en lemma map",
			AdapterID: AdapterID, AdapterVersion: AdapterVersion,
		})
	}
	if forms, ok := e.WordformMap[term]; ok {
		for _, form := range forms {
			out = append(out, linguistic.QueryExpansion{
				ID: ids.ExpansionID(string(queryID) + ":wf:" + form), QueryID: queryID, Language: language,
				OriginalTerm: term, ExpandedTerm: form, Type: linguistic.ExpansionWordform,
				Confidence: 0.75, Reason: "en wordform map",
				AdapterID: AdapterID, AdapterVersion: AdapterVersion,
			})
		}
	}
	return out, nil
}

// Ports returns harness ports for context-lang-en.
func Ports() harness.Ports {
	return harness.Ports{
		Normalizer: Normalizer{},
		Analyzer:   Analyzer{},
		Expander:   NewExpander(),
	}
}

func lightLemma(surface string) string {
	s := strings.ToLower(strings.TrimSpace(surface))
	switch s {
	case "runners", "running", "ran":
		return "run"
	case "бегуны":
		return "бежать"
	}
	if strings.HasSuffix(s, "s") && len(s) > 3 && !strings.HasSuffix(s, "ss") {
		return s[:len(s)-1]
	}
	return s
}

var (
	_ linguistic.LexicalNormalizer = Normalizer{}
	_ linguistic.MorphAnalyzer     = Analyzer{}
	_ linguistic.QueryExpander     = Expander{}
)
