package registry_test

import (
	"context"
	"testing"

	"github.com/fastygo/context/internal/linguistic/registry"
)

func TestForLanguageRU(t *testing.T) {
	t.Parallel()
	ports, ok := registry.ForLanguage("ru")
	if !ok {
		t.Fatal("ru adapter must be registered")
	}
	exps, err := ports.Expander.Expand(context.Background(), "q", "дорога", "ru")
	if err != nil || len(exps) == 0 {
		t.Fatalf("ru expansion: %v %d", err, len(exps))
	}
	if ports.Version.AdapterID != "context-lang-ru" {
		t.Fatalf("adapter id %q", ports.Version.AdapterID)
	}
}

func TestForLanguageRegionTagAndCase(t *testing.T) {
	t.Parallel()
	if _, ok := registry.ForLanguage("RU-ru"); !ok {
		t.Fatal("region subtag must resolve to base language")
	}
	if _, ok := registry.ForLanguage("en_US"); !ok {
		t.Fatal("underscore variant must resolve")
	}
}

func TestForLanguageUnknown(t *testing.T) {
	t.Parallel()
	if _, ok := registry.ForLanguage("xx"); ok {
		t.Fatal("unknown language must not resolve")
	}
	if _, ok := registry.ForLanguage(""); ok {
		t.Fatal("empty language must not resolve")
	}
}
