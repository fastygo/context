package devcli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fastygo/context/internal/devcli"
)

func TestInspectQueryAndPackID(t *testing.T) {
	t.Setenv("CONTEXT_ENABLE_DENSE", "")
	t.Setenv("CONTEXT_SPARSE_KIND", "")
	root := t.TempDir()
	corpus := filepath.Join(root, "corpus")
	data := filepath.Join(root, "data")
	if err := os.MkdirAll(corpus, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "# Promo\n\nNew Year Mongolian dishes action want REPAIR42 token.\n"
	if err := os.WriteFile(filepath.Join(corpus, "menu.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.InitProject(data, corpus, "proj_insp", "Inspect"); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.Ingest(data, "proj_insp", ""); err != nil {
		t.Fatal(err)
	}

	rep, err := devcli.Inspect(data, "proj_insp", "REPAIR42", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !rep.OK || rep.PackID == "" || len(rep.Selected) < 1 {
		t.Fatalf("%#v", rep)
	}
	raw, err := json.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), filepath.ToSlash(corpus)) || strings.Contains(string(raw), data) {
		t.Fatalf("inspector leaked host path: %s", raw)
	}
	if rep.Budget.SelectedCount != len(rep.Selected) {
		t.Fatalf("budget counts: %#v", rep.Budget)
	}

	again, err := devcli.Inspect(data, "proj_insp", "", "", string(rep.PackID))
	if err != nil {
		t.Fatal(err)
	}
	if again.Mode != "pack_id" || again.PackID != rep.PackID {
		t.Fatalf("%#v", again)
	}

	_, err = devcli.Inspect(data, "other", "REPAIR42", "", "")
	if err == nil {
		t.Fatal("expected project mismatch")
	}
}
