package devcli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fastygo/context/internal/devcli"
	"github.com/fastygo/context/internal/models/localhash"
)

func TestLocalHashDenseCommitIntegration(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("CONTEXT_PG_DSN"))
	if dsn == "" {
		t.Skip("set CONTEXT_PG_DSN to run local_hash dense integration")
	}
	t.Setenv("CONTEXT_PG_DSN", dsn)
	t.Setenv("CONTEXT_ENABLE_DENSE", "1")
	t.Setenv("CONTEXT_EMBEDDER_KIND", "local_hash")
	t.Setenv("CONTEXT_EMBEDDING_VERSION", localhash.Version)
	t.Setenv("CONTEXT_EMBEDDING_DIMENSION", "32")
	t.Setenv("CONTEXT_VECTOR_COLLECTION", "context_dense_local_hash_v1")
	t.Setenv("CONTEXT_METADATA_KIND", "memory")
	t.Setenv("CONTEXT_DENSE_REBUILD", "")

	root := t.TempDir()
	data := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("# Note\n\nContextPack local hash embedder.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.InitProject(data, root, "local-hash", "LocalHash"); err != nil {
		t.Fatal(err)
	}
	st, err := devcli.Ingest(data, "local-hash", root)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if st.Snapshot.EmbedModelVersion != localhash.Version {
		t.Fatalf("embed=%q", st.Snapshot.EmbedModelVersion)
	}
	if st.Chunks[0].EmbeddingVersion != localhash.Version {
		t.Fatalf("chunk=%q", st.Chunks[0].EmbeddingVersion)
	}

	res, err := devcli.Search(data, "local-hash", "ContextPack local", "dense", "")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Candidates) == 0 {
		t.Fatal("expected dense hits from local_hash vectors")
	}
}
