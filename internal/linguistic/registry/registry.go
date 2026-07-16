// Package registry selects in-repo language adapters by BCP 47 tag without
// importing language packages at call sites. External context-lang-* adapters
// plug in through the same linguistic ports (ADR-0015 / ADR-0037).
package registry

import (
	"strings"

	"github.com/fastygo/context/internal/linguistic"
	"github.com/fastygo/context/internal/linguistic/en"
	"github.com/fastygo/context/internal/linguistic/ru"
)

// Ports bundles the linguistic surface for one language.
type Ports struct {
	Language   linguistic.LanguageCode
	Normalizer linguistic.LexicalNormalizer
	Analyzer   linguistic.MorphAnalyzer
	Expander   linguistic.QueryExpander
	Version    linguistic.AnalyzerVersion
}

// ForLanguage returns adapter ports for a language tag ("ru", "en", "ru-RU").
// Unknown languages return ok=false; callers degrade to no expansion.
func ForLanguage(lang string) (Ports, bool) {
	tag := normalizeTag(lang)
	switch tag {
	case "ru":
		return Ports{
			Language:   "ru",
			Normalizer: ru.Normalizer{},
			Analyzer:   ru.Analyzer{},
			Expander:   ru.Expander{},
			Version:    ru.Version(),
		}, true
	case "en":
		return Ports{
			Language:   "en",
			Normalizer: en.Normalizer{},
			Analyzer:   en.Analyzer{},
			Expander:   en.NewExpander(),
			Version: linguistic.AnalyzerVersion{
				AdapterID:      en.AdapterID,
				AdapterVersion: en.AdapterVersion,
				FeatureScheme:  linguistic.FeatureSchemeUD,
			},
		}, true
	default:
		return Ports{}, false
	}
}

// Supported lists language tags with in-repo adapters (stable order).
func Supported() []string { return []string{"en", "ru"} }

func normalizeTag(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if i := strings.IndexAny(lang, "-_"); i > 0 {
		lang = lang[:i]
	}
	return lang
}
