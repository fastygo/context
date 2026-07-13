package sparse_test

import (
	"context"
	"testing"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/fake"
	"github.com/fastygo/context/internal/retrieval/index"
	"github.com/fastygo/context/internal/retrieval/sparse"
)

func TestRetrieverMapsHitsThroughIndex(t *testing.T) {
	t.Parallel()
	idx := index.NewMemory()
	idx.Add(index.ChunkRecord{
		ProjectID: "p1", SnapshotID: "s1", ChunkID: "c1", SourceID: "src",
		Span: foundation.ByteSpan{Start: 0, End: 5}, Text: "ContextPack hybrid retrieval",
		TextChecksum: "aa", TrustLevel: foundation.TrustProject,
	})
	idx.Add(index.ChunkRecord{
		ProjectID: "p1", SnapshotID: "s1", ChunkID: "c2", SourceID: "src",
		Span: foundation.ByteSpan{Start: 0, End: 5}, Text: "unrelated noise",
		TextChecksum: "bb", TrustLevel: foundation.TrustProject,
	})

	r := sparse.Retriever{
		Client:      fake.SparseClient{Index: idx},
		Index:       idx,
		Explanation: "test sparse",
	}
	cands, err := r.Retrieve(context.Background(), retrieval.RetrievalPlan{
		ID: "plan", ProjectID: "p1", SnapshotID: "s1", TopNRawPool: 10,
	}, "ContextPack")
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) == 0 {
		t.Fatal("expected candidates")
	}
	if cands[0].ChunkID != "c1" {
		t.Fatalf("got %#v", cands)
	}
	if cands[0].Contributions[0].Explanation != "test sparse" {
		t.Fatalf("explanation=%q", cands[0].Contributions[0].Explanation)
	}
}

func TestRetrieverRequiresClientAndIndex(t *testing.T) {
	t.Parallel()
	_, err := (sparse.Retriever{}).Retrieve(context.Background(), retrieval.RetrievalPlan{
		ID: "p", ProjectID: "p1", SnapshotID: "s1",
	}, "q")
	if err == nil {
		t.Fatal("expected error")
	}
}
