package token_test

import (
	"testing"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/indexing/chunk"
	"github.com/fastygo/context/internal/indexing/token"
)

func TestCapturePreservesAbsoluteSpans(t *testing.T) {
	t.Parallel()
	rc := chunk.RawChunk{
		Span: foundation.ByteSpan{Start: 10, End: 21},
		Text: "hello world",
	}
	toks := token.Capture("p1", "s1", "c1", "en", rc)
	if len(toks) != 2 {
		t.Fatalf("tokens=%d", len(toks))
	}
	if toks[0].Surface != "hello" || toks[0].Span.Start != 10 || toks[0].Span.End != 15 {
		t.Fatalf("tok0=%#v", toks[0])
	}
	if toks[1].Surface != "world" || toks[1].Span.Start != 16 || toks[1].Span.End != 21 {
		t.Fatalf("tok1=%#v", toks[1])
	}
}
