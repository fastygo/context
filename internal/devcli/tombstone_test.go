package devcli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fastygo/context/internal/devcli"
	"github.com/fastygo/context/internal/ids"
)

func TestTombstoneSourceExcludedFromSearch(t *testing.T) {
	t.Setenv("CONTEXT_ENABLE_DENSE", "")
	t.Setenv("CONTEXT_SPARSE_KIND", "")
	t.Setenv("CONTEXT_METADATA_KIND", "memory")

	root := t.TempDir()
	corpus := filepath.Join(root, "corpus")
	data := filepath.Join(root, "data")
	if err := os.MkdirAll(corpus, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corpus, "a.txt"), []byte("alpha unique phrase\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corpus, "b.txt"), []byte("beta other content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.InitProject(data, corpus, "proj_tomb", "Tomb"); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.Ingest(data, "proj_tomb", ""); err != nil {
		t.Fatal(err)
	}
	st, err := (devcli.Workspace{DataDir: data}).Load()
	if err != nil {
		t.Fatal(err)
	}
	var sourceA ids.SourceID
	for _, ch := range st.Chunks {
		if strings.Contains(ch.Text, "alpha unique") {
			sourceA = ch.SourceID
			break
		}
	}
	if sourceA == "" {
		t.Fatalf("source for a.txt not found; chunks=%#v", st.Chunks)
	}
	before, err := devcli.Search(data, "proj_tomb", "alpha unique", "exact", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Candidates) == 0 {
		t.Fatal("expected hit before tombstone")
	}
	if _, err := devcli.TombstoneSource(data, "proj_tomb", string(sourceA)); err != nil {
		t.Fatal(err)
	}
	after, err := devcli.Search(data, "proj_tomb", "alpha unique", "exact", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Candidates) != 0 {
		t.Fatalf("tombstoned source must not appear in search: %#v", after.Candidates)
	}
	beta, err := devcli.Search(data, "proj_tomb", "beta other", "exact", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(beta.Candidates) == 0 {
		t.Fatal("live source must still search")
	}
}
