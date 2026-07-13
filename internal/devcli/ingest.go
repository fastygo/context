package devcli

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/artifacts/localfs"
	"github.com/fastygo/context/internal/config"
	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing"
	"github.com/fastygo/context/internal/indexing/commit"
	"github.com/fastygo/context/internal/indexing/hashing"
	"github.com/fastygo/context/internal/indexing/morph"
	"github.com/fastygo/context/internal/indexing/pipeline"
	"github.com/fastygo/context/internal/indexing/source"
	"github.com/fastygo/context/internal/retrieval/dense"
)

// Ingest indexes files under path into the workspace snapshot.
// When CONTEXT_ENABLE_DENSE=1, dense vectors are upserted before the snapshot
// becomes active (ADR-0021). Version pins are always written on chunks.
func Ingest(dataDir, projectID, path string) (State, error) {
	if err := requireQuotaResource(dataDir, "chunks"); err != nil {
		return State{}, err
	}
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

	cfg, err := config.LoadStorageConfigFromEnv()
	if err != nil {
		return State{}, err
	}

	arts, err := localfs.New(ws.ArtifactsDir())
	if err != nil {
		return State{}, err
	}
	runner := pipeline.NewDefault(source.LocalFiles{})
	snapID := ids.SnapshotID(fmt.Sprintf("snap_%d", len(st.Chunks)+1))
	ctx := context.Background()
	res, err := runner.Run(ctx, st.Project.ID, snapID, absRoot, nil)
	if err != nil {
		return State{}, err
	}

	morphVer := res.MorphVersion
	if morphVer.AnalyzerVersion == "" {
		morphVer = morph.Hook{}.Version()
	}
	embVer := cfg.Vector.EmbeddingVersion
	if embVer == "" {
		embVer = config.DefaultEmbeddingVersion
	}
	sparseVer := sparseVersionFor(cfg.Sparse.Kind)
	denseOn := denseEnabledByEnv()
	sparseOn := cfg.Sparse.Kind == config.StoreKindPostgresFTS

	chunks := make([]IndexedChunk, 0)
	for _, leaf := range res.Leaves {
		raws := res.RawChunks[leaf.PathKey]
		version := chunkerVersionFor(leaf.RelativePath)
		sourceID := sourceIDFromPathKey(leaf.PathKey)
		artID := ids.ArtifactID("src_" + sanitizeID(string(sourceID)))
		if _, err := arts.Put(ctx, st.Project.ID, artID, "application/octet-stream", []byte(leaf.RelativePath), nil); err != nil {
			_ = err
		}
		for _, rc := range raws {
			chHash := hashing.ChunkHash(version, leaf.PathKey, rc.Span.Start, rc.Span.End, rc.Text)
			chunkID := commit.StableChunkID(st.Project.ID, chHash)
			chunks = append(chunks, IndexedChunk{
				ChunkID:           chunkID,
				SourceID:          sourceID,
				SnapshotID:        res.Snapshot.ID,
				PathKey:           leaf.PathKey,
				RelativePath:      leaf.RelativePath,
				SpanStart:         rc.Span.Start,
				SpanEnd:           rc.Span.End,
				Text:              rc.Text,
				TextChecksum:      rc.TextChecksum,
				ChunkHash:         chHash,
				TrustLevel:        foundation.TrustProject,
				ChunkerVersion:    version,
				EmbeddingVersion:  embVer,
				MorphVersion:      morphVer.AnalyzerVersion,
				DictionaryVersion: string(morphVer.DictionaryVersion),
				SparseVersion:     sparseVer,
			})
		}
	}

	snap := annotateSnapshot(res.Snapshot, cfg, embVer, morphVer.AdapterVersion, denseOn, sparseOn)

	prevActive := st.Project.ActiveSnapshotID
	prevChunks := st.Chunks

	if denseOn {
		if err := commitDenseVectors(ctx, cfg, st.Project.ID, snap.ID, chunks); err != nil {
			failed, ferr := (commit.Builder{}).Fail(snap, "dense_write_failed")
			if ferr != nil {
				return State{}, apperr.Wrap(apperr.Internal, "dense write failed", err)
			}
			st.Snapshot = failed
			st.Project.ActiveSnapshotID = prevActive
			st.Chunks = prevChunks
			st.CorpusRoot = absRoot
			st.LastFailed = &FailedAttempt{Snapshot: failed, Chunks: chunks}
			_ = ws.Save(st)
			_ = recordFailedSnapshot(ctx, st.Project, failed)
			return st, apperr.Wrap(apperr.Internal, "dense write failed", err)
		}
	}

	sparseH, err := OpenSparse(ctx, nil)
	if err != nil {
		return State{}, err
	}
	defer sparseH.Closer()
	if sparseH.UsesFTS {
		if err := sparseH.UpsertChunks(ctx, st.Project.ID, chunks); err != nil {
			failed, ferr := (commit.Builder{}).Fail(snap, "sparse_write_failed")
			if ferr != nil {
				return State{}, apperr.Wrap(apperr.Internal, "sparse fts write failed", err)
			}
			st.Snapshot = failed
			st.Project.ActiveSnapshotID = prevActive
			st.Chunks = prevChunks
			st.CorpusRoot = absRoot
			st.LastFailed = &FailedAttempt{Snapshot: failed, Chunks: chunks}
			_ = ws.Save(st)
			_ = recordFailedSnapshot(ctx, st.Project, failed)
			return st, apperr.Wrap(apperr.Internal, "sparse fts write failed", err)
		}
		snap.SparseEnabled = true
	}

	st.Snapshot = snap
	st.Project.ActiveSnapshotID = snap.ID
	st.Chunks = chunks
	st.CorpusRoot = absRoot
	st.LastFailed = nil
	if err := ws.Save(st); err != nil {
		return State{}, err
	}

	handle, err := OpenMetadata(ctx)
	if err != nil {
		return State{}, err
	}
	defer handle.Close()
	if handle.UsesPostgres() {
		if err := PersistIngest(ctx, handle.Store, st.Project, snap, res.Leaves, chunks); err != nil {
			return State{}, apperr.Wrap(apperr.Internal, "persist ingest metadata", err)
		}
	}
	return st, nil
}

func annotateSnapshot(
	snap indexing.IndexSnapshot,
	cfg config.StorageConfig,
	embVer, morphAdapterVer string,
	denseOn, sparseOn bool,
) indexing.IndexSnapshot {
	snap.EmbedModelVersion = embVer
	if morphAdapterVer != "" {
		snap.MorphVersion = morphAdapterVer
	}
	snap.DenseEnabled = denseOn
	snap.SparseEnabled = sparseOn
	snap.VectorNamespace = indexing.VectorNamespace{
		Name:             cfg.Vector.Collection,
		ProjectID:        snap.ProjectID,
		SnapshotID:       snap.ID,
		EmbeddingVersion: embVer,
	}
	return snap
}

func commitDenseVectors(ctx context.Context, cfg config.StorageConfig, projectID ids.ProjectID, snapshotID ids.SnapshotID, chunks []IndexedChunk) error {
	store, ns, emb, err := openDenseStore(ctx)
	if err != nil {
		return err
	}
	defer store.Close()
	ns.ProjectID = projectID
	ns.SnapshotID = snapshotID
	docs := make([]dense.ChunkDoc, 0, len(chunks))
	for _, ch := range chunks {
		docs = append(docs, dense.ChunkDoc{
			ProjectID:        projectID,
			SnapshotID:       snapshotID,
			ChunkID:          ch.ChunkID,
			Text:             ch.Text,
			Language:         ch.Language,
			ChunkerVersion:   ch.ChunkerVersion,
			EmbeddingVersion: firstNonEmpty(ch.EmbeddingVersion, ns.EmbeddingVersion, cfg.Vector.EmbeddingVersion),
			MorphVersion:     ch.MorphVersion,
			Span:             foundation.ByteSpan{Start: ch.SpanStart, End: ch.SpanEnd},
		})
	}
	return dense.UpsertEmbedded(ctx, store, emb, ns, docs)
}

func recordFailedSnapshot(ctx context.Context, project corpus.Project, snap indexing.IndexSnapshot) error {
	handle, err := OpenMetadata(ctx)
	if err != nil {
		return err
	}
	defer handle.Close()
	if !handle.UsesPostgres() {
		return nil
	}
	// Record failed snapshot without flipping active pointer (ADR-0021).
	if err := handle.Store.PutProject(ctx, project); err != nil {
		return err
	}
	return handle.Store.PutSnapshot(ctx, snap)
}

func sparseVersionFor(kind config.StoreKind) string {
	switch kind {
	case config.StoreKindPostgresFTS:
		return "postgres_fts"
	case config.StoreKindContextSparse:
		return "context_sparse"
	default:
		return "fake"
	}
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
