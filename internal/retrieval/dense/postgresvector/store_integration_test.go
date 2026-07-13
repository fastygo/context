package postgresvector

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/fastygo/context/internal/config"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/indexing"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/fake"
)

func integrationDSN(t *testing.T) string {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("CONTEXT_PG_DSN"))
	if dsn == "" {
		t.Skip("set CONTEXT_PG_DSN to run pgvector integration tests")
	}
	return dsn
}

func TestStoreUpsertSearchIntegration(t *testing.T) {
	dsn := integrationDSN(t)
	ctx := context.Background()
	store, err := Open(ctx, dsn, Config{
		Collection: "context_dense_v1_test",
		Dimension:  config.DefaultEmbeddingDimension,
		Metric:     config.DefaultVectorMetric,
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	caps := store.Capabilities()
	if !caps.SupportsProjectFilter || !caps.SupportsSnapshotFilter {
		t.Fatalf("expected project/snapshot filters: %#v", caps)
	}
	if caps.SupportsTemporalFilter {
		t.Fatal("pgvector PoC must not claim server-side temporal filters")
	}

	ns := indexing.VectorNamespace{
		Name: "context_dense_v1_test", ProjectID: "p-int", SnapshotID: "snap-int",
		EmbeddingVersion: config.DefaultEmbeddingVersion,
	}
	text := "runners run in the park"
	vec := fake.HashEmbed(text, config.DefaultEmbeddingDimension)
	err = store.Upsert(ctx, ns, []retrieval.VectorPoint{{
		ChunkID: "c-run", ProjectID: "p-int", SnapshotID: "snap-int",
		EmbeddingVersion: config.DefaultEmbeddingVersion,
		ChunkerVersion:   "para-v1",
		MorphVersion:     "simple-v1",
		ContextRef:       "ref-1",
		Language:         "en",
		Span:             foundation.ByteSpan{Start: 0, End: uint64(len(text))},
		Vector:           vec,
	}})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	otherNS := ns
	otherNS.SnapshotID = "snap-other"
	leak, err := store.Search(ctx, otherNS, vec, 5)
	if err != nil {
		t.Fatalf("Search other snapshot: %v", err)
	}
	if len(leak) != 0 {
		t.Fatalf("cross-snapshot leak: %#v", leak)
	}

	hits, err := store.Search(ctx, ns, vec, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 1 || hits[0].ChunkID != "c-run" {
		t.Fatalf("hits=%#v", hits)
	}
	if hits[0].EmbeddingVersion != config.DefaultEmbeddingVersion {
		t.Fatalf("embedding_version=%q", hits[0].EmbeddingVersion)
	}
	if hits[0].ChunkerVersion != "para-v1" || hits[0].MorphVersion != "simple-v1" {
		t.Fatalf("provenance=%#v", hits[0])
	}
	if hits[0].ContextRef != "ref-1" || hits[0].SnapshotID != "snap-int" {
		t.Fatalf("refs=%#v", hits[0])
	}
	if hits[0].Score < 0.99 {
		t.Fatalf("expected near-identical cosine score, got %v", hits[0].Score)
	}
}

func TestStoreRejectsDimensionMismatch(t *testing.T) {
	dsn := integrationDSN(t)
	ctx := context.Background()
	store, err := Open(ctx, dsn, Config{Dimension: 8, Metric: "cosine", Collection: "dim-test"})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	ns := indexing.VectorNamespace{
		Name: "dim-test", ProjectID: "p1", SnapshotID: "s1", EmbeddingVersion: "v1",
	}
	err = store.Upsert(ctx, ns, []retrieval.VectorPoint{{
		ChunkID: "c1", ProjectID: "p1", SnapshotID: "s1", EmbeddingVersion: "v1",
		Vector: []float32{1, 2, 3},
	}})
	if err == nil {
		t.Fatal("expected dimension error")
	}
}
