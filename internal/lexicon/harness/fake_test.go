package harness_test

import (
	"testing"

	"github.com/fastygo/context/internal/lexicon/fake"
	"github.com/fastygo/context/internal/lexicon/harness"
)

func TestFakeResourceSatisfiesLexiconContract(t *testing.T) {
	t.Parallel()
	seed := harness.DefaultSeed()
	res := fake.NewResource()
	res.Senses[seed.Sense.ID] = seed.Sense
	res.Concepts[seed.Concept.ID] = seed.Concept
	res.Attestations[seed.Attestation.ID] = seed.Attestation
	res.Sources[seed.Source.ID] = seed.Source
	harness.RunContract(t, res, seed)
}
