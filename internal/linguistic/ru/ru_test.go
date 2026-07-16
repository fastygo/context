package ru_test

import (
	"context"
	"testing"

	"github.com/fastygo/context/internal/linguistic"
	"github.com/fastygo/context/internal/linguistic/harness"
	"github.com/fastygo/context/internal/linguistic/ru"
)

func TestInternalContract(t *testing.T) {
	harness.RunContract(t, harness.Ports{
		Normalizer: ru.Normalizer{},
		Analyzer:   ru.Analyzer{},
		Expander:   ru.Expander{},
	})
}

func TestRussianExpansionThroughInternalContract(t *testing.T) {
	t.Parallel()
	exps, err := (ru.Expander{}).Expand(context.Background(), "q_ru", "дорога", "ru")
	if err != nil {
		t.Fatal(err)
	}
	if len(exps) == 0 {
		t.Fatal("expected expansions")
	}
	seen := map[string]bool{}
	for _, e := range exps {
		if err := e.Validate(); err != nil {
			t.Fatalf("invalid expansion: %v", err)
		}
		seen[e.ExpandedTerm] = true
	}
	if !seen["дороги"] || !seen["дорогу"] {
		t.Fatalf("missing paradigm forms, got %v", seen)
	}
}

func TestRussianAnalyzerAmbiguityInternal(t *testing.T) {
	t.Parallel()
	token := linguistic.TokenOccurrence{
		ID: "tok_ru", ProjectID: "p", SourceID: "s", ChunkID: "c",
		Language: "ru", Script: "Cyrl", Surface: "дорогой", Normalized: "дорогой",
	}
	token.Span.End = 14
	analyses, err := (ru.Analyzer{}).Analyze(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	lemmaSet := map[linguistic.Lemma]bool{}
	for _, a := range analyses {
		lemmaSet[a.Lemma] = true
	}
	if !lemmaSet["дорога"] || !lemmaSet["дорогой"] {
		t.Fatalf("expected ambiguity дорога/дорогой, got %v", lemmaSet)
	}
}
