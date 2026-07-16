package ru_test

import (
	"context"
	"testing"

	ru "github.com/fastygo/context/pkg/lang/ru"
	"github.com/fastygo/context/pkg/langcontract"
	"github.com/fastygo/context/pkg/langtestkit"
)

// TestPublicContract proves context-lang-ru passes the published harness
// exactly the way an external adapter repository would (A1/A2, ADR-0037).
func TestPublicContract(t *testing.T) {
	langtestkit.RunContract(t, ru.Ports())
}

func TestAnalyzerRussianAmbiguity(t *testing.T) {
	t.Parallel()
	token := langcontract.TokenOccurrence{
		ID: "tok1", ProjectID: "p", SourceID: "s", ChunkID: "c",
		Language: "ru", Script: "Cyrl",
		Surface: "дорогой", Normalized: "дорогой",
		Span:             langcontract.ByteSpan{Start: 0, End: 14},
		TokenizerVersion: "t-v1", NormalizerVersion: "n-v1",
	}
	analyses, err := (ru.Analyzer{}).Analyze(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if len(analyses) < 2 {
		t.Fatalf("ambiguity must be explicit; got %d analyses", len(analyses))
	}
	selected := 0
	lemmaSet := map[string]bool{}
	for _, a := range analyses {
		if err := a.Validate(); err != nil {
			t.Fatalf("invalid analysis: %v", err)
		}
		if a.Selected {
			selected++
		}
		lemmaSet[a.Lemma] = true
	}
	if selected != 1 {
		t.Fatalf("exactly one analysis must be selected, got %d", selected)
	}
	if !lemmaSet["дорога"] || !lemmaSet["дорогой"] {
		t.Fatalf("want both дорога/дорогой lemma candidates, got %v", lemmaSet)
	}
	if token.Surface != "дорогой" {
		t.Fatal("surface mutated")
	}
}

func TestExpanderRussianParadigm(t *testing.T) {
	t.Parallel()
	exps, err := (ru.Expander{}).Expand(context.Background(), "q1", "дорога", "ru")
	if err != nil {
		t.Fatal(err)
	}
	if len(exps) == 0 {
		t.Fatal("expected ru expansions")
	}
	want := map[string]bool{"дороги": false, "дорогу": false}
	for _, e := range exps {
		if err := e.Validate(); err != nil {
			t.Fatalf("invalid expansion: %v", err)
		}
		if e.OriginalTerm != "дорога" {
			t.Fatalf("original_term=%q", e.OriginalTerm)
		}
		if e.Reason == "" {
			t.Fatalf("expansion without reason: %#v", e)
		}
		if _, ok := want[e.ExpandedTerm]; ok {
			want[e.ExpandedTerm] = true
		}
	}
	for form, found := range want {
		if !found {
			t.Errorf("missing expansion %q", form)
		}
	}
}

func TestExpanderOtherLanguageEmpty(t *testing.T) {
	t.Parallel()
	exps, err := (ru.Expander{}).Expand(context.Background(), "q1", "Straße", "de")
	if err != nil {
		t.Fatal(err)
	}
	if len(exps) != 0 {
		t.Fatalf("de term must not expand via ru adapter, got %#v", exps)
	}
}

func TestNormalizerFoldsForMatching(t *testing.T) {
	t.Parallel()
	out, ver, err := (ru.Normalizer{}).Normalize(context.Background(), "Жёлтая ДОРОГА", "ru", "Cyrl")
	if err != nil {
		t.Fatal(err)
	}
	if out != "желтая дорога" {
		t.Fatalf("normalize=%q", out)
	}
	if ver.AdapterID != ru.AdapterID || ver.NormalizerVersion == "" {
		t.Fatalf("version pin: %#v", ver)
	}
}
