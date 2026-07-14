package devcli

import (
	"context"
	"time"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/ids"
)

// TombstoneResult is CLI/HTTP JSON for source soft-delete.
type TombstoneResult struct {
	ProjectID  ids.ProjectID `json:"project_id"`
	SourceID   ids.SourceID  `json:"source_id"`
	Tombstoned bool          `json:"tombstoned"`
	At         time.Time     `json:"tombstoned_at"`
}

// TombstoneSource soft-deletes a source in local workspace state and, when
// configured, in the metadata store. Search/pack exclude its chunks.
func TombstoneSource(dataDir, projectID, sourceID string) (TombstoneResult, error) {
	if sourceID == "" {
		return TombstoneResult{}, apperr.New(apperr.Validation, "source_id required")
	}
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return TombstoneResult{}, err
	}
	if projectID != "" && ids.ProjectID(projectID) != st.Project.ID {
		return TombstoneResult{}, apperr.New(apperr.Permission, "project id mismatch")
	}
	src := ids.SourceID(sourceID)
	found := false
	for _, ch := range st.Chunks {
		if ch.SourceID == src {
			found = true
			break
		}
	}
	if !found {
		return TombstoneResult{}, apperr.New(apperr.NotFound, "source not found in workspace chunks")
	}
	at := time.Now().UTC()
	already := false
	for _, id := range st.TombstonedSourceIDs {
		if id == src {
			already = true
			break
		}
	}
	if !already {
		st.TombstonedSourceIDs = append(st.TombstonedSourceIDs, src)
		if err := ws.Save(st); err != nil {
			return TombstoneResult{}, err
		}
	}
	ctx := context.Background()
	if handle, err := OpenMetadata(ctx); err == nil {
		defer handle.Close()
		_ = handle.Store.TombstoneSource(ctx, st.Project.ID, src, at)
	}
	return TombstoneResult{
		ProjectID:  st.Project.ID,
		SourceID:   src,
		Tombstoned: true,
		At:         at,
	}, nil
}
