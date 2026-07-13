// Package commit builds IndexSnapshot records with dual Merkle roots (ADR-0021).
package commit

import (
	"fmt"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing"
	"github.com/fastygo/context/internal/indexing/manifest"
)

// Versions pins parser/chunker/embed/morph/lexicon versions on a snapshot.
type Versions struct {
	ParserVersion      string
	ChunkerVersion     string
	EmbedModelVersion  string
	MorphVersion       string
	LexiconResourceVer string
}

func (v Versions) Validate() error {
	if v.ParserVersion == "" || v.ChunkerVersion == "" {
		return apperr.New(apperr.Validation, "parser_version and chunker_version required")
	}
	return nil
}

// Input is the sealed snapshot payload before status transitions.
type Input struct {
	SnapshotID       ids.SnapshotID
	ProjectID        ids.ProjectID
	ParentSnapshotID ids.SnapshotID
	Versions         Versions
	Leaves           []manifest.SourceLeaf
	ChunkHashes      []foundation.ChecksumHex
	DenseEnabled     bool
	SparseEnabled    bool
}

// Builder creates building/ready/failed snapshots.
type Builder struct {
	Manifest manifest.Builder
}

func (b Builder) Building(in Input) (indexing.IndexSnapshot, error) {
	if err := in.SnapshotID.Validate(); err != nil {
		return indexing.IndexSnapshot{}, apperr.Wrap(apperr.Validation, "snapshot_id", err)
	}
	if err := in.ProjectID.Validate(); err != nil {
		return indexing.IndexSnapshot{}, apperr.Wrap(apperr.Validation, "project_id", err)
	}
	if err := in.Versions.Validate(); err != nil {
		return indexing.IndexSnapshot{}, err
	}
	return indexing.IndexSnapshot{
		ID:               in.SnapshotID,
		ProjectID:        in.ProjectID,
		ParentSnapshotID: in.ParentSnapshotID,
		Status:           foundation.SnapshotBuilding,
		ParserVersion:    in.Versions.ParserVersion,
		ChunkerVersion:   in.Versions.ChunkerVersion,
		EmbedModelVersion: in.Versions.EmbedModelVersion,
		MorphVersion:     firstNonEmpty(in.Versions.MorphVersion, "noop-v1"),
		DenseEnabled:     in.DenseEnabled,
		SparseEnabled:    in.SparseEnabled,
		SourceMerkleAlgo: foundation.SourceMerkleAlgo,
		ChunkSetMerkleAlgo: foundation.ChunkSetMerkleAlgo,
	}, nil
}

func (b Builder) SealReady(building indexing.IndexSnapshot, leaves []manifest.SourceLeaf, chunkHashes []foundation.ChecksumHex) (indexing.IndexSnapshot, error) {
	if building.Status != foundation.SnapshotBuilding {
		return indexing.IndexSnapshot{}, apperr.New(apperr.Conflict, "only building snapshots can seal ready")
	}
	ready := building
	ready.SourceMerkleRoot = b.Manifest.SourceRoot(leaves)
	ready.ChunkSetHash = b.Manifest.ChunkRoot(chunkHashes)
	ready.Status = foundation.SnapshotReady
	ready.FailureReason = ""
	if err := ready.Validate(); err != nil {
		return indexing.IndexSnapshot{}, apperr.Wrap(apperr.Validation, "ready snapshot", err)
	}
	return ready, nil
}

func (b Builder) Fail(building indexing.IndexSnapshot, reason string) (indexing.IndexSnapshot, error) {
	if reason == "" {
		return indexing.IndexSnapshot{}, apperr.New(apperr.Validation, "failure_reason required")
	}
	failed := building
	failed.Status = foundation.SnapshotFailed
	failed.FailureReason = reason
	if err := failed.Validate(); err != nil {
		return indexing.IndexSnapshot{}, err
	}
	return failed, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// StableChunkID derives a deterministic chunk id from project and chunk hash.
func StableChunkID(projectID ids.ProjectID, chunkHash foundation.ChecksumHex) ids.ChunkID {
	return ids.ChunkID(fmt.Sprintf("chk_%s_%s", projectID, chunkHash[:16]))
}
