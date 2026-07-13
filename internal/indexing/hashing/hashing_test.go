package hashing_test

import (
	"testing"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/indexing/hashing"
)

func TestPathKeyStableAndIndependentOfHostPath(t *testing.T) {
	t.Parallel()
	a := hashing.PathKey("proj", "docs/README.md")
	b := hashing.PathKey("proj", `docs\README.md`)
	c := hashing.PathKey("proj", "./docs/README.md")
	if a == "" || a != b || a != c {
		t.Fatalf("path keys differ: %s %s %s", a, b, c)
	}
	other := hashing.PathKey("other", "docs/README.md")
	if a == other {
		t.Fatal("path key must include project_id")
	}
}

func TestChunkHashStable(t *testing.T) {
	t.Parallel()
	h1 := hashing.ChunkHash("paragraph-v1", "pk", 0, 5, "hello")
	h2 := hashing.ChunkHash("paragraph-v1", "pk", 0, 5, "hello")
	h3 := hashing.ChunkHash("paragraph-v1", "pk", 0, 4, "hell")
	if h1 != h2 {
		t.Fatal("expected stable hash")
	}
	if h1 == h3 {
		t.Fatal("expected text/span change to alter hash")
	}
}

func TestMerkleRootsDeterministic(t *testing.T) {
	t.Parallel()
	leaves1 := map[string]foundation.ChecksumHex{
		"b": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"a": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	leaves2 := map[string]foundation.ChecksumHex{
		"a": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"b": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
	if hashing.SourceMerkleRoot(leaves1) != hashing.SourceMerkleRoot(leaves2) {
		t.Fatal("source merkle root must be insertion-order independent")
	}
	chunksA := []foundation.ChecksumHex{
		"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	chunksB := []foundation.ChecksumHex{chunksA[1], chunksA[0]}
	if hashing.ChunkSetHash(chunksA) != hashing.ChunkSetHash(chunksB) {
		t.Fatal("chunk set hash must be order independent")
	}
}

func TestSourceLeafIncludesArtifactChecksum(t *testing.T) {
	t.Parallel()
	h1 := hashing.SourceLeafHash("pk", "file", []byte("abc"))
	h2 := hashing.SourceLeafHash("pk", "file", []byte("abd"))
	if h1 == h2 {
		t.Fatal("leaf hash must change when artifact bytes change")
	}
}
