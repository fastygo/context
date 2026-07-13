package memory_test

import (
	"context"
	"testing"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/storage/memory"
)

// ADR-0025: chunks under project_a must be invisible to project_b.
func TestNoCrossProjectChunkLeakage(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	_ = store.PutProject(ctx, corpus.Project{ID: "pa", Name: "A", TenantID: "ten_1"})
	_ = store.PutProject(ctx, corpus.Project{ID: "pb", Name: "B", TenantID: "ten_2"})
	ch := corpus.Chunk{
		ID: "c_secret", ProjectID: "pa", SourceID: "s1", ArtifactID: "a1",
		SnapshotID: "snap_a", ChunkerVersion: "para-v1",
		Span: foundation.ByteSpan{Start: 0, End: 4}, TextChecksum: "aa", ChunkHash: "bb",
	}
	if err := store.PutChunk(ctx, ch); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetChunk(ctx, "pb", "c_secret"); !apperr.Is(err, apperr.NotFound) {
		t.Fatalf("project_b must not see project_a chunk: %v", err)
	}
	list, err := store.ListChunks(ctx, "pb", "snap_a")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("leak: %#v", list)
	}
	got, err := store.GetChunk(ctx, "pa", "c_secret")
	if err != nil || got.ID != "c_secret" {
		t.Fatalf("owner read: %#v %v", got, err)
	}
}
