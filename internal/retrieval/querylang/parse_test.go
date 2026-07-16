package querylang_test

import (
	"strings"
	"testing"

	"github.com/fastygo/context/internal/retrieval/querylang"
)

func mustParse(t *testing.T, q string) querylang.Query {
	t.Helper()
	parsed, err := querylang.Parse(q)
	if err != nil {
		t.Fatalf("Parse(%q): %v", q, err)
	}
	return parsed
}

func TestParseImplicitAnd(t *testing.T) {
	t.Parallel()
	q := mustParse(t, "контекст память")
	if got := q.Root.Canonical(); got != "(AND контекст память)" {
		t.Fatalf("canonical=%q", got)
	}
}

func TestParseExplicitBoolean(t *testing.T) {
	t.Parallel()
	q := mustParse(t, "(индекс OR поиск) AND память NOT чат")
	want := "(AND (OR индекс поиск) память (NOT чат))"
	if got := q.Root.Canonical(); got != want {
		t.Fatalf("canonical=%q want=%q", got, want)
	}
}

func TestParseMinusNegation(t *testing.T) {
	t.Parallel()
	q := mustParse(t, "память -чат")
	if got := q.Root.Canonical(); got != "(AND память (NOT чат))" {
		t.Fatalf("canonical=%q", got)
	}
}

func TestParsePhraseAndMorphPhrase(t *testing.T) {
	t.Parallel()
	q := mustParse(t, `"железная дорога"`)
	if q.Root.Kind != querylang.KindPhrase || q.Root.Morph {
		t.Fatalf("want exact phrase, got %#v", q.Root)
	}
	q2 := mustParse(t, `~"железная дорога"`)
	if q2.Root.Kind != querylang.KindPhrase || !q2.Root.Morph {
		t.Fatalf("want morph phrase, got %#v", q2.Root)
	}
	if q2.Root.Text != "железная дорога" {
		t.Fatalf("phrase text=%q", q2.Root.Text)
	}
}

func TestParseMorphTerm(t *testing.T) {
	t.Parallel()
	q := mustParse(t, "~дорога")
	if q.Root.Kind != querylang.KindTerm || !q.Root.Morph {
		t.Fatalf("want morph term, got %#v", q.Root)
	}
}

func TestParseLangDirective(t *testing.T) {
	t.Parallel()
	q := mustParse(t, "lang:ru дорога")
	if q.Language != "ru" {
		t.Fatalf("language=%q", q.Language)
	}
	if q.Root.Canonical() != "дорога" {
		t.Fatalf("canonical=%q", q.Root.Canonical())
	}
	q2 := mustParse(t, "lang:RU-ru дорога")
	if q2.Language != "ru" {
		t.Fatalf("region subtag not folded: %q", q2.Language)
	}
}

func TestParseCaseInsensitiveOperators(t *testing.T) {
	t.Parallel()
	q := mustParse(t, "память or чат")
	if q.Root.Kind != querylang.KindOr {
		t.Fatalf("lowercase or must parse as operator, got %#v", q.Root)
	}
}

func TestParseErrors(t *testing.T) {
	t.Parallel()
	cases := []string{
		"",
		"   ",
		`"незакрытая фраза`,
		"NOT чат",
		"-чат",
		"память OR -чат",
		"(память",
		"~",
		"lang:ru",
		"NOT NOT память",
		`""`,
	}
	for _, c := range cases {
		if _, err := querylang.Parse(c); err == nil {
			t.Errorf("Parse(%q): expected error", c)
		}
	}
}

func TestParseDeterministicCanonical(t *testing.T) {
	t.Parallel()
	a := mustParse(t, `лексема AND ("словарь" OR корпус) -шум`).Root.Canonical()
	b := mustParse(t, `лексема AND ("словарь" OR корпус) -шум`).Root.Canonical()
	if a != b || !strings.Contains(a, "(OR") {
		t.Fatalf("canonical unstable or wrong: %q vs %q", a, b)
	}
}
