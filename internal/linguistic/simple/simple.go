// Package simple provides no-op/simple linguistic adapters for tests (ADR-0015).
package simple

import (
	"context"
	"strings"

	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing/normalize"
	"github.com/fastygo/context/internal/linguistic"
)

// Normalizer applies ADR-0018 hashing normalization for query/text hooks.
type Normalizer struct{}

func (Normalizer) Normalize(ctx context.Context, text string, language linguistic.LanguageCode, script linguistic.ScriptCode) (string, linguistic.AnalyzerVersion, error) {
	if err := ctx.Err(); err != nil {
		return "", linguistic.AnalyzerVersion{}, err
	}
	out, err := normalize.ForHashing([]byte(text))
	if err != nil {
		return "", linguistic.AnalyzerVersion{}, err
	}
	return out, linguistic.AnalyzerVersion{
		AdapterID:         "simple-lang",
		AdapterVersion:    "simple-v1",
		NormalizerVersion: "nfc-lf-v1",
		FeatureScheme:     linguistic.FeatureSchemeUD,
	}, nil
}

// Analyzer returns a single lemma=surface analysis.
type Analyzer struct{}

func (Analyzer) Analyze(ctx context.Context, token linguistic.TokenOccurrence) ([]linguistic.MorphAnalysis, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []linguistic.MorphAnalysis{{
		ID:              ids.MorphAnalysisID(string(token.ID) + ":simple"),
		TokenID:         token.ID,
		Language:        token.Language,
		Lemma:           linguistic.Lemma(strings.ToLower(token.Normalized)),
		Confidence:      0.5,
		Selected:        true,
		SelectionReason: "simple_lower",
		AnalyzerID:      "simple-lang",
		AnalyzerVersion: "simple-v1",
	}}, nil
}

// Generator echoes the lemma as a wordform.
type Generator struct{}

func (Generator) Generate(ctx context.Context, language linguistic.LanguageCode, lexemeID ids.LexemeID, features linguistic.MorphFeatureSet) ([]linguistic.WordForm, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []linguistic.WordForm{{
		Form:     string(lexemeID),
		Lemma:    linguistic.Lemma(lexemeID),
		LexemeID: lexemeID,
		Features: features,
	}}, nil
}

// Expander expands known fixtures; unknown terms yield no expansions (rejectable).
type Expander struct {
	// LemmaMap maps surface -> lemma for lemma expansions.
	LemmaMap map[string]string
	// WordformMap maps lemma -> extra wordforms.
	WordformMap map[string][]string
}

func (e Expander) Expand(ctx context.Context, queryID ids.QueryID, term string, language linguistic.LanguageCode) ([]linguistic.QueryExpansion, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	term = strings.TrimSpace(term)
	var out []linguistic.QueryExpansion
	if lemma, ok := e.LemmaMap[term]; ok && lemma != term {
		out = append(out, linguistic.QueryExpansion{
			ID:             ids.ExpansionID(string(queryID) + ":lemma:" + lemma),
			QueryID:        queryID,
			Language:       language,
			OriginalTerm:   term,
			ExpandedTerm:   lemma,
			Type:           linguistic.ExpansionLemma,
			Confidence:     0.8,
			Reason:         "simple lemma map",
			AdapterID:      "simple-lang",
			AdapterVersion: "simple-v1",
		})
	}
	if forms, ok := e.WordformMap[term]; ok {
		for _, form := range forms {
			out = append(out, linguistic.QueryExpansion{
				ID:             ids.ExpansionID(string(queryID) + ":wf:" + form),
				QueryID:        queryID,
				Language:       language,
				OriginalTerm:   term,
				ExpandedTerm:   form,
				Type:           linguistic.ExpansionWordform,
				Confidence:     0.6,
				Reason:         "simple wordform map",
				AdapterID:      "simple-lang",
				AdapterVersion: "simple-v1",
			})
		}
	}
	return out, nil
}

var (
	_ linguistic.LexicalNormalizer = Normalizer{}
	_ linguistic.MorphAnalyzer     = Analyzer{}
	_ linguistic.MorphGenerator    = Generator{}
	_ linguistic.QueryExpander     = Expander{}
)
