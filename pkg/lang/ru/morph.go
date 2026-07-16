package ru

import (
	"sort"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// PartOfSpeech is the coarse POS tag used by the rule engine.
type PartOfSpeech string

const (
	POSNoun      PartOfSpeech = "noun"
	POSAdjective PartOfSpeech = "adj"
	POSVerb      PartOfSpeech = "verb"
	POSUnknown   PartOfSpeech = "unknown"
)

// Candidate is one possible analysis of a surface wordform. Ambiguity is
// explicit: callers receive every plausible candidate, ordered by confidence.
type Candidate struct {
	Lemma      string
	POS        PartOfSpeech
	Paradigm   string
	Stem       string
	Reflexive  bool
	Confidence float64
	Reason     string
}

// ExpansionKind classifies generated forms (mirrors langcontract types).
type ExpansionKind string

const (
	KindLemma    ExpansionKind = "lemma"
	KindWordform ExpansionKind = "wordform"
	KindAccent   ExpansionKind = "accent"
)

// GeneratedForm is one explainable expansion produced from a query term.
type GeneratedForm struct {
	Form       string
	Kind       ExpansionKind
	Reason     string
	Confidence float64
}

// Fold normalizes a word for matching: NFC, lowercase, ё→е.
// The fold is one-way; original surfaces are never rewritten.
func Fold(s string) string {
	s = norm.NFC.String(s)
	s = strings.ToLower(s)
	return strings.ReplaceAll(s, "ё", "е")
}

type formSpec struct {
	suffix string
	tag    string
}

type paradigm struct {
	id          string
	pos         PartOfSpeech
	lemmaSuffix string
	prior       float64
	minStem     int // minimum stem length in runes
	forms       []formSpec
}

// Ending tables are written in folded form (ё collapsed to е).
var paradigms = []paradigm{
	{
		id: "noun-a", pos: POSNoun, lemmaSuffix: "а", prior: 0.55, minStem: 3,
		forms: []formSpec{
			{"а", "sg-nom"}, {"ы", "sg-gen"}, {"и", "sg-gen-vel"}, {"е", "sg-dat-loc"},
			{"у", "sg-acc"}, {"ой", "sg-ins"}, {"ою", "sg-ins-var"},
			{"", "pl-gen"}, {"ам", "pl-dat"}, {"ами", "pl-ins"}, {"ах", "pl-loc"},
		},
	},
	{
		id: "noun-ya", pos: POSNoun, lemmaSuffix: "я", prior: 0.5, minStem: 3,
		forms: []formSpec{
			{"я", "sg-nom"}, {"и", "sg-gen"}, {"е", "sg-dat-loc"}, {"ю", "sg-acc"},
			{"ей", "sg-ins"}, {"ею", "sg-ins-var"},
			{"ям", "pl-dat"}, {"ями", "pl-ins"}, {"ях", "pl-loc"},
		},
	},
	{
		id: "noun-m", pos: POSNoun, lemmaSuffix: "", prior: 0.5, minStem: 3,
		forms: []formSpec{
			{"", "sg-nom"}, {"а", "sg-gen"}, {"у", "sg-dat"}, {"ом", "sg-ins"}, {"е", "sg-loc"},
			{"ы", "pl-nom"}, {"и", "pl-nom-vel"}, {"ов", "pl-gen"}, {"ей", "pl-gen-soft"},
			{"ам", "pl-dat"}, {"ами", "pl-ins"}, {"ах", "pl-loc"},
		},
	},
	{
		id: "noun-msoft", pos: POSNoun, lemmaSuffix: "ь", prior: 0.45, minStem: 3,
		forms: []formSpec{
			{"ь", "sg-nom"}, {"я", "sg-gen"}, {"ю", "sg-dat"}, {"ем", "sg-ins"}, {"е", "sg-loc"},
			{"и", "pl-nom"}, {"ей", "pl-gen"}, {"ям", "pl-dat"}, {"ями", "pl-ins"}, {"ях", "pl-loc"},
		},
	},
	{
		id: "noun-j", pos: POSNoun, lemmaSuffix: "й", prior: 0.45, minStem: 3,
		forms: []formSpec{
			{"й", "sg-nom"}, {"я", "sg-gen"}, {"ю", "sg-dat"}, {"ем", "sg-ins"}, {"е", "sg-loc"},
			{"и", "pl-nom"}, {"ев", "pl-gen"}, {"ям", "pl-dat"}, {"ями", "pl-ins"}, {"ях", "pl-loc"},
		},
	},
	{
		id: "noun-o", pos: POSNoun, lemmaSuffix: "о", prior: 0.5, minStem: 3,
		forms: []formSpec{
			{"о", "sg-nom"}, {"а", "sg-gen"}, {"у", "sg-dat"}, {"ом", "sg-ins"}, {"е", "sg-loc"},
			{"ам", "pl-dat"}, {"ами", "pl-ins"}, {"ах", "pl-loc"},
		},
	},
	{
		id: "noun-e", pos: POSNoun, lemmaSuffix: "е", prior: 0.4, minStem: 3,
		forms: []formSpec{
			{"е", "sg-nom"}, {"я", "sg-gen"}, {"ю", "sg-dat"}, {"ем", "sg-ins"},
			{"и", "sg-loc-pl"}, {"ям", "pl-dat"}, {"ями", "pl-ins"}, {"ях", "pl-loc"},
		},
	},
	{
		id: "noun-f3", pos: POSNoun, lemmaSuffix: "ь", prior: 0.4, minStem: 3,
		forms: []formSpec{
			{"ь", "sg-nom"}, {"и", "sg-gen-dat-loc"}, {"ью", "sg-ins"},
			{"ей", "pl-gen"}, {"ям", "pl-dat"}, {"ями", "pl-ins"}, {"ях", "pl-loc"},
		},
	},
	{
		id: "adj-hard", pos: POSAdjective, lemmaSuffix: "ый", prior: 0.55, minStem: 3,
		forms: []formSpec{
			{"ый", "m-nom"}, {"ого", "m-gen"}, {"ому", "m-dat"}, {"ым", "m-ins"}, {"ом", "m-loc"},
			{"ая", "f-nom"}, {"ой", "f-obl"}, {"ую", "f-acc"},
			{"ое", "n-nom"}, {"ые", "pl-nom"}, {"ых", "pl-gen"}, {"ыми", "pl-ins"},
		},
	},
	{
		id: "adj-soft", pos: POSAdjective, lemmaSuffix: "ий", prior: 0.5, minStem: 3,
		forms: []formSpec{
			{"ий", "m-nom"}, {"его", "m-gen"}, {"ему", "m-dat"}, {"им", "m-ins"}, {"ем", "m-loc"},
			{"яя", "f-nom"}, {"ей", "f-obl"}, {"юю", "f-acc"},
			{"ее", "n-nom"}, {"ие", "pl-nom"}, {"их", "pl-gen"}, {"ими", "pl-ins"},
		},
	},
	{
		id: "adj-vel", pos: POSAdjective, lemmaSuffix: "ий", prior: 0.5, minStem: 3,
		forms: []formSpec{
			{"ий", "m-nom"}, {"ого", "m-gen"}, {"ому", "m-dat"}, {"им", "m-ins"}, {"ом", "m-loc"},
			{"ая", "f-nom"}, {"ой", "f-obl"}, {"ую", "f-acc"},
			{"ое", "n-nom"}, {"ие", "pl-nom"}, {"их", "pl-gen"}, {"ими", "pl-ins"},
		},
	},
	{
		id: "adj-oj", pos: POSAdjective, lemmaSuffix: "ой", prior: 0.45, minStem: 3,
		forms: []formSpec{
			{"ой", "m-nom"}, {"ого", "m-gen"}, {"ому", "m-dat"}, {"им", "m-ins"}, {"ом", "m-loc"},
			{"ая", "f-nom"}, {"ую", "f-acc"},
			{"ое", "n-nom"}, {"ие", "pl-nom"}, {"их", "pl-gen"}, {"ими", "pl-ins"},
		},
	},
	{
		id: "verb-at", pos: POSVerb, lemmaSuffix: "ать", prior: 0.5, minStem: 2,
		forms: []formSpec{
			{"ать", "inf"}, {"аю", "pres-1sg"}, {"аешь", "pres-2sg"}, {"ает", "pres-3sg"},
			{"аем", "pres-1pl"}, {"аете", "pres-2pl"}, {"ают", "pres-3pl"},
			{"ал", "past-m"}, {"ала", "past-f"}, {"ало", "past-n"}, {"али", "past-pl"},
			{"ай", "imp-sg"}, {"айте", "imp-pl"},
		},
	},
	{
		id: "verb-yat", pos: POSVerb, lemmaSuffix: "ять", prior: 0.45, minStem: 2,
		forms: []formSpec{
			{"ять", "inf"}, {"яю", "pres-1sg"}, {"яешь", "pres-2sg"}, {"яет", "pres-3sg"},
			{"яем", "pres-1pl"}, {"яете", "pres-2pl"}, {"яют", "pres-3pl"},
			{"ял", "past-m"}, {"яла", "past-f"}, {"яло", "past-n"}, {"яли", "past-pl"},
			{"яй", "imp-sg"}, {"яйте", "imp-pl"},
		},
	},
	{
		id: "verb-et", pos: POSVerb, lemmaSuffix: "еть", prior: 0.45, minStem: 2,
		forms: []formSpec{
			{"еть", "inf"}, {"ею", "pres-1sg"}, {"еешь", "pres-2sg"}, {"еет", "pres-3sg"},
			{"еем", "pres-1pl"}, {"еете", "pres-2pl"}, {"еют", "pres-3pl"},
			{"ел", "past-m"}, {"ела", "past-f"}, {"ело", "past-n"}, {"ели", "past-pl"},
		},
	},
	{
		id: "verb-it", pos: POSVerb, lemmaSuffix: "ить", prior: 0.5, minStem: 2,
		forms: []formSpec{
			{"ить", "inf"}, {"ю", "pres-1sg"}, {"ишь", "pres-2sg"}, {"ит", "pres-3sg"},
			{"им", "pres-1pl"}, {"ите", "pres-2pl"}, {"ят", "pres-3pl"},
			{"ил", "past-m"}, {"ила", "past-f"}, {"ило", "past-n"}, {"или", "past-pl"},
			{"и", "imp-sg"},
		},
	},
	{
		id: "verb-ovat", pos: POSVerb, lemmaSuffix: "овать", prior: 0.5, minStem: 2,
		forms: []formSpec{
			{"овать", "inf"}, {"ую", "pres-1sg"}, {"уешь", "pres-2sg"}, {"ует", "pres-3sg"},
			{"уем", "pres-1pl"}, {"уете", "pres-2pl"}, {"уют", "pres-3pl"},
			{"овал", "past-m"}, {"овала", "past-f"}, {"овало", "past-n"}, {"овали", "past-pl"},
			{"уй", "imp-sg"}, {"уйте", "imp-pl"},
		},
	},
	{
		id: "verb-evat", pos: POSVerb, lemmaSuffix: "евать", prior: 0.45, minStem: 2,
		forms: []formSpec{
			{"евать", "inf"}, {"ую", "pres-1sg"}, {"уешь", "pres-2sg"}, {"ует", "pres-3sg"},
			{"уем", "pres-1pl"}, {"уете", "pres-2pl"}, {"уют", "pres-3pl"},
			{"евал", "past-m"}, {"евала", "past-f"}, {"евало", "past-n"}, {"евали", "past-pl"},
			{"уй", "imp-sg"}, {"уйте", "imp-pl"},
		},
	},
}

// exceptionParadigms is a tiny curated fixture (folded forms). It exists to
// prove the irregular path; real coverage belongs to dictionary adapters.
var exceptionParadigms = map[string]struct {
	pos   PartOfSpeech
	forms []string
}{
	"бежать": {POSVerb, []string{
		"бегу", "бежишь", "бежит", "бежим", "бежите", "бегут",
		"бежал", "бежала", "бежало", "бежали", "беги", "бегите",
	}},
	"идти": {POSVerb, []string{
		"иду", "идешь", "идет", "идем", "идете", "идут",
		"шел", "шла", "шло", "шли", "иди", "идите",
	}},
	"человек": {POSNoun, []string{
		"люди", "человека", "человеку", "человеком", "человеке",
		"людей", "людям", "людьми", "людях",
	}},
	"ребенок": {POSNoun, []string{
		"дети", "ребенка", "ребенку", "ребенком", "ребенке",
		"детей", "детям", "детьми", "детях",
	}},
	"год": {POSNoun, []string{
		"года", "году", "годом", "годе", "годы", "лет", "годам", "годами", "годах",
	}},
}

var exceptionFormToLemma map[string]string

func init() {
	exceptionFormToLemma = make(map[string]string)
	for lemma, e := range exceptionParadigms {
		for _, f := range e.forms {
			exceptionFormToLemma[f] = lemma
		}
	}
}

const (
	// DefaultMaxForms bounds ExpandWord output.
	DefaultMaxForms = 32
	// expandConfidenceFloor filters analysis candidates used for generation.
	expandConfidenceFloor = 0.4
	// expandMaxCandidates bounds how many distinct analyses generate forms.
	expandMaxCandidates = 2
)

var velarHusher = map[rune]bool{
	'г': true, 'к': true, 'х': true, 'ж': true, 'ш': true, 'ч': true, 'щ': true,
}

// AnalyzeWord returns candidate analyses for one Russian surface word.
// The input may be any case and may contain ё; matching is done on the fold.
func AnalyzeWord(word string) []Candidate {
	w := Fold(strings.TrimSpace(word))
	if w == "" || !isCyrillic(w) {
		return nil
	}
	base := w
	reflexive := false
	if strings.HasSuffix(base, "ся") && runeLen(base) > 4 {
		base = strings.TrimSuffix(base, "ся")
		reflexive = true
	} else if strings.HasSuffix(base, "сь") && runeLen(base) > 4 {
		base = strings.TrimSuffix(base, "сь")
		reflexive = true
	}

	seen := map[string]bool{}
	var out []Candidate

	if lemma, ok := exceptionFormToLemma[w]; ok {
		e := exceptionParadigms[lemma]
		out = append(out, Candidate{
			Lemma: lemma, POS: e.pos, Paradigm: "exception", Stem: lemma,
			Confidence: 0.95, Reason: "exception fixture",
		})
		seen[lemma+"|exception"] = true
	}
	if _, ok := exceptionParadigms[w]; ok {
		e := exceptionParadigms[w]
		key := w + "|exception-lemma"
		if !seen[key] {
			out = append(out, Candidate{
				Lemma: w, POS: e.pos, Paradigm: "exception", Stem: w,
				Confidence: 0.95, Reason: "exception lemma",
			})
			seen[key] = true
		}
	}

	for _, p := range paradigms {
		if reflexive && p.pos != POSVerb {
			continue
		}
		for _, f := range p.forms {
			stem, ok := stripSuffix(base, f.suffix, p.minStem)
			if !ok {
				continue
			}
			lemma := stem + p.lemmaSuffix
			if reflexive {
				lemma = appendReflexive(lemma)
			}
			key := lemma + "|" + p.id
			if seen[key] {
				continue
			}
			seen[key] = true
			conf := p.prior + 0.04*float64(runeLen(f.suffix))
			if f.suffix == "" {
				if p.lemmaSuffix == "" {
					// Citation form itself (e.g. masc noun nominative).
					conf = p.prior + 0.1
				} else {
					// Oblique zero-ending guess (e.g. plural genitive) is weak.
					conf = p.prior - 0.2
				}
			}
			if conf > 0.9 {
				conf = 0.9
			}
			out = append(out, Candidate{
				Lemma: lemma, POS: p.pos, Paradigm: p.id, Stem: stem,
				Reflexive: reflexive,
				Confidence: conf,
				Reason:     "ru " + p.id + " " + f.tag,
			})
		}
	}

	out = append(out, Candidate{
		Lemma: w, POS: POSUnknown, Paradigm: "surface", Stem: w,
		Confidence: 0.3, Reason: "surface fallback",
	})

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Confidence != out[j].Confidence {
			return out[i].Confidence > out[j].Confidence
		}
		if out[i].Lemma != out[j].Lemma {
			return out[i].Lemma < out[j].Lemma
		}
		return out[i].Paradigm < out[j].Paradigm
	})
	return out
}

// LemmaSet returns the folded lemma candidates for a word (deduplicated,
// confidence order, bounded). It always includes the folded word itself.
func LemmaSet(word string) []string {
	w := Fold(strings.TrimSpace(word))
	if w == "" {
		return nil
	}
	out := []string{w}
	seen := map[string]bool{w: true}
	for _, c := range AnalyzeWord(word) {
		if c.Confidence < expandConfidenceFloor {
			continue
		}
		if !seen[c.Lemma] {
			seen[c.Lemma] = true
			out = append(out, c.Lemma)
		}
		if len(out) >= 4 {
			break
		}
	}
	return out
}

// ExpandWord generates explainable expansions for a query term: paradigm
// wordforms of the strongest analyses, the lemma when it differs, and an
// accent (ё→е) variant when the original spelling contains ё.
func ExpandWord(word string, max int) []GeneratedForm {
	if max <= 0 {
		max = DefaultMaxForms
	}
	original := strings.TrimSpace(word)
	w := Fold(original)
	if w == "" || !isCyrillic(w) {
		return nil
	}

	var out []GeneratedForm
	seen := map[string]bool{}
	add := func(form string, kind ExpansionKind, reason string, conf float64) {
		if form == "" || seen[form] || len(out) >= max {
			return
		}
		seen[form] = true
		out = append(out, GeneratedForm{Form: form, Kind: kind, Reason: reason, Confidence: conf})
	}

	// The folded spelling itself is a useful expansion when the original
	// contains ё (query "ёлка" must reach texts spelled "елка").
	if strings.Contains(strings.ToLower(norm.NFC.String(original)), "ё") {
		add(w, KindAccent, "ru yo-fold variant", 0.85)
	}
	seen[w] = true

	if _, ok := exceptionParadigms[w]; ok {
		for _, f := range exceptionParadigms[w].forms {
			add(f, KindWordform, "ru exception paradigm", 0.9)
		}
		return out
	}
	if lemma, ok := exceptionFormToLemma[w]; ok {
		add(lemma, KindLemma, "ru exception lemma", 0.9)
		for _, f := range exceptionParadigms[lemma].forms {
			add(f, KindWordform, "ru exception paradigm", 0.85)
		}
		return out
	}

	candidates := AnalyzeWord(original)
	used := 0
	usedLemmas := map[string]bool{}
	for _, c := range candidates {
		if used >= expandMaxCandidates {
			break
		}
		if c.Confidence < expandConfidenceFloor || c.Paradigm == "surface" {
			continue
		}
		if usedLemmas[c.Lemma] {
			continue
		}
		usedLemmas[c.Lemma] = true
		used++
		if c.Lemma != w {
			add(c.Lemma, KindLemma, "ru "+c.Paradigm+" lemma", c.Confidence)
		}
		p, ok := paradigmByID(c.Paradigm)
		if !ok {
			continue
		}
		for _, f := range p.forms {
			form := applySpelling(c.Stem, f.suffix)
			if c.Reflexive {
				form = appendReflexive(form)
			}
			add(form, KindWordform, "ru "+p.id+" "+f.tag, c.Confidence-0.05)
		}
	}
	return out
}

func paradigmByID(id string) (paradigm, bool) {
	for _, p := range paradigms {
		if p.id == id {
			return p, true
		}
	}
	return paradigm{}, false
}

// applySpelling applies the ы→и / я→а / ю→у spelling rule after velars and hushers.
func applySpelling(stem, suffix string) string {
	if suffix == "" {
		return stem
	}
	runes := []rune(stem)
	if len(runes) == 0 {
		return stem + suffix
	}
	last := runes[len(runes)-1]
	if !velarHusher[last] {
		return stem + suffix
	}
	sr := []rune(suffix)
	switch sr[0] {
	case 'ы':
		sr[0] = 'и'
	case 'я':
		sr[0] = 'а'
	case 'ю':
		sr[0] = 'у'
	}
	return stem + string(sr)
}

func appendReflexive(form string) string {
	if form == "" {
		return form
	}
	runes := []rune(form)
	if isVowel(runes[len(runes)-1]) {
		return form + "сь"
	}
	return form + "ся"
}

func isVowel(r rune) bool {
	switch r {
	case 'а', 'е', 'и', 'о', 'у', 'ы', 'э', 'ю', 'я':
		return true
	}
	return false
}

func stripSuffix(word, suffix string, minStem int) (string, bool) {
	if suffix == "" {
		// Zero ending: the whole word acts as stem; require consonant final.
		runes := []rune(word)
		if len(runes) < minStem+1 {
			return "", false
		}
		if isVowel(runes[len(runes)-1]) || runes[len(runes)-1] == 'ь' || runes[len(runes)-1] == 'й' {
			return "", false
		}
		return word, true
	}
	if !strings.HasSuffix(word, suffix) {
		return "", false
	}
	stem := strings.TrimSuffix(word, suffix)
	if runeLen(stem) < minStem {
		return "", false
	}
	return stem, true
}

func runeLen(s string) int { return len([]rune(s)) }

func isCyrillic(s string) bool {
	for _, r := range s {
		if r >= 0x0400 && r <= 0x04FF {
			continue
		}
		if r == '-' {
			continue
		}
		return false
	}
	return s != ""
}
