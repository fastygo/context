// Package snapshotxfer defines portable IndexSnapshot bundles (stabilization C2).
// Export/import verifies integrity before any active_snapshot flip (ADR-0012/0021).
package snapshotxfer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing"
	"github.com/fastygo/context/internal/indexing/hashing"
)

// FormatVersion is the sealed bundle schema id.
const FormatVersion = "snapshot-bundle-v1"

// Chunk is a portable indexed chunk (no host absolute paths).
type Chunk struct {
	ChunkID          ids.ChunkID            `json:"chunk_id"`
	SourceID         ids.SourceID           `json:"source_id"`
	SnapshotID       ids.SnapshotID         `json:"snapshot_id"`
	PathKey          string                 `json:"path_key"`
	RelativePath     string                 `json:"relative_path,omitempty"`
	SpanStart        uint64                 `json:"span_start"`
	SpanEnd          uint64                 `json:"span_end"`
	Text             string                 `json:"text"`
	TextChecksum     foundation.ChecksumHex `json:"text_checksum"`
	ChunkHash        foundation.ChecksumHex `json:"chunk_hash"`
	TrustLevel       foundation.TrustLevel  `json:"trust_level"`
	ChunkerVersion   string                 `json:"chunker_version,omitempty"`
	EmbeddingVersion string                 `json:"embedding_version,omitempty"`
	MorphVersion     string                 `json:"morph_version,omitempty"`
	Language         string                 `json:"language,omitempty"`
}

// Bundle is a sealed, checksummed snapshot transport unit.
type Bundle struct {
	FormatVersion       string                  `json:"format_version"`
	ExportedAt          time.Time               `json:"exported_at"`
	Project             corpus.Project          `json:"project"`
	Snapshot            indexing.IndexSnapshot  `json:"snapshot"`
	Chunks              []Chunk                 `json:"chunks"`
	TombstonedSourceIDs []ids.SourceID          `json:"tombstoned_source_ids,omitempty"`
	BundleChecksum      foundation.ChecksumHex  `json:"bundle_checksum"`
}

// payloadForChecksum is the canonical body hashed into BundleChecksum.
type payloadForChecksum struct {
	FormatVersion       string                 `json:"format_version"`
	ExportedAt          time.Time              `json:"exported_at"`
	Project             corpus.Project         `json:"project"`
	Snapshot            indexing.IndexSnapshot `json:"snapshot"`
	Chunks              []Chunk                `json:"chunks"`
	TombstonedSourceIDs []ids.SourceID         `json:"tombstoned_source_ids,omitempty"`
}

// SealChecksum computes BundleChecksum over the payload (excluding the checksum field).
func SealChecksum(b Bundle) (foundation.ChecksumHex, error) {
	p := payloadForChecksum{
		FormatVersion:       b.FormatVersion,
		ExportedAt:          b.ExportedAt.UTC(),
		Project:             b.Project,
		Snapshot:            b.Snapshot,
		Chunks:              b.Chunks,
		TombstonedSourceIDs: b.TombstonedSourceIDs,
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return "", apperr.Wrap(apperr.Internal, "bundle marshal", err)
	}
	sum := sha256.Sum256(raw)
	return foundation.ChecksumHex(hex.EncodeToString(sum[:])), nil
}

// Verify rejects corrupt, partial, or non-activatable bundles.
func Verify(b Bundle) error {
	if b.FormatVersion != FormatVersion {
		return apperr.New(apperr.Validation, fmt.Sprintf("unsupported format_version %q", b.FormatVersion))
	}
	if err := b.Project.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "project", err)
	}
	if err := b.Snapshot.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "snapshot", err)
	}
	if b.Snapshot.ProjectID != b.Project.ID {
		return apperr.New(apperr.Validation, "snapshot project_id mismatch")
	}
	if b.Snapshot.Status != foundation.SnapshotReady {
		return apperr.New(apperr.Conflict, "only ready snapshots may be imported for activation")
	}
	if !b.Snapshot.Status.IsSearchableAsActive() {
		return apperr.New(apperr.Conflict, "snapshot status not searchable as active")
	}
	want, err := SealChecksum(b)
	if err != nil {
		return err
	}
	if b.BundleChecksum != want {
		return apperr.New(apperr.Validation, "bundle_checksum mismatch (corrupt or tampered)")
	}
	if len(b.Chunks) == 0 {
		return apperr.New(apperr.Validation, "bundle has no chunks (partial)")
	}
	hashes := make([]foundation.ChecksumHex, 0, len(b.Chunks))
	seen := make(map[ids.ChunkID]struct{}, len(b.Chunks))
	for i, ch := range b.Chunks {
		if ch.SnapshotID != b.Snapshot.ID {
			return apperr.New(apperr.Validation, fmt.Sprintf("chunk[%d] snapshot_id mismatch", i))
		}
		if ch.ChunkID == "" || ch.ChunkHash == "" {
			return apperr.New(apperr.Validation, fmt.Sprintf("chunk[%d] incomplete", i))
		}
		if _, dup := seen[ch.ChunkID]; dup {
			return apperr.New(apperr.Validation, fmt.Sprintf("duplicate chunk_id %q", ch.ChunkID))
		}
		seen[ch.ChunkID] = struct{}{}
		if err := ch.ChunkHash.Validate(); err != nil {
			return apperr.Wrap(apperr.Validation, fmt.Sprintf("chunk[%d] hash", i), err)
		}
		hashes = append(hashes, ch.ChunkHash)
	}
	gotSet := hashing.ChunkSetHash(hashes)
	if gotSet != b.Snapshot.ChunkSetHash {
		return apperr.New(apperr.Validation, "chunk_set_hash mismatch (partial or corrupt chunks)")
	}
	return nil
}

// NewBundle builds a sealed bundle from project/snapshot/chunks.
func NewBundle(
	project corpus.Project,
	snap indexing.IndexSnapshot,
	chunks []Chunk,
	tombstoned []ids.SourceID,
	exportedAt time.Time,
) (Bundle, error) {
	b := Bundle{
		FormatVersion: FormatVersion,
		ExportedAt:    exportedAt.UTC(),
		Project: corpus.Project{
			ID:               project.ID,
			Name:             project.Name,
			TenantID:         project.TenantID,
			ActiveSnapshotID: snap.ID,
		},
		Snapshot:            snap,
		Chunks:              append([]Chunk(nil), chunks...),
		TombstonedSourceIDs: append([]ids.SourceID(nil), tombstoned...),
	}
	sort.Slice(b.Chunks, func(i, j int) bool { return b.Chunks[i].ChunkID < b.Chunks[j].ChunkID })
	sum, err := SealChecksum(b)
	if err != nil {
		return Bundle{}, err
	}
	b.BundleChecksum = sum
	if err := Verify(b); err != nil {
		return Bundle{}, err
	}
	return b, nil
}
