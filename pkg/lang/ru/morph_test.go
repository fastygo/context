package ru_test

import (
	"testing"

	ru "github.com/fastygo/context/pkg/lang/ru"
)

func lemmas(t *testing.T, word string) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, c := range ru.AnalyzeWord(word) {
		out[c.Lemma] = true
	}
	return out
}

func forms(t *testing.T, word string) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, f := range ru.ExpandWord(word, 0) {
		out[f.Form] = true
	}
	return out
}

func TestAnalyzeNounInflection(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"дороги":  "дорога",
		"дорогу":  "дорога",
		"дорогой": "дорога", // ambiguous with adjective; must still be present
		"бегуны":  "бегун",
		"бегунов": "бегун",
		"столом":  "стол",
		"ночью":   "ночь",
		"недели":  "неделя",
	}
	for word, wantLemma := range cases {
		if !lemmas(t, word)[wantLemma] {
			t.Errorf("AnalyzeWord(%q): missing lemma %q; got %v", word, wantLemma, lemmas(t, word))
		}
	}
}

func TestAnalyzeVerbInflection(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"читает":   "читать",
		"читали":   "читать",
		"говорит":  "говорить",
		"говорили": "говорить",
		"рисует":   "рисовать",
	}
	for word, wantLemma := range cases {
		if !lemmas(t, word)[wantLemma] {
			t.Errorf("AnalyzeWord(%q): missing lemma %q; got %v", word, wantLemma, lemmas(t, word))
		}
	}
}

func TestAnalyzeReflexiveVerb(t *testing.T) {
	t.Parallel()
	if !lemmas(t, "учится")["учиться"] {
		t.Errorf("учится: want lemma учиться, got %v", lemmas(t, "учится"))
	}
	if !lemmas(t, "училась")["учиться"] {
		t.Errorf("училась: want lemma учиться, got %v", lemmas(t, "училась"))
	}
}

func TestAnalyzeExceptionFixture(t *testing.T) {
	t.Parallel()
	if !lemmas(t, "люди")["человек"] {
		t.Errorf("люди: want lemma человек, got %v", lemmas(t, "люди"))
	}
	if !lemmas(t, "бегут")["бежать"] {
		t.Errorf("бегут: want lemma бежать, got %v", lemmas(t, "бегут"))
	}
	if !lemmas(t, "шли")["идти"] {
		t.Errorf("шли: want lemma идти, got %v", lemmas(t, "шли"))
	}
}

func TestAnalyzeAmbiguityIsExplicit(t *testing.T) {
	t.Parallel()
	// "дорогой" is instrumental of дорога and masculine nominative of дорогой.
	got := lemmas(t, "дорогой")
	if !got["дорога"] || !got["дорогой"] {
		t.Fatalf("дорогой: expected both дорога and дорогой candidates, got %v", got)
	}
	// The best candidate is marked first; others must remain visible.
	cands := ru.AnalyzeWord("дорогой")
	if len(cands) < 2 {
		t.Fatalf("expected multiple candidates, got %d", len(cands))
	}
	for i := 1; i < len(cands); i++ {
		if cands[i].Confidence > cands[0].Confidence {
			t.Fatalf("candidates not ordered by confidence: %#v", cands)
		}
	}
}

func TestAnalyzeDeterministicOrder(t *testing.T) {
	t.Parallel()
	a := ru.AnalyzeWord("дороги")
	b := ru.AnalyzeWord("дороги")
	if len(a) != len(b) {
		t.Fatalf("nondeterministic candidate count")
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("nondeterministic order at %d: %#v vs %#v", i, a[i], b[i])
		}
	}
}

func TestExpandNounParadigm(t *testing.T) {
	t.Parallel()
	got := forms(t, "дорога")
	for _, want := range []string{"дороги", "дороге", "дорогу", "дорогой", "дорогам", "дорогами", "дорогах"} {
		if !got[want] {
			t.Errorf("ExpandWord(дорога): missing %q; got %v", want, got)
		}
	}
	for f := range got {
		if f == "дорога" {
			t.Errorf("expansion must not repeat the original term")
		}
	}
}

func TestExpandVerbParadigm(t *testing.T) {
	t.Parallel()
	got := forms(t, "читать")
	for _, want := range []string{"читаю", "читает", "читают", "читал", "читала", "читали"} {
		if !got[want] {
			t.Errorf("ExpandWord(читать): missing %q; got %v", want, got)
		}
	}
}

func TestExpandFromInflectedForm(t *testing.T) {
	t.Parallel()
	// Query arrives as an oblique form; expansion must reach sibling forms.
	got := forms(t, "дороги")
	if !got["дорога"] && !got["дороге"] {
		t.Errorf("ExpandWord(дороги): expected sibling forms of дорога, got %v", got)
	}
}

func TestExpandSpellingRuleAfterVelar(t *testing.T) {
	t.Parallel()
	got := forms(t, "книга")
	if !got["книги"] {
		t.Errorf("ExpandWord(книга): want книги (ы→и after г), got %v", got)
	}
	if got["книгы"] {
		t.Errorf("ExpandWord(книга): forbidden spelling книгы generated")
	}
}

func TestExpandYoFold(t *testing.T) {
	t.Parallel()
	got := ru.ExpandWord("ёлка", 0)
	foundAccent := false
	for _, f := range got {
		if f.Form == "елка" && f.Kind == ru.KindAccent {
			foundAccent = true
		}
	}
	if !foundAccent {
		t.Errorf("ExpandWord(ёлка): want accent variant елка, got %#v", got)
	}
}

func TestExpandExceptionParadigm(t *testing.T) {
	t.Parallel()
	got := forms(t, "человек")
	if !got["люди"] || !got["людей"] {
		t.Errorf("ExpandWord(человек): want люди/людей, got %v", got)
	}
	gotVerb := forms(t, "бежать")
	if !gotVerb["бегут"] || !gotVerb["бежит"] {
		t.Errorf("ExpandWord(бежать): want бегут/бежит, got %v", gotVerb)
	}
}

func TestExpandCapAndDedup(t *testing.T) {
	t.Parallel()
	got := ru.ExpandWord("дорога", 5)
	if len(got) > 5 {
		t.Fatalf("cap violated: %d forms", len(got))
	}
	seen := map[string]bool{}
	for _, f := range got {
		if seen[f.Form] {
			t.Fatalf("duplicate form %q", f.Form)
		}
		seen[f.Form] = true
	}
}

func TestExpandNonCyrillicEmpty(t *testing.T) {
	t.Parallel()
	if got := ru.ExpandWord("running", 0); len(got) != 0 {
		t.Fatalf("non-cyrillic input must not expand, got %#v", got)
	}
	if got := ru.AnalyzeWord("hello"); len(got) != 0 {
		t.Fatalf("non-cyrillic input must not analyze, got %#v", got)
	}
}

func TestExpandEveryFormExplainable(t *testing.T) {
	t.Parallel()
	for _, f := range ru.ExpandWord("говорить", 0) {
		if f.Reason == "" || f.Confidence <= 0 || f.Form == "" {
			t.Fatalf("unexplainable expansion: %#v", f)
		}
	}
}

func TestLemmaSetIncludesSelf(t *testing.T) {
	t.Parallel()
	got := ru.LemmaSet("дороги")
	if len(got) == 0 || got[0] != "дороги" {
		t.Fatalf("LemmaSet must start with the folded word, got %v", got)
	}
	found := false
	for _, l := range got {
		if l == "дорога" {
			found = true
		}
	}
	if !found {
		t.Fatalf("LemmaSet(дороги) missing дорога: %v", got)
	}
}

func TestFoldOneWay(t *testing.T) {
	t.Parallel()
	if ru.Fold("Ёлка") != "елка" {
		t.Fatalf("Fold(Ёлка)=%q", ru.Fold("Ёлка"))
	}
	if ru.Fold("ДОРОГА") != "дорога" {
		t.Fatalf("Fold(ДОРОГА)=%q", ru.Fold("ДОРОГА"))
	}
}
