package jsonres_test

import (
	"testing"

	"github.com/fastygo/context/internal/lexicon/harness"
	"github.com/fastygo/context/internal/lexicon/jsonres"
)

func TestCuratedJSONPassesLexiconHarness(t *testing.T) {
	t.Parallel()
	ad, err := jsonres.Load(jsonres.HarnessSeedJSON())
	if err != nil {
		t.Fatal(err)
	}
	harness.RunContract(t, ad, harness.DefaultSeed())
}
