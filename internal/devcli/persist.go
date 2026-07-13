package devcli

import (
	"context"
	"fmt"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/artifacts"
	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing"
	"github.com/fastygo/context/internal/indexing/manifest"
	"github.com/fastygo/context/internal/storage"
)

// PersistIngest writes project/sources/chunks/snapshot(/artifact meta) to meta.
// When meta implements TxRunner, snapshot activation runs in one transaction.
func PersistIngest(ctx context.Context, meta storage.MetadataStore, project corpus.Project, snap indexing.IndexSnapshot, leaves []manifest.SourceLeaf, chunks []IndexedChunk) error {
	if meta == nil {
		return apperr.New(apperr.Validation, "metadata store required")
	}
	write := func(ctx context.Context) error {
		if err := meta.PutProject(ctx, project); err != nil {
			return err
		}
		seenSrc := map[ids.SourceID]struct{}{}
		seenArt := map[ids.ArtifactID]struct{}{}
		for _, leaf := range leaves {
			sourceID := sourceIDFromPathKey(leaf.PathKey)
			if _, ok := seenSrc[sourceID]; !ok {
				seenSrc[sourceID] = struct{}{}
				src := corpus.Source{
					ID:         sourceID,
					ProjectID:  project.ID,
					Type:       corpus.SourceTypeFile,
					PathKey:    leaf.PathKey,
					TrustLevel: foundation.TrustProject,
					MediaType:  "application/octet-stream",
					Checksum:   leaf.ArtifactHash,
				}
				if err := meta.PutSource(ctx, src); err != nil {
					return err
				}
			}
			artID := ids.ArtifactID("src_" + sanitizeID(string(sourceID)))
			if _, ok := seenArt[artID]; !ok {
				seenArt[artID] = struct{}{}
				if am, ok := meta.(storage.ArtifactMetaStore); ok {
					sum := leaf.ArtifactHash
					if sum == "" {
						sum = leaf.LeafHash
					}
					art := artifacts.Artifact{
						ID:           artID,
						ProjectID:    project.ID,
						SourceID:     sourceID,
						MediaType:    "application/octet-stream",
						ByteSize:     0,
						Checksum:     sum,
						StorageURI:   fmt.Sprintf("localfs://%s", leaf.RelativePath),
						ArtifactType: artifacts.TypeBlob,
					}
					if err := am.PutArtifactMeta(ctx, art); err != nil {
						return err
					}
				}
			}
		}
		for _, ch := range chunks {
			artID := ids.ArtifactID("src_" + sanitizeID(string(ch.SourceID)))
			domain := corpus.Chunk{
				ID:             ch.ChunkID,
				ProjectID:      project.ID,
				SourceID:       ch.SourceID,
				ArtifactID:     artID,
				SnapshotID:     ch.SnapshotID,
				ChunkerVersion: chunkerVersionFor(ch.RelativePath),
				Span:           foundation.ByteSpan{Start: ch.SpanStart, End: ch.SpanEnd},
				TextChecksum:   ch.TextChecksum,
				ChunkHash:      ch.ChunkHash,
			}
			if err := meta.PutChunk(ctx, domain); err != nil {
				return err
			}
		}
		if err := meta.PutSnapshot(ctx, snap); err != nil {
			return err
		}
		return meta.SetActiveSnapshot(ctx, project.ID, snap.ID)
	}

	if tx, ok := meta.(storage.TxRunner); ok {
		return tx.WithTx(ctx, write)
	}
	return write(ctx)
}
