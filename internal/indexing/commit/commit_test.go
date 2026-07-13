package commit_test

import (
	"testing"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/indexing/commit"
	"github.com/fastygo/context/internal/indexing/manifest"
)

func TestSealReadyAndFail(t *testing.T) {
	t.Parallel()
	b := commit.Builder{Manifest: manifest.Builder{}}
	building, err := b.Building(commit.Input{
		SnapshotID: "snap1",
		ProjectID:  "p1",
		Versions:   commit.Versions{ParserVersion: "p-v1", ChunkerVersion: "c-v1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	ready, err := b.SealReady(building, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ready.Status != foundation.SnapshotReady || ready.SourceMerkleRoot == "" || ready.ChunkSetHash == "" {
		t.Fatalf("ready=%#v", ready)
	}
	failed, err := b.Fail(building, "dense_write_failed")
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != foundation.SnapshotFailed {
		t.Fatalf("failed=%#v", failed)
	}
}
