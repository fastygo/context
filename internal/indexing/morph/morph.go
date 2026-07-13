// Package morph provides no-op/simple language hooks for indexing traces.
package morph

import (
	"context"

	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/linguistic"
)

const (
	AdapterID         = "noop-lang"
	AdapterVersion    = "noop-v1"
	AnalyzerVersion   = "noop-analyzer-v1"
	DictionaryVersion = "none"
	FeatureScheme     = linguistic.FeatureSchemeUD
)

// Hook records tokenizer/analyzer versions without performing morphology.
type Hook struct{}

func (Hook) Version() linguistic.AnalyzerVersion {
	return linguistic.AnalyzerVersion{
		AdapterID:         AdapterID,
		AdapterVersion:    AdapterVersion,
		NormalizerVersion: "identity-v1",
		TokenizerVersion:  "whitespace-v1",
		AnalyzerVersion:   AnalyzerVersion,
		DictionaryVersion: DictionaryVersion,
		FeatureScheme:     FeatureScheme,
	}
}

func (h Hook) Analyze(_ context.Context, token linguistic.TokenOccurrence) ([]linguistic.MorphAnalysis, error) {
	return []linguistic.MorphAnalysis{{
		ID:                ids.MorphAnalysisID(string(token.ID) + ":noop"),
		TokenID:           token.ID,
		Language:          token.Language,
		Lemma:             linguistic.Lemma(token.Normalized),
		Confidence:        0,
		Selected:          false,
		SelectionReason:   linguistic.CapabilityMissing,
		AnalyzerID:        AdapterID,
		AnalyzerVersion:   AnalyzerVersion,
		DictionaryVersion: DictionaryVersion,
	}}, nil
}

var _ linguistic.MorphAnalyzer = Hook{}
