package devcli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/devcli"
)

func TestSnapshotExportImportActivateRoundTrip(t *testing.T) {
	t.Setenv("CONTEXT_ENABLE_DENSE", "")
	t.Setenv("CONTEXT_SPARSE_KIND", "")
	t.Setenv("CONTEXT_METADATA_KIND", "memory")

	root := t.TempDir()
	corpus := filepath.Join(root, "corpus")
	dataA := filepath.Join(root, "data-a")
	dataB := filepath.Join(root, "data-b")
	bundle := filepath.Join(root, "snap.bundle.json")
	if err := os.MkdirAll(corpus, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corpus, "note.md"), []byte("# Note\n\nPORTABLE42 evidence\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.InitProject(dataA, corpus, "proj_xfer", "Xfer"); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.Ingest(dataA, "proj_xfer", ""); err != nil {
		t.Fatal(err)
	}
	exp, _, err := devcli.ExportSnapshotBundle(dataA, "proj_xfer", bundle)
	if err != nil || !exp.OK || exp.Chunks < 1 {
		t.Fatalf("export=%#v err=%v", exp, err)
	}
	imp, err := devcli.ImportSnapshotBundle(dataB, "proj_xfer", bundle, true)
	if err != nil || !imp.Activated || !imp.Verified {
		t.Fatalf("import=%#v err=%v", imp, err)
	}
	hit, err := devcli.Search(dataB, "proj_xfer", "PORTABLE42", "exact", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(hit.Candidates) == 0 {
		t.Fatal("imported snapshot must be searchable")
	}
}

func TestSnapshotImportRefusesCorruptBundle(t *testing.T) {
	t.Setenv("CONTEXT_METADATA_KIND", "memory")
	root := t.TempDir()
	corpus := filepath.Join(root, "corpus")
	data := filepath.Join(root, "data")
	bundle := filepath.Join(root, "bad.bundle.json")
	if err := os.MkdirAll(corpus, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corpus, "note.md"), []byte("# Note\n\nX\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.InitProject(data, corpus, "proj_bad", "Bad"); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.Ingest(data, "proj_bad", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := devcli.ExportSnapshotBundle(data, "proj_bad", bundle); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(bundle)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	m["bundle_checksum"] = "0000000000000000000000000000000000000000000000000000000000000000"
	tampered, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(bundle, tampered, 0o644); err != nil {
		t.Fatal(err)
	}
	dataB := filepath.Join(root, "data-b")
	_, err = devcli.ImportSnapshotBundle(dataB, "proj_bad", bundle, true)
	if !apperr.Is(err, apperr.Validation) {
		t.Fatalf("want validation on corrupt bundle, got %v", err)
	}
	if _, err := (devcli.Workspace{DataDir: dataB}).Load(); !apperr.Is(err, apperr.NotFound) {
		t.Fatalf("corrupt import must not create workspace: %v", err)
	}
}
