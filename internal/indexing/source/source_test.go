package source_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fastygo/context/internal/indexing/ignore"
	"github.com/fastygo/context/internal/indexing/source"
)

func TestLocalFilesSkipsVendorAndContextignore(t *testing.T) {
	root := t.TempDir()
	mustWrite := func(rel, body string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("keep/note.md", "# keep\n")
	mustWrite("vendor/lib/x.md", "# vendor\n")
	mustWrite("build/out.md", "# build\n")
	mustWrite("secret/skip.md", "# skip\n")
	mustWrite(ignore.FileName, "secret/\n")

	got, err := (source.LocalFiles{}).List(context.Background(), "p1", root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].RelativePath != "keep/note.md" {
		t.Fatalf("got=%#v", got)
	}
}
