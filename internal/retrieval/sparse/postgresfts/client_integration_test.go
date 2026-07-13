package postgresfts

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/fastygo/context/internal/config"
	"github.com/fastygo/context/internal/retrieval"
)

func integrationDSN(t *testing.T) string {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("CONTEXT_PG_DSN"))
	if dsn == "" {
		t.Skip("set CONTEXT_PG_DSN to run postgres FTS integration tests")
	}
	return dsn
}

func TestFTSUpsertSearchIntegration(t *testing.T) {
	dsn := integrationDSN(t)
	ctx := context.Background()
	client, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer client.Close()
	if err := client.EnsureSchema(ctx); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	caps := client.Capabilities()
	if caps.BackendID != BackendID || caps.Kind != string(config.StoreKindPostgresFTS) {
		t.Fatalf("caps=%#v", caps)
	}
	if !caps.SupportsProjectFilter || !caps.SupportsSnapshotFilter {
		t.Fatal("expected project/snapshot filters")
	}
	if caps.SupportsTemporalFilter || caps.SupportsMetadataFilter {
		t.Fatal("FTS must not claim temporal/metadata server filters")
	}

	err = client.Upsert(ctx, []Document{
		{ProjectID: "p-fts", SnapshotID: "snap-a", ChunkID: "c1", Language: "en", Body: "ContextPack evidence and hybrid retrieval"},
		{ProjectID: "p-fts", SnapshotID: "snap-a", ChunkID: "c2", Language: "en", Body: "unrelated river bank erosion"},
		{ProjectID: "p-fts", SnapshotID: "snap-b", ChunkID: "c1", Language: "en", Body: "ContextPack in other snapshot"},
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	hits, err := client.Search(ctx, "p-fts", "snap-a", "ContextPack hybrid", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) == 0 || hits[0].ChunkID != "c1" {
		t.Fatalf("hits=%#v", hits)
	}

	leak, err := client.Search(ctx, "p-fts", "snap-b", "ContextPack", 10)
	if err != nil {
		t.Fatalf("Search other snap: %v", err)
	}
	for _, h := range leak {
		if h.ChunkID == "c2" {
			t.Fatalf("cross-snapshot leak: %#v", leak)
		}
	}

	empty, err := client.Search(ctx, "p-fts", "snap-a", "", 10)
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty query should yield no hits: %#v err=%v", empty, err)
	}

	_ = retrieval.SparseHit{}
}

func TestOpenRequiresDSN(t *testing.T) {
	t.Parallel()
	_, err := Open(context.Background(), "")
	if err == nil {
		t.Fatal("expected error")
	}
}
