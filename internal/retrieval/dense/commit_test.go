package dense_test

import (
	"context"
	"testing"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/indexing"
	modelfake "github.com/fastygo/context/internal/models/fake"
	"github.com/fastygo/context/internal/retrieval/dense"
	"github.com/fastygo/context/internal/retrieval/fake"
)

func TestUpsertEmbeddedPinsVersions(t *testing.T) {
	t.Parallel()
	store := fake.NewVectorStore()
	emb := modelfake.Embedder{Dim: 8}
	ns := indexing.VectorNamespace{
		Name:             "test_v1",
		ProjectID:        "p1",
		SnapshotID:       "snap1",
		EmbeddingVersion: "fake-hash-v1",
	}
	err := dense.UpsertEmbedded(context.Background(), store, emb, ns, []dense.ChunkDoc{
		{
			ProjectID: "p1", SnapshotID: "snap1", ChunkID: "c1",
			Text: "ContextPack evidence", Language: "en",
			ChunkerVersion: "paragraph-v1", EmbeddingVersion: "fake-hash-v1",
			MorphVersion: "noop-analyzer-v1",
			Span:         foundation.ByteSpan{Start: 0, End: 20},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	hits, err := store.Search(context.Background(), ns, mustEmbed(t, emb, "ContextPack"), 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 || hits[0].ChunkID != "c1" {
		t.Fatalf("hits=%#v", hits)
	}
	if hits[0].EmbeddingVersion != "fake-hash-v1" || hits[0].ChunkerVersion != "paragraph-v1" {
		t.Fatalf("pins=%#v", hits[0])
	}
	if hits[0].MorphVersion != "noop-analyzer-v1" {
		t.Fatalf("morph=%q", hits[0].MorphVersion)
	}
}

func TestUpsertEmbeddedRejectsVersionMismatch(t *testing.T) {
	t.Parallel()
	store := fake.NewVectorStore()
	ns := indexing.VectorNamespace{
		Name: "t", ProjectID: "p1", SnapshotID: "s1", EmbeddingVersion: "v1",
	}
	err := dense.UpsertEmbedded(context.Background(), store, modelfake.Embedder{Dim: 8}, ns, []dense.ChunkDoc{
		{ProjectID: "p1", SnapshotID: "s1", ChunkID: "c1", Text: "x", EmbeddingVersion: "other"},
	})
	if err == nil {
		t.Fatal("expected mismatch error")
	}
}

func mustEmbed(t *testing.T, emb modelfake.Embedder, text string) []float32 {
	t.Helper()
	vecs, _, err := emb.Embed(context.Background(), []string{text})
	if err != nil {
		t.Fatal(err)
	}
	return vecs[0]
}
