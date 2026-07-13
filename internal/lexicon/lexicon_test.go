package lexicon_test

import (
	"testing"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/lexicon"
)

func TestSenseRejectsZeroValue(t *testing.T) {
	t.Parallel()
	if err := (lexicon.Sense{}).Validate(); err == nil {
		t.Fatal("expected zero sense to fail")
	}
}

func TestAttestationRequiresQuoteAndSpan(t *testing.T) {
	t.Parallel()
	a := lexicon.Attestation{
		ID:        "att1",
		ProjectID: "p1",
		SourceID:  "s1",
		Quote:     "example",
		Span:      foundation.ByteSpan{Start: 0, End: 7},
	}
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
	a.Quote = ""
	if err := a.Validate(); err == nil {
		t.Fatal("empty quote should fail")
	}
}

func TestConceptRequiresPreferredLabel(t *testing.T) {
	t.Parallel()
	c := lexicon.Concept{ID: "c1", ProjectID: "p1"}
	if err := c.Validate(); err == nil {
		t.Fatal("expected missing label to fail")
	}
}
