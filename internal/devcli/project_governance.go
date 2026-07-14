package devcli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/artifacts/localfs"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing/snapshotxfer"
	"github.com/fastygo/context/internal/retrieval"
)

const projectArchiveFormat = "project-archive-v1"

// ProjectArchive is a portable project export without host paths (stabilization C7).
type ProjectArchive struct {
	FormatVersion string                   `json:"format_version"`
	ExportedAt    time.Time                `json:"exported_at"`
	Snapshot      snapshotxfer.Bundle      `json:"snapshot"`
	Focuses       []retrieval.FocusProfile `json:"focuses,omitempty"`
}

// ProjectExportResult is CLI/HTTP JSON after writing a project archive.
type ProjectExportResult struct {
	OK         bool           `json:"ok"`
	ProjectID  ids.ProjectID  `json:"project_id"`
	SnapshotID ids.SnapshotID `json:"snapshot_id"`
	Chunks     int            `json:"chunks"`
	Focuses    int            `json:"focuses"`
	OutPath    string         `json:"out_path,omitempty"`
}

// ProjectDeleteResult reports what was removed.
type ProjectDeleteResult struct {
	OK                    bool          `json:"ok"`
	ProjectID             ids.ProjectID `json:"project_id"`
	WorkspaceCleared      bool          `json:"workspace_cleared"`
	MetadataDeleted       bool          `json:"metadata_deleted"`
	ArtifactsDeleted      bool          `json:"artifacts_deleted"`
	SourcesTombstoned     int           `json:"sources_tombstoned_before_delete"`
}

// ExportProject writes a project archive (snapshot bundle + focuses, no host paths).
func ExportProject(dataDir, projectID, outPath string) (ProjectExportResult, error) {
	meta, bundle, err := ExportSnapshotBundle(dataDir, projectID, "")
	if err != nil {
		return ProjectExportResult{}, err
	}
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return ProjectExportResult{}, err
	}
	if outPath == "" {
		return ProjectExportResult{}, apperr.New(apperr.Validation, "out path required")
	}
	arch := ProjectArchive{
		FormatVersion: projectArchiveFormat,
		ExportedAt:    time.Now().UTC(),
		Snapshot:      bundle,
		Focuses:       append([]retrieval.FocusProfile(nil), st.Focuses...),
	}
	if err := writeJSONFile(outPath, arch); err != nil {
		return ProjectExportResult{}, err
	}
	return ProjectExportResult{
		OK: true, ProjectID: meta.ProjectID, SnapshotID: meta.SnapshotID,
		Chunks: meta.Chunks, Focuses: len(arch.Focuses), OutPath: filepath.ToSlash(outPath),
	}, nil
}

// DeleteProject removes workspace search state, artifact bytes, and metadata rows.
// Requires confirmProjectID to match projectID (safety latch).
func DeleteProject(dataDir, projectID, confirmProjectID string) (ProjectDeleteResult, error) {
	if confirmProjectID == "" || confirmProjectID != projectID {
		return ProjectDeleteResult{}, apperr.New(apperr.Validation, "confirm_project_id must equal project_id")
	}
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return ProjectDeleteResult{}, err
	}
	if ids.ProjectID(projectID) != st.Project.ID {
		return ProjectDeleteResult{}, apperr.New(apperr.Permission, "project id mismatch")
	}
	res := ProjectDeleteResult{OK: true, ProjectID: st.Project.ID}
	at := time.Now().UTC()
	seen := map[ids.SourceID]struct{}{}
	for _, ch := range st.Chunks {
		if _, ok := seen[ch.SourceID]; ok {
			continue
		}
		seen[ch.SourceID] = struct{}{}
		already := false
		for _, id := range st.TombstonedSourceIDs {
			if id == ch.SourceID {
				already = true
				break
			}
		}
		if !already {
			st.TombstonedSourceIDs = append(st.TombstonedSourceIDs, ch.SourceID)
			res.SourcesTombstoned++
		}
	}

	ctx := context.Background()
	if handle, err := OpenMetadata(ctx); err == nil {
		defer handle.Close()
		for id := range seen {
			_ = handle.Store.TombstoneSource(ctx, st.Project.ID, id, at)
		}
		if err := handle.Store.DeleteProject(ctx, st.Project.ID); err != nil {
			if !apperr.Is(err, apperr.NotFound) {
				return ProjectDeleteResult{}, err
			}
		} else {
			res.MetadataDeleted = true
		}
	}

	arts, err := localfs.New(ws.ArtifactsDir())
	if err == nil {
		if err := arts.DeleteProject(ctx, st.Project.ID); err != nil {
			return ProjectDeleteResult{}, err
		}
		res.ArtifactsDeleted = true
	}

	if err := os.Remove(ws.StatePath()); err != nil && !os.IsNotExist(err) {
		return ProjectDeleteResult{}, apperr.Wrap(apperr.Validation, "clear workspace state", err)
	}
	res.WorkspaceCleared = true
	return res, nil
}

func writeJSONFile(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return apperr.Wrap(apperr.Validation, "create archive dir", err)
	}
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.Internal, "encode archive", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return apperr.Wrap(apperr.Validation, "write archive tmp", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return apperr.Wrap(apperr.Validation, "replace archive", err)
	}
	return nil
}
