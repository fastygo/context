package dense_test

import (
	"context"
	"testing"

	"github.com/fastygo/context/internal/config"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/indexing"
	modelfake "github.com/fastygo/context/internal/models/fake"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/dense"
	"github.com/fastygo/context/internal/retrieval/fake"
	"github.com/fastygo/context/internal/retrieval/index"
)

func TestDenseRetrieverUsesIndexFilters(t *testing.T) {
	t.Parallel()
	mem := index.NewMemory(
		index.ChunkRecord{
			ProjectID: "p1", SnapshotID: "snap1", ChunkID: "c-en", SourceID: "s1",
			Span: foundation.ByteSpan{Start: 0, End: 10}, Text: "runners park",
			TextChecksum: "a", TrustLevel: foundation.TrustProject,
			Language: "en", AnalyzerVersion: "simple-v1",
		},
		index.ChunkRecord{
			ProjectID: "p1", SnapshotID: "snap1", ChunkID: "c-ru", SourceID: "s2",
			Span: foundation.ByteSpan{Start: 0, End: 10}, Text: "бегуны парк",
			TextChecksum: "b", TrustLevel: foundation.TrustProject,
			Language: "ru", AnalyzerVersion: "simple-v1",
		},
	)
	store := fake.NewVectorStore()
	ns := indexing.VectorNamespace{
		Name: "ns", ProjectID: "p1", SnapshotID: "snap1",
		EmbeddingVersion: config.DefaultEmbeddingVersion,
	}
	for _, rec := range mem.List("p1", "snap1") {
		_ = store.Upsert(context.Background(), ns, []retrieval.VectorPoint{{
			ChunkID: rec.ChunkID, ProjectID: rec.ProjectID, SnapshotID: rec.SnapshotID,
			EmbeddingVersion: config.DefaultEmbeddingVersion,
			ChunkerVersion:   "para-v1",
			MorphVersion:     "simple-v1",
			Language:         rec.Language,
			Span:             rec.Span,
			Vector:           fake.HashEmbed(rec.Text, config.DefaultEmbeddingDimension),
		}})
	}

	r := dense.Retriever{
		Store: store, Embedder: modelfake.Embedder{Dim: config.DefaultEmbeddingDimension},
		Index: mem, Namespace: ns,
	}
	plan := retrieval.RetrievalPlan{
		ID: "plan1", ProjectID: "p1", SnapshotID: "snap1", TopNRawPool: 10,
		Strategies: []retrieval.RetrieverStrategy{{RetrieverID: dense.RetrieverID}},
		Filters:    retrieval.RetrievalFilters{Language: "en"},
	}
	cands, err := r.Retrieve(context.Background(), plan, "runners")
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 1 || cands[0].ChunkID != "c-en" {
		t.Fatalf("cands=%#v", cands)
	}
	if cands[0].Contributions[0].AnalyzerVersion != "simple-v1" {
		t.Fatalf("analyzer=%q", cands[0].Contributions[0].AnalyzerVersion)
	}
	if cands[0].Contributions[0].EmbedVersion != config.DefaultEmbeddingVersion {
		t.Fatalf("embed=%q", cands[0].Contributions[0].EmbedVersion)
	}
	if cands[0].Contributions[0].Reasons[0] != foundation.ReasonDenseSimilarity {
		t.Fatalf("reason=%v", cands[0].Contributions[0].Reasons)
	}
}
