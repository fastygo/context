package parse_test

import (
	"context"
	"testing"

	"github.com/fastygo/context/internal/indexing/parse"
)

func TestHTMLPreservesOriginalAndConfidence(t *testing.T) {
	t.Parallel()
	raw := []byte(`<!doctype html><html><body><h1>Title</h1><p>Hello <b>runners</b></p><script>x()</script></body></html>`)
	doc, err := (parse.HTML{}).Parse(context.Background(), "text/html", raw)
	if err != nil {
		t.Fatal(err)
	}
	if string(doc.Original) != string(raw) {
		t.Fatal("original bytes must be preserved")
	}
	if doc.ParserVersion == "" || doc.ExtractionConfidence < 0.8 {
		t.Fatalf("confidence/version: %#v", doc)
	}
	if doc.LowConfidence {
		t.Fatal("html strip should not be low-confidence by default")
	}
	if !contains(doc.Text, "Title") || !contains(doc.Text, "Hello") || !contains(doc.Text, "runners") {
		t.Fatalf("text=%q", doc.Text)
	}
	if contains(doc.Text, "script") || contains(doc.Text, "x()") {
		t.Fatalf("script leaked: %q", doc.Text)
	}
}

func TestPDFFlagsLowConfidenceAndKeepsOriginal(t *testing.T) {
	t.Parallel()
	// Minimal fake PDF with a Tj string operator.
	raw := []byte("%PDF-1.4\n1 0 obj<<>>stream\nBT (Hello runners) Tj ET\nendstream\nendobj\n%%EOF\n")
	doc, err := (parse.PDF{}).Parse(context.Background(), "application/pdf", raw)
	if err != nil {
		t.Fatal(err)
	}
	if string(doc.Original) != string(raw) {
		t.Fatal("original bytes must be preserved")
	}
	if !doc.LowConfidence || doc.ExtractionConfidence >= 0.8 {
		t.Fatalf("pdf must flag low confidence: %#v", doc)
	}
	if !contains(doc.Text, "Hello runners") {
		t.Fatalf("text=%q", doc.Text)
	}
}

func TestRegistrySelectsHTMLAndPDF(t *testing.T) {
	t.Parallel()
	reg := parse.Registry{}
	if _, ok := reg.For("text/html").(parse.HTML); !ok {
		t.Fatalf("want HTML, got %T", reg.For("text/html"))
	}
	if _, ok := reg.For("application/pdf").(parse.PDF); !ok {
		t.Fatalf("want PDF, got %T", reg.For("application/pdf"))
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})())
}
