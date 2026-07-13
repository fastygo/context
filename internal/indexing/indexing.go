// Package indexing defines IndexSnapshot, manifest, and index handle types.
package indexing

import (
	"fmt"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
)

// IndexSnapshot is an immutable index generation (ADR-0008, ADR-0021).
type IndexSnapshot struct {
	ID                 ids.SnapshotID
	ProjectID          ids.ProjectID
	ParentSnapshotID   ids.SnapshotID // empty for root
	Status             foundation.SnapshotStatus
	SourceMerkleRoot   foundation.ChecksumHex
	ChunkSetHash       foundation.ChecksumHex
	SourceMerkleAlgo   string
	ChunkSetMerkleAlgo string
	ParserVersion      string
	ChunkerVersion     string
	EmbedModelVersion  string
	MorphVersion       string
	SparseIndexRef     SparseIndexRef
	VectorNamespace    VectorNamespace
	DenseEnabled       bool
	SparseEnabled      bool
	FailureReason      string
}

func (s IndexSnapshot) Validate() error {
	if err := s.ID.Validate(); err != nil {
		return err
	}
	if err := s.ProjectID.Validate(); err != nil {
		return err
	}
	if err := s.Status.Validate(); err != nil {
		return err
	}
	if s.Status == foundation.SnapshotReady {
		if err := s.SourceMerkleRoot.Validate(); err != nil {
			return fmt.Errorf("ready snapshot source_merkle_root: %w", err)
		}
		if err := s.ChunkSetHash.Validate(); err != nil {
			return fmt.Errorf("ready snapshot chunk_set_hash: %w", err)
		}
		if s.ParserVersion == "" || s.ChunkerVersion == "" {
			return fmt.Errorf("ready snapshot: parser/chunker versions required")
		}
	}
	if s.Status == foundation.SnapshotFailed && s.FailureReason == "" {
		return fmt.Errorf("failed snapshot: failure_reason required")
	}
	return nil
}

// ManifestNode is a Merkle-style source-tree node.
type ManifestNode struct {
	PathKey    string
	NodeHash   foundation.ChecksumHex
	SourceID   ids.SourceID // empty for internal nodes
	ChildKeys  []string     // sorted child path keys
	IsLeaf     bool
}

func (n ManifestNode) Validate() error {
	if n.PathKey == "" {
		return fmt.Errorf("manifest path_key: empty")
	}
	return n.NodeHash.Validate()
}

// ChunkAlias maps a stable alias to a chunk within a snapshot.
type ChunkAlias struct {
	ProjectID  ids.ProjectID
	SnapshotID ids.SnapshotID
	Alias      string
	ChunkID    ids.ChunkID
}

func (a ChunkAlias) Validate() error {
	if err := a.ProjectID.Validate(); err != nil {
		return err
	}
	if err := a.SnapshotID.Validate(); err != nil {
		return err
	}
	if a.Alias == "" {
		return fmt.Errorf("chunk alias: empty")
	}
	return a.ChunkID.Validate()
}

// ContextRef is a short model-visible source reference (ADR-0013).
type ContextRef struct {
	ID         ids.ContextRefID
	ProjectID  ids.ProjectID
	SnapshotID ids.SnapshotID
	ChunkID    ids.ChunkID
	Label      string
}

func (r ContextRef) Validate() error {
	if err := r.ID.Validate(); err != nil {
		return err
	}
	if err := r.ProjectID.Validate(); err != nil {
		return err
	}
	if err := r.SnapshotID.Validate(); err != nil {
		return err
	}
	if err := r.ChunkID.Validate(); err != nil {
		return err
	}
	if r.Label == "" {
		return fmt.Errorf("context_ref label: empty")
	}
	return nil
}

// PathAlias maps a logical path without exposing host filesystem layout.
type PathAlias struct {
	ID        ids.PathAliasID
	ProjectID ids.ProjectID
	PathKey   string
	Alias     string
}

func (p PathAlias) Validate() error {
	if err := p.ID.Validate(); err != nil {
		return err
	}
	if err := p.ProjectID.Validate(); err != nil {
		return err
	}
	if p.PathKey == "" || p.Alias == "" {
		return fmt.Errorf("path_alias: path_key and alias required")
	}
	return nil
}

// VectorNamespace identifies a dense vector partition handle (ADR-0004).
type VectorNamespace struct {
	Name             string
	ProjectID        ids.ProjectID
	SnapshotID       ids.SnapshotID
	EmbeddingVersion string
}

func (n VectorNamespace) Validate() error {
	if n.Name == "" {
		return fmt.Errorf("vector_namespace name: empty")
	}
	if err := n.ProjectID.Validate(); err != nil {
		return err
	}
	return n.SnapshotID.Validate()
}

// SparseIndexRef points at a sparse/FTS index generation (ADR-0009 / ADR-0017).
type SparseIndexRef struct {
	URI        string
	ProjectID  ids.ProjectID
	SnapshotID ids.SnapshotID
	BundleHash foundation.ChecksumHex // optional until sealed
	Version    string
}

func (r SparseIndexRef) Validate() error {
	if err := r.ProjectID.Validate(); err != nil {
		return err
	}
	return r.SnapshotID.Validate()
}
