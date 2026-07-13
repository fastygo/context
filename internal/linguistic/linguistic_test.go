package linguistic_test

import (
	"testing"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/linguistic"
)

func TestTokenOccurrenceRejectsEmptySurface(t *testing.T) {
	t.Parallel()
	tok := linguistic.TokenOccurrence{
		ID:        "t1",
		ProjectID: "p1",
		SourceID:  "s1",
		ChunkID:   "c1",
		Language:  "en",
		Span:      foundation.ByteSpan{Start: 0, End: 1},
	}
	if err := tok.Validate(); err == nil {
		t.Fatal("expected empty surface to fail")
	}
}

func TestQueryExpansionTypeValidation(t *testing.T) {
	t.Parallel()
	if err := linguistic.ExpansionLemma.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := linguistic.ExpansionType("telepathy").Validate(); err == nil {
		t.Fatal("unknown expansion type should fail")
	}
}

func TestMorphAnalysisRequiresAnalyzer(t *testing.T) {
	t.Parallel()
	m := linguistic.MorphAnalysis{
		ID:       "m1",
		TokenID:  "t1",
		Language: "en",
	}
	if err := m.Validate(); err == nil {
		t.Fatal("expected missing analyzer_id to fail")
	}
}
