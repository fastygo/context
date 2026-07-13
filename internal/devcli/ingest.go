package devcli

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/artifacts/localfs"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing/commit"
	"github.com/fastygo/context/internal/indexing/hashing"
	"github.com/fastygo/context/internal/indexing/pipeline"
	"github.com/fastygo/context/internal/indexing/source"
)

// Ingest indexes files under path into the workspace snapshot.
func Ingest(dataDir, projectID, path string) (State, error) {
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return State{}, err
	}
	if projectID != "" && ids.ProjectID(projectID) != st.Project.ID {
		return State{}, apperr.New(apperr.Validation, "project id mismatch with workspace")
	}
	root := path
	if root == "" {
		root = st.CorpusRoot
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return State{}, apperr.Wrap(apperr.Validation, "ingest path", err)
	}

	arts, err := localfs.New(ws.ArtifactsDir())
	if err != nil {
		return State{}, err
	}
	runner := pipeline.NewDefault(source.LocalFiles{})
	snapID := ids.SnapshotID(fmt.Sprintf("snap_%d", len(st.Chunks)+1))
	res, err := runner.Run(context.Background(), st.Project.ID, snapID, absRoot, nil)
	if err != nil {
		return State{}, err
	}

	chunks := make([]IndexedChunk, 0)
	for _, leaf := range res.Leaves {
		raws := res.RawChunks[leaf.PathKey]
		version := chunkerVersionFor(leaf.RelativePath)
		sourceID := sourceIDFromPathKey(leaf.PathKey)
		artID := ids.ArtifactID("src_" + sanitizeID(string(sourceID)))
		if _, err := arts.Put(context.Background(), st.Project.ID, artID, "application/octet-stream", []byte(leaf.RelativePath), nil); err != nil {
			_ = err
		}
		for _, rc := range raws {
			chHash := hashing.ChunkHash(version, leaf.PathKey, rc.Span.Start, rc.Span.End, rc.Text)
			chunkID := commit.StableChunkID(st.Project.ID, chHash)
			chunks = append(chunks, IndexedChunk{
				ChunkID:      chunkID,
				SourceID:     sourceID,
				SnapshotID:   res.Snapshot.ID,
				PathKey:      leaf.PathKey,
				RelativePath: leaf.RelativePath,
				SpanStart:    rc.Span.Start,
				SpanEnd:      rc.Span.End,
				Text:         rc.Text,
				TextChecksum: rc.TextChecksum,
				ChunkHash:    chHash,
				TrustLevel:   foundation.TrustProject,
			})
		}
	}

	st.Snapshot = res.Snapshot
	st.Project.ActiveSnapshotID = res.Snapshot.ID
	st.Chunks = chunks
	st.CorpusRoot = absRoot
	if err := ws.Save(st); err != nil {
		return State{}, err
	}

	ctx := context.Background()
	handle, err := OpenMetadata(ctx)
	if err != nil {
		return State{}, err
	}
	defer handle.Close()
	if handle.UsesPostgres() {
		if err := PersistIngest(ctx, handle.Store, st.Project, res.Snapshot, res.Leaves, chunks); err != nil {
			return State{}, apperr.Wrap(apperr.Internal, "persist ingest metadata", err)
		}
	}

	sparseH, err := OpenSparse(ctx, nil)
	if err != nil {
		return State{}, err
	}
	defer sparseH.Closer()
	if sparseH.UsesFTS {
		if err := sparseH.UpsertChunks(ctx, st.Project.ID, chunks); err != nil {
			return State{}, apperr.Wrap(apperr.Internal, "persist ingest sparse fts", err)
		}
	}
	return st, nil
}

func chunkerVersionFor(rel string) string {
	switch filepath.Ext(rel) {
	case ".md", ".markdown":
		return "markdown-section-v1"
	default:
		return "paragraph-v1"
	}
}

func sourceIDFromPathKey(pathKey string) ids.SourceID {
	if len(pathKey) >= 16 {
		return ids.SourceID(pathKey[:16])
	}
	return ids.SourceID(pathKey)
}

func sanitizeID(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' {
			b = append(b, c)
		} else {
			b = append(b, '_')
		}
	}
	return string(b)
}
