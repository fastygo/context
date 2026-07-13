package ignore_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fastygo/context/internal/indexing/ignore"
)

func TestMatchVendorAndGlobs(t *testing.T) {
	t.Parallel()
	pats := ignore.Compile(nil, []string{"vendor/", "*.exe", "tmp/*"})
	if !ignore.Match("vendor/pkg/x.go", pats) {
		t.Fatal("vendor prefix")
	}
	if !ignore.MatchDir("vendor", pats) {
		t.Fatal("vendor dir")
	}
	if !ignore.Match("bin/tool.exe", pats) {
		t.Fatal("exe glob")
	}
	if ignore.Match("src/main.go", pats) {
		t.Fatal("should keep source")
	}
	if !ignore.Match("tmp/a.txt", pats) {
		t.Fatal("tmp/*")
	}
}

func TestLoadFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ignore.FileName)
	body := "# comment\n\nnode_modules/\n*.log\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ignore.LoadFile(path)
	if err != nil || len(got) != 2 {
		t.Fatalf("got=%v err=%v", got, err)
	}
	pats := ignore.Compile(got, nil)
	if !ignore.Match("node_modules/x", pats) || !ignore.Match("a.log", pats) {
		t.Fatal("file patterns")
	}
}
