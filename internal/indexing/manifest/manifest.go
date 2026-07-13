// Package manifest builds source leaves and diffs manifests between snapshots.
package manifest

import (
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/indexing/hashing"
)

// SourceLeaf is one Level-A manifest entry.
type SourceLeaf struct {
	PathKey      string
	RelativePath string
	SourceType   string
	LeafHash     foundation.ChecksumHex
	ArtifactHash foundation.ChecksumHex
}

// Builder constructs source leaves and Merkle roots.
type Builder struct{}

func (Builder) Leaf(pathKey, relativePath, sourceType string, original []byte) SourceLeaf {
	return SourceLeaf{
		PathKey:      pathKey,
		RelativePath: relativePath,
		SourceType:   sourceType,
		LeafHash:     hashing.SourceLeafHash(pathKey, sourceType, original),
		ArtifactHash: hashing.ArtifactChecksum(original),
	}
}

func (Builder) SourceRoot(leaves []SourceLeaf) foundation.ChecksumHex {
	m := make(map[string]foundation.ChecksumHex, len(leaves))
	for _, leaf := range leaves {
		m[leaf.PathKey] = leaf.LeafHash
	}
	return hashing.SourceMerkleRoot(m)
}

func (Builder) ChunkRoot(chunkHashes []foundation.ChecksumHex) foundation.ChecksumHex {
	return hashing.ChunkSetHash(chunkHashes)
}

// ChangeKind classifies manifest diff rows.
type ChangeKind string

const (
	Added     ChangeKind = "added"
	Removed   ChangeKind = "removed"
	Changed   ChangeKind = "changed"
	Unchanged ChangeKind = "unchanged"
)

// SourceChange is one source-level diff entry.
type SourceChange struct {
	PathKey    string
	Kind       ChangeKind
	OldHash    foundation.ChecksumHex
	NewHash    foundation.ChecksumHex
}

// ChunkChange is one chunk-level diff entry.
type ChunkChange struct {
	ChunkHash foundation.ChecksumHex
	Kind      ChangeKind
}

// DiffSources compares previous and next source leaf maps keyed by path_key.
func DiffSources(prev, next map[string]foundation.ChecksumHex) []SourceChange {
	out := make([]SourceChange, 0)
	seen := map[string]bool{}
	for k, nh := range next {
		seen[k] = true
		oh, ok := prev[k]
		switch {
		case !ok:
			out = append(out, SourceChange{PathKey: k, Kind: Added, NewHash: nh})
		case oh != nh:
			out = append(out, SourceChange{PathKey: k, Kind: Changed, OldHash: oh, NewHash: nh})
		default:
			out = append(out, SourceChange{PathKey: k, Kind: Unchanged, OldHash: oh, NewHash: nh})
		}
	}
	for k, oh := range prev {
		if seen[k] {
			continue
		}
		out = append(out, SourceChange{PathKey: k, Kind: Removed, OldHash: oh})
	}
	return out
}

// DiffChunks compares previous and next chunk hash sets.
func DiffChunks(prev, next []foundation.ChecksumHex) []ChunkChange {
	prevSet := map[foundation.ChecksumHex]bool{}
	for _, h := range prev {
		prevSet[h] = true
	}
	nextSet := map[foundation.ChecksumHex]bool{}
	for _, h := range next {
		nextSet[h] = true
	}
	out := make([]ChunkChange, 0)
	for h := range nextSet {
		if prevSet[h] {
			out = append(out, ChunkChange{ChunkHash: h, Kind: Unchanged})
		} else {
			out = append(out, ChunkChange{ChunkHash: h, Kind: Added})
		}
	}
	for h := range prevSet {
		if !nextSet[h] {
			out = append(out, ChunkChange{ChunkHash: h, Kind: Removed})
		}
	}
	return out
}

// ChangedPathKeys returns path keys that are added or changed (not unchanged/removed).
func ChangedPathKeys(changes []SourceChange) []string {
	var out []string
	for _, c := range changes {
		if c.Kind == Added || c.Kind == Changed {
			out = append(out, c.PathKey)
		}
	}
	return out
}
