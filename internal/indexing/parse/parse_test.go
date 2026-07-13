package parse_test

import (
	"context"
	"testing"

	"github.com/fastygo/context/internal/indexing/parse"
)

func TestMarkdownPreservesHeadingAncestry(t *testing.T) {
	t.Parallel()
	doc, err := (parse.Markdown{}).Parse(context.Background(), "text/markdown", []byte("# A\n\npara\n\n## B\n\nmore\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Sections) < 2 {
		t.Fatalf("sections=%v", doc.Sections)
	}
	found := false
	for _, s := range doc.Sections {
		if s.Title == "B" {
			found = true
			if len(s.Ancestry) < 2 || s.Ancestry[0] != "A" || s.Ancestry[1] != "B" {
				t.Fatalf("ancestry=%v", s.Ancestry)
			}
		}
	}
	if !found {
		t.Fatal("missing section B")
	}
}

func TestPlaintextParagraphBoundaries(t *testing.T) {
	t.Parallel()
	doc, err := (parse.PlainText{}).Parse(context.Background(), "text/plain", []byte("one\n\ntwo"))
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, b := range doc.Boundaries {
		if b.Kind == parse.BoundaryParagraph {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("paragraphs=%d bounds=%v", n, doc.Boundaries)
	}
}
