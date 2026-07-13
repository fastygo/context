package pipeline_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/indexing/manifest"
	"github.com/fastygo/context/internal/indexing/pipeline"
	"github.com/fastygo/context/internal/indexing/source"
)

func TestIndexingStableAcrossReruns(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write(t, root, "a.txt", "hello\n\nworld\n")
	write(t, root, "doc.md", "# Hi\n\nSection body.\n")

	runner := pipeline.NewDefault(source.LocalFiles{})
	ctx := context.Background()
	r1, err := runner.Run(ctx, "proj1", "snap1", root, nil)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := runner.Run(ctx, "proj1", "snap2", root, leafMap(r1))
	if err != nil {
		t.Fatal(err)
	}
	if r1.Snapshot.SourceMerkleRoot != r2.Snapshot.SourceMerkleRoot {
		t.Fatalf("source root unstable: %s vs %s", r1.Snapshot.SourceMerkleRoot, r2.Snapshot.SourceMerkleRoot)
	}
	if r1.Snapshot.ChunkSetHash != r2.Snapshot.ChunkSetHash {
		t.Fatalf("chunk set unstable: %s vs %s", r1.Snapshot.ChunkSetHash, r2.Snapshot.ChunkSetHash)
	}
	if r1.Snapshot.Status != foundation.SnapshotReady {
		t.Fatalf("status=%s", r1.Snapshot.Status)
	}
	for _, ch := range r2.SourceDiff {
		if ch.Kind != manifest.Unchanged {
			t.Fatalf("expected all unchanged, got %#v", r2.SourceDiff)
		}
	}
	if len(r1.Tokens) == 0 {
		t.Fatal("expected token spans")
	}
}

func TestEditMarksOnlyChangedSource(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write(t, root, "a.txt", "one\n")
	write(t, root, "b.txt", "two\n")
	runner := pipeline.NewDefault(source.LocalFiles{})
	ctx := context.Background()
	r1, err := runner.Run(ctx, "proj1", "snap1", root, nil)
	if err != nil {
		t.Fatal(err)
	}
	write(t, root, "b.txt", "two changed\n")
	r2, err := runner.Run(ctx, "proj1", "snap2", root, leafMap(r1))
	if err != nil {
		t.Fatal(err)
	}
	var changed, unchanged int
	for _, ch := range r2.SourceDiff {
		switch ch.Kind {
		case manifest.Changed:
			changed++
		case manifest.Unchanged:
			unchanged++
		case manifest.Added, manifest.Removed:
			t.Fatalf("unexpected kind %#v", ch)
		}
	}
	if changed != 1 || unchanged != 1 {
		t.Fatalf("changed=%d unchanged=%d diff=%#v", changed, unchanged, r2.SourceDiff)
	}
	if r1.Snapshot.SourceMerkleRoot == r2.Snapshot.SourceMerkleRoot {
		t.Fatal("source root should change after edit")
	}
}

func write(t *testing.T, root, name, body string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func leafMap(r pipeline.Result) map[string]foundation.ChecksumHex {
	out := make(map[string]foundation.ChecksumHex, len(r.Leaves))
	for _, leaf := range r.Leaves {
		out[leaf.PathKey] = leaf.LeafHash
	}
	return out
}
