package localhash_test

import (
	"context"
	"math"
	"testing"

	"github.com/fastygo/context/internal/models/localhash"
)

func TestEmbedDeterministicAndNormalized(t *testing.T) {
	t.Parallel()
	e := localhash.Embedder{Dim: 16}
	a, ver, err := e.Embed(context.Background(), []string{"ContextPack", "ContextPack", "other"})
	if err != nil {
		t.Fatal(err)
	}
	if ver != localhash.Version {
		t.Fatalf("ver=%q", ver)
	}
	if len(a) != 3 || len(a[0]) != 16 {
		t.Fatalf("shape=%d x %d", len(a), len(a[0]))
	}
	for i := range a[0] {
		if a[0][i] != a[1][i] {
			t.Fatal("same text must match")
		}
	}
	if a[0][0] == a[2][0] && equalVec(a[0], a[2]) {
		t.Fatal("different text should differ")
	}
	var sum float64
	for _, x := range a[0] {
		sum += float64(x) * float64(x)
	}
	if math.Abs(sum-1) > 1e-5 {
		t.Fatalf("L2 norm=%v", sum)
	}
}

func TestDefaultDim(t *testing.T) {
	t.Parallel()
	e := localhash.Embedder{}
	if e.Dimension() != localhash.DefaultDim {
		t.Fatalf("dim=%d", e.Dimension())
	}
}

func equalVec(a, b []float32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
