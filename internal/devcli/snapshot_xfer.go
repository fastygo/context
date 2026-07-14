package devcli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing/snapshotxfer"
)

// SnapshotExportResult is CLI/HTTP JSON after writing a bundle file.
type SnapshotExportResult struct {
	OK             bool           `json:"ok"`
	ProjectID      ids.ProjectID  `json:"project_id"`
	SnapshotID     ids.SnapshotID `json:"snapshot_id"`
	Chunks         int            `json:"chunks"`
	OutPath        string         `json:"out_path,omitempty"`
	BundleChecksum string         `json:"bundle_checksum"`
}

// SnapshotImportResult is CLI/HTTP JSON after verifying (and optionally activating) a bundle.
type SnapshotImportResult struct {
	OK         bool           `json:"ok"`
	ProjectID  ids.ProjectID  `json:"project_id"`
	SnapshotID ids.SnapshotID `json:"snapshot_id"`
	Chunks     int            `json:"chunks"`
	Activated  bool           `json:"activated"`
	Verified   bool           `json:"verified"`
}

// ExportSnapshotBundle builds a sealed portable bundle for the active ready snapshot.
// When outPath is non-empty, writes JSON to that path (must not be under host corpus
// secrets — callers pass an explicit path). Host CorpusRoot is not included.
func ExportSnapshotBundle(dataDir, projectID, outPath string) (SnapshotExportResult, snapshotxfer.Bundle, error) {
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return SnapshotExportResult{}, snapshotxfer.Bundle{}, err
	}
	if projectID != "" && ids.ProjectID(projectID) != st.Project.ID {
		return SnapshotExportResult{}, snapshotxfer.Bundle{}, apperr.New(apperr.Permission, "project id mismatch")
	}
	if st.Snapshot.Status != foundation.SnapshotReady {
		return SnapshotExportResult{}, snapshotxfer.Bundle{}, apperr.New(apperr.Conflict, "active snapshot is not ready")
	}
	snapID := st.Project.ActiveSnapshotID
	if snapID == "" {
		snapID = st.Snapshot.ID
	}
	if st.Snapshot.ID != snapID {
		return SnapshotExportResult{}, snapshotxfer.Bundle{}, apperr.New(apperr.Conflict, "state snapshot id mismatch")
	}
	var chunks []snapshotxfer.Chunk
	for _, ch := range st.Chunks {
		if ch.SnapshotID != snapID {
			continue
		}
		chunks = append(chunks, snapshotxfer.Chunk{
			ChunkID: ch.ChunkID, SourceID: ch.SourceID, SnapshotID: ch.SnapshotID,
			PathKey: ch.PathKey, RelativePath: ch.RelativePath,
			SpanStart: ch.SpanStart, SpanEnd: ch.SpanEnd, Text: ch.Text,
			TextChecksum: ch.TextChecksum, ChunkHash: ch.ChunkHash, TrustLevel: ch.TrustLevel,
			ChunkerVersion: ch.ChunkerVersion, EmbeddingVersion: ch.EmbeddingVersion,
			MorphVersion: ch.MorphVersion, Language: ch.Language,
		})
	}
	b, err := snapshotxfer.NewBundle(st.Project, st.Snapshot, chunks, st.TombstonedSourceIDs, time.Now().UTC())
	if err != nil {
		return SnapshotExportResult{}, snapshotxfer.Bundle{}, err
	}
	res := SnapshotExportResult{
		OK: true, ProjectID: st.Project.ID, SnapshotID: snapID,
		Chunks: len(b.Chunks), BundleChecksum: string(b.BundleChecksum),
	}
	if outPath != "" {
		if err := writeBundleFile(outPath, b); err != nil {
			return SnapshotExportResult{}, snapshotxfer.Bundle{}, err
		}
		res.OutPath = filepath.ToSlash(outPath)
	}
	return res, b, nil
}

// ImportSnapshotBundle verifies a bundle file and optionally activates it into dataDir.
// Corrupt or partial bundles never flip active_snapshot_id.
func ImportSnapshotBundle(dataDir, projectID, inPath string, activate bool) (SnapshotImportResult, error) {
	raw, err := os.ReadFile(inPath)
	if err != nil {
		return SnapshotImportResult{}, apperr.Wrap(apperr.Validation, "read bundle", err)
	}
	return ImportSnapshotBundleBytes(dataDir, projectID, raw, activate)
}

// ImportSnapshotBundleBytes verifies raw bundle JSON and optionally activates it.
func ImportSnapshotBundleBytes(dataDir, projectID string, raw []byte, activate bool) (SnapshotImportResult, error) {
	var b snapshotxfer.Bundle
	if err := json.Unmarshal(raw, &b); err != nil {
		return SnapshotImportResult{}, apperr.Wrap(apperr.Validation, "decode bundle", err)
	}
	if err := snapshotxfer.Verify(b); err != nil {
		return SnapshotImportResult{}, err
	}
	if projectID != "" && ids.ProjectID(projectID) != b.Project.ID {
		return SnapshotImportResult{}, apperr.New(apperr.Permission, "project id mismatch")
	}
	res := SnapshotImportResult{
		OK: true, ProjectID: b.Project.ID, SnapshotID: b.Snapshot.ID,
		Chunks: len(b.Chunks), Verified: true,
	}
	if !activate {
		return res, nil
	}
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		if !apperr.Is(err, apperr.NotFound) {
			return SnapshotImportResult{}, err
		}
		st = State{}
	}
	if st.Project.ID != "" && st.Project.ID != b.Project.ID {
		return SnapshotImportResult{}, apperr.New(apperr.Conflict, "workspace project differs from bundle; refuse activate")
	}
	st.Project = b.Project
	st.Project.ActiveSnapshotID = b.Snapshot.ID
	st.Snapshot = b.Snapshot
	st.Chunks = make([]IndexedChunk, 0, len(b.Chunks))
	for _, ch := range b.Chunks {
		st.Chunks = append(st.Chunks, IndexedChunk{
			ChunkID: ch.ChunkID, SourceID: ch.SourceID, SnapshotID: ch.SnapshotID,
			PathKey: ch.PathKey, RelativePath: ch.RelativePath,
			SpanStart: ch.SpanStart, SpanEnd: ch.SpanEnd, Text: ch.Text,
			TextChecksum: ch.TextChecksum, ChunkHash: ch.ChunkHash, TrustLevel: ch.TrustLevel,
			ChunkerVersion: ch.ChunkerVersion, EmbeddingVersion: ch.EmbeddingVersion,
			MorphVersion: ch.MorphVersion, Language: ch.Language,
		})
	}
	st.TombstonedSourceIDs = append([]ids.SourceID(nil), b.TombstonedSourceIDs...)
	st.LastFailed = nil
	if err := ws.Save(st); err != nil {
		return SnapshotImportResult{}, err
	}
	res.Activated = true
	return res, nil
}

func writeBundleFile(path string, b snapshotxfer.Bundle) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return apperr.Wrap(apperr.Validation, "create bundle dir", err)
	}
	raw, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.Internal, "encode bundle", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return apperr.Wrap(apperr.Validation, "write bundle tmp", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return apperr.Wrap(apperr.Validation, "replace bundle", err)
	}
	return nil
}
