package snapshotxfer_test

import (
	"testing"
	"time"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/indexing"
	"github.com/fastygo/context/internal/indexing/hashing"
	"github.com/fastygo/context/internal/indexing/snapshotxfer"
)

func sampleBundle(t *testing.T) snapshotxfer.Bundle {
	t.Helper()
	h1 := foundation.ChecksumHex("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	h2 := foundation.ChecksumHex("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	set := hashing.ChunkSetHash([]foundation.ChecksumHex{h1, h2})
	snap := indexing.IndexSnapshot{
		ID: "snap1", ProjectID: "p1", Status: foundation.SnapshotReady,
		SourceMerkleRoot: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		ChunkSetHash:     set,
		ParserVersion:    "plain-v1", ChunkerVersion: "paragraph-v1",
		SourceMerkleAlgo: foundation.SourceMerkleAlgo, ChunkSetMerkleAlgo: foundation.ChunkSetMerkleAlgo,
	}
	b, err := snapshotxfer.NewBundle(
		corpus.Project{ID: "p1", Name: "Demo"},
		snap,
		[]snapshotxfer.Chunk{
			{ChunkID: "c1", SourceID: "s1", SnapshotID: "snap1", PathKey: "a", SpanStart: 0, SpanEnd: 1,
				Text: "a", TextChecksum: "d1", ChunkHash: h1, TrustLevel: foundation.TrustProject},
			{ChunkID: "c2", SourceID: "s2", SnapshotID: "snap1", PathKey: "b", SpanStart: 0, SpanEnd: 1,
				Text: "b", TextChecksum: "d2", ChunkHash: h2, TrustLevel: foundation.TrustProject},
		},
		nil,
		time.Unix(100, 0).UTC(),
	)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestVerifyRejectsChecksumTamper(t *testing.T) {
	t.Parallel()
	b := sampleBundle(t)
	b.BundleChecksum = "deadbeef"
	if err := snapshotxfer.Verify(b); !apperr.Is(err, apperr.Validation) {
		t.Fatalf("want validation, got %v", err)
	}
}

func TestVerifyRejectsPartialChunks(t *testing.T) {
	t.Parallel()
	b := sampleBundle(t)
	b.Chunks = b.Chunks[:1]
	sum, err := snapshotxfer.SealChecksum(b)
	if err != nil {
		t.Fatal(err)
	}
	b.BundleChecksum = sum
	if err := snapshotxfer.Verify(b); !apperr.Is(err, apperr.Validation) {
		t.Fatalf("want chunk_set mismatch, got %v", err)
	}
}

func TestVerifyRejectsNonReady(t *testing.T) {
	t.Parallel()
	b := sampleBundle(t)
	b.Snapshot.Status = foundation.SnapshotBuilding
	b.Snapshot.SourceMerkleRoot = ""
	b.Snapshot.ChunkSetHash = ""
	sum, err := snapshotxfer.SealChecksum(b)
	if err != nil {
		t.Fatal(err)
	}
	b.BundleChecksum = sum
	if err := snapshotxfer.Verify(b); err == nil {
		t.Fatal("expected reject non-ready")
	}
}
