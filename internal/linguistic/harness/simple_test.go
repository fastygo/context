package harness_test

import (
	"testing"

	"github.com/fastygo/context/internal/linguistic/harness"
	"github.com/fastygo/context/internal/linguistic/simple"
)

func TestSimpleAdapterSatisfiesLinguisticContract(t *testing.T) {
	t.Parallel()
	lemma, wordform := harness.DefaultExpanderMaps()
	harness.RunContract(t, harness.Ports{
		Normalizer: simple.Normalizer{},
		Analyzer:   simple.Analyzer{},
		Expander:   simple.Expander{LemmaMap: lemma, WordformMap: wordform},
	})
}
