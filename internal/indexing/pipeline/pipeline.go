// Package pipeline runs discover → parse → chunk → manifest → snapshot seal.
package pipeline

import (
	"context"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing"
	"github.com/fastygo/context/internal/indexing/chunk"
	"github.com/fastygo/context/internal/indexing/commit"
	"github.com/fastygo/context/internal/indexing/hashing"
	"github.com/fastygo/context/internal/indexing/manifest"
	"github.com/fastygo/context/internal/indexing/morph"
	"github.com/fastygo/context/internal/indexing/parse"
	"github.com/fastygo/context/internal/indexing/source"
	"github.com/fastygo/context/internal/indexing/token"
	"github.com/fastygo/context/internal/linguistic"
)

// Result is one indexing pass output for tests and later storage writers.
type Result struct {
	Snapshot     indexing.IndexSnapshot
	Leaves       []manifest.SourceLeaf
	ChunkHashes  []foundation.ChecksumHex
	RawChunks    map[string][]chunk.RawChunk // path_key → chunks
	Tokens       []linguistic.TokenOccurrence
	SourceDiff   []manifest.SourceChange
	MorphVersion linguistic.AnalyzerVersion
}

// Runner executes a deterministic local indexing pass.
type Runner struct {
	Sources  source.Adapter
	Parsers  parse.Registry
	Chunkers struct {
		Plain    chunk.Paragraph
		Markdown chunk.MarkdownSection
	}
	Enricher chunk.Enricher
	Manifest manifest.Builder
	Commit   commit.Builder
	Morph    morph.Hook
}

// NewDefault returns a Runner with PoC defaults.
func NewDefault(src source.Adapter) Runner {
	return Runner{
		Sources:  src,
		Parsers:  parse.Registry{},
		Enricher: chunk.NoopEnricher{},
		Manifest: manifest.Builder{},
		Commit:   commit.Builder{Manifest: manifest.Builder{}},
		Morph:    morph.Hook{},
	}
}

// Run indexes sources under root into a sealed ready snapshot.
func (r Runner) Run(ctx context.Context, projectID ids.ProjectID, snapshotID ids.SnapshotID, root string, prevLeaves map[string]foundation.ChecksumHex) (Result, error) {
	discovered, err := r.Sources.List(ctx, projectID, root)
	if err != nil {
		return Result{}, err
	}
	versions := commit.Versions{
		ParserVersion:  "mixed",
		ChunkerVersion: "mixed",
		MorphVersion:   r.Morph.Version().AdapterVersion,
	}
	building, err := r.Commit.Building(commit.Input{
		SnapshotID: snapshotID,
		ProjectID:  projectID,
		Versions:   versions,
	})
	if err != nil {
		return Result{}, err
	}

	leaves := make([]manifest.SourceLeaf, 0, len(discovered))
	leafMap := make(map[string]foundation.ChecksumHex, len(discovered))
	rawByPath := make(map[string][]chunk.RawChunk)
	var chunkHashes []foundation.ChecksumHex
	var tokens []linguistic.TokenOccurrence
	parserVersions := map[string]bool{}
	chunkerVersions := map[string]bool{}

	for _, d := range discovered {
		pathKey := hashing.PathKey(projectID, d.RelativePath)
		leaf := r.Manifest.Leaf(pathKey, d.RelativePath, string(d.SourceType), d.Bytes)
		leaves = append(leaves, leaf)
		leafMap[pathKey] = leaf.LeafHash

		parser := r.Parsers.For(d.MediaType)
		doc, err := parser.Parse(ctx, d.MediaType, d.Bytes)
		if err != nil {
			failed, _ := r.Commit.Fail(building, "validation_error")
			_ = failed
			return Result{}, err
		}
		parserVersions[doc.ParserVersion] = true

		var chunker chunk.Chunker = r.Chunkers.Plain
		if d.MediaType == "text/markdown" {
			chunker = r.Chunkers.Markdown
		}
		chunkerVersions[chunker.Version()] = true
		raw, err := chunker.Chunk(ctx, doc)
		if err != nil {
			return Result{}, err
		}
		if r.Enricher != nil {
			raw, err = r.Enricher.Enrich(ctx, doc, raw)
			if err != nil {
				return Result{}, err
			}
		}
		for i, rc := range raw {
			chHash := hashing.ChunkHash(chunker.Version(), pathKey, rc.Span.Start, rc.Span.End, rc.Text)
			chunkHashes = append(chunkHashes, chHash)
			chunkID := commit.StableChunkID(projectID, chHash)
			toks := token.Capture(projectID, ids.SourceID(pathKey[:16]), chunkID, "und", rc)
			tokens = append(tokens, toks...)
			_ = i
		}
		rawByPath[pathKey] = raw
	}

	if len(parserVersions) == 1 {
		for v := range parserVersions {
			building.ParserVersion = v
		}
	}
	if len(chunkerVersions) == 1 {
		for v := range chunkerVersions {
			building.ChunkerVersion = v
		}
	}

	ready, err := r.Commit.SealReady(building, leaves, chunkHashes)
	if err != nil {
		return Result{}, err
	}
	diff := manifest.DiffSources(prevLeaves, leafMap)
	return Result{
		Snapshot:     ready,
		Leaves:       leaves,
		ChunkHashes:  chunkHashes,
		RawChunks:    rawByPath,
		Tokens:       tokens,
		SourceDiff:   diff,
		MorphVersion: r.Morph.Version(),
	}, nil
}

// Ensure non-empty discovery for callers that need at least one file.
func RequireSources(n int) error {
	if n == 0 {
		return apperr.New(apperr.Validation, "no indexable sources discovered")
	}
	return nil
}
