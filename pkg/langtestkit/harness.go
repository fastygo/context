// Package langtestkit is the published contract harness for context-lang-* adapters (A2).
// Adapter repos: go get github.com/fastygo/context/pkg/langtestkit
package langtestkit

import (
	"context"

	"github.com/fastygo/context/pkg/langcontract"
)

// Ports is the minimal linguistic surface a language adapter must satisfy.
type Ports struct {
	Normalizer langcontract.LexicalNormalizer
	Analyzer   langcontract.MorphAnalyzer
	Expander   langcontract.QueryExpander
}

// TB is the testing surface used by RunContract (*testing.T satisfies it).
type TB interface {
	Helper()
	Fatal(args ...any)
	Fatalf(format string, args ...any)
}

// FixtureEN is an English token whose surface must not be overwritten.
func FixtureEN() langcontract.TokenOccurrence {
	return langcontract.TokenOccurrence{
		ID: "tok_en_runners", ProjectID: "p_harness", SourceID: "s_en", ChunkID: "c_en",
		Language: "en", Script: "Latn",
		Surface: "Runners", Normalized: "Runners",
		Span:              langcontract.ByteSpan{Start: 4, End: 11},
		TokenizerVersion:  "harness-ws-v1",
		NormalizerVersion: "identity-v1",
	}
}

// FixtureRU is a second-language surface for multilingual contract coverage.
func FixtureRU() langcontract.TokenOccurrence {
	return langcontract.TokenOccurrence{
		ID: "tok_ru_beguny", ProjectID: "p_harness", SourceID: "s_ru", ChunkID: "c_ru",
		Language: "ru", Script: "Cyrl",
		Surface: "Бегуны", Normalized: "Бегуны",
		Span:              langcontract.ByteSpan{Start: 0, End: 12},
		TokenizerVersion:  "harness-ws-v1",
		NormalizerVersion: "identity-v1",
	}
}

// DefaultExpanderMaps are fixture expansions used by reference adapters in CI.
func DefaultExpanderMaps() (lemma map[string]string, wordform map[string][]string) {
	return map[string]string{"running": "run"},
		map[string][]string{"run": {"runners"}}
}

// RunContract asserts span preservation, analyzer version pins, original
// surface integrity, and explainable expansions for en + one additional language.
func RunContract(t TB, ports Ports) {
	t.Helper()
	if ports.Normalizer == nil || ports.Analyzer == nil || ports.Expander == nil {
		t.Fatal("langtestkit: all linguistic ports required")
	}
	ctx := context.Background()

	en := FixtureEN()
	surfaceBefore := en.Surface
	spanBefore := en.Span

	norm, ver, err := ports.Normalizer.Normalize(ctx, en.Surface, en.Language, en.Script)
	if err != nil {
		t.Fatalf("normalize en: %v", err)
	}
	if err := ver.Validate(); err != nil {
		t.Fatalf("analyzer_version: %v", err)
	}
	if ver.AdapterID == "" || ver.AdapterVersion == "" {
		t.Fatalf("normalize must pin adapter id/version: %#v", ver)
	}
	if en.Surface != surfaceBefore {
		t.Fatal("normalize must not mutate TokenOccurrence.Surface")
	}
	if norm == "" {
		t.Fatal("normalized text empty")
	}

	analyses, err := ports.Analyzer.Analyze(ctx, en)
	if err != nil {
		t.Fatalf("analyze en: %v", err)
	}
	if len(analyses) == 0 {
		t.Fatal("expected morph analyses")
	}
	for _, a := range analyses {
		if err := a.Validate(); err != nil {
			t.Fatalf("analysis invalid: %v", err)
		}
		if a.TokenID != en.ID {
			t.Fatalf("analysis token_id=%q want=%q", a.TokenID, en.ID)
		}
	}
	if en.Surface != surfaceBefore || en.Span != spanBefore {
		t.Fatal("analyze must not overwrite original surface or span")
	}

	ru := FixtureRU()
	ruSurface := ru.Surface
	ruAnalyses, err := ports.Analyzer.Analyze(ctx, ru)
	if err != nil {
		t.Fatalf("analyze ru: %v", err)
	}
	if len(ruAnalyses) == 0 {
		t.Fatal("expected ru analyses")
	}
	if ru.Surface != ruSurface {
		t.Fatal("ru surface must remain original")
	}
	if ruAnalyses[0].Language != "ru" {
		t.Fatalf("ru language=%q", ruAnalyses[0].Language)
	}

	exps, err := ports.Expander.Expand(ctx, "q_harness", "run", "en")
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	if len(exps) == 0 {
		t.Fatal("expected explainable expansion for fixture term run")
	}
	foundWF := false
	for _, e := range exps {
		if err := e.Validate(); err != nil {
			t.Fatalf("expansion: %v", err)
		}
		if e.OriginalTerm != "run" {
			t.Fatalf("original_term=%q", e.OriginalTerm)
		}
		if e.Type == langcontract.ExpansionWordform && e.ExpandedTerm == "runners" {
			foundWF = true
		}
	}
	if !foundWF {
		t.Fatalf("expected wordform expansion run→runners, got %#v", exps)
	}

	_, ver2, err := ports.Normalizer.Normalize(ctx, en.Surface, en.Language, en.Script)
	if err != nil {
		t.Fatal(err)
	}
	if ver2.AdapterVersion != ver.AdapterVersion || ver2.AdapterID != ver.AdapterID {
		t.Fatalf("unstable version pin: %#v vs %#v", ver, ver2)
	}
}
