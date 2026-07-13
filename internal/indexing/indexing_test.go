package indexing_test

import (
	"testing"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/indexing"
)

func TestReadySnapshotRequiresRoots(t *testing.T) {
	t.Parallel()
	s := indexing.IndexSnapshot{
		ID:        "snap1",
		ProjectID: "p1",
		Status:    foundation.SnapshotReady,
	}
	if err := s.Validate(); err == nil {
		t.Fatal("ready without merkle roots should fail")
	}
	s.SourceMerkleRoot = "aa"
	s.ChunkSetHash = "bb"
	s.ParserVersion = "p1"
	s.ChunkerVersion = "c1"
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestFailedSnapshotRequiresReason(t *testing.T) {
	t.Parallel()
	s := indexing.IndexSnapshot{
		ID:        "snap1",
		ProjectID: "p1",
		Status:    foundation.SnapshotFailed,
	}
	if err := s.Validate(); err == nil {
		t.Fatal("failed without reason should fail")
	}
	s.FailureReason = "dense_write_failed"
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestBuildingSnapshotMinimal(t *testing.T) {
	t.Parallel()
	s := indexing.IndexSnapshot{
		ID:        "snap1",
		ProjectID: "p1",
		Status:    foundation.SnapshotBuilding,
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}
