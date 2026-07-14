// Package corpus defines project-scoped source and chunk domain models.
package corpus

import (
	"fmt"
	"time"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
)

// Project is an isolated index/workspace boundary (ADR-0025).
// Optional TenantID is the outer ACL/quota/billing root when multi-tenant
// hosting is enabled; empty TenantID is valid for single-tenant local serve.
type Project struct {
	ID               ids.ProjectID
	Name             string
	TenantID         ids.TenantID  `json:"tenant_id,omitempty"`
	ActiveSnapshotID ids.SnapshotID // empty until first ready snapshot
}

func (p Project) Validate() error {
	if err := p.ID.Validate(); err != nil {
		return err
	}
	if p.Name == "" {
		return fmt.Errorf("project name: empty")
	}
	return nil
}

// SourceType classifies how a source entered the corpus.
type SourceType string

const (
	SourceTypeFile     SourceType = "file"
	SourceTypeArtifact SourceType = "artifact"
	SourceTypeURL      SourceType = "url"
	SourceTypeTool     SourceType = "tool_output"
	SourceTypeSpec     SourceType = "spec"
)

// Source is a registered project source.
type Source struct {
	ID               ids.SourceID
	ProjectID        ids.ProjectID
	Type             SourceType
	PathKey          string // stable logical key; not a host filesystem path
	URI              string // adapter-specific locator; may be empty
	TrustLevel       foundation.TrustLevel
	MediaType        string
	Checksum         foundation.ChecksumHex // of original artifact bytes when available
	TemporalMetadata *TemporalMetadata      // optional source-domain time; never runtime trace time
	// TombstonedAt marks soft-delete (stabilization C1). Nil means live.
	// Tombstoned sources must not contribute chunks to search or new packs.
	TombstonedAt *time.Time `json:"tombstoned_at,omitempty"`
}

// IsTombstoned reports whether the source is soft-deleted.
func (s Source) IsTombstoned() bool {
	return s.TombstonedAt != nil
}

func (s Source) Validate() error {
	if err := s.ID.Validate(); err != nil {
		return err
	}
	if err := s.ProjectID.Validate(); err != nil {
		return err
	}
	if s.Type == "" {
		return fmt.Errorf("source type: empty")
	}
	if s.PathKey == "" {
		return fmt.Errorf("path_key: empty")
	}
	if err := s.TrustLevel.Validate(); err != nil {
		return err
	}
	if s.TemporalMetadata != nil {
		return s.TemporalMetadata.Validate()
	}
	return nil
}

// SourceRef points at a source span for citations and evidence.
type SourceRef struct {
	ProjectID  ids.ProjectID          `json:"project_id"`
	SourceID   ids.SourceID           `json:"source_id"`
	ChunkID    ids.ChunkID            `json:"chunk_id,omitempty"`
	Span       foundation.ByteSpan    `json:"span"`
	Checksum   foundation.ChecksumHex `json:"checksum"`
	ContextRef ids.ContextRefID       `json:"context_ref,omitempty"`
}

func (r SourceRef) Validate() error {
	if err := r.ProjectID.Validate(); err != nil {
		return err
	}
	if err := r.SourceID.Validate(); err != nil {
		return err
	}
	if err := r.Span.Validate(); err != nil {
		return err
	}
	return r.Checksum.Validate()
}

// Chunk is an indexed source span with provenance.
type Chunk struct {
	ID               ids.ChunkID
	ProjectID        ids.ProjectID
	SourceID         ids.SourceID
	ArtifactID       ids.ArtifactID
	SnapshotID       ids.SnapshotID
	ChunkerVersion   string
	Span             foundation.ByteSpan
	TextChecksum     foundation.ChecksumHex
	ChunkHash        foundation.ChecksumHex
	Language           string // BCP 47; empty allowed until language adapter runs
	EmbeddingVersion   string
	SparseVersion      string
	MorphVersion       string // analyzer / morph adapter pin (ADR-0011)
	DictionaryVersion  string // lexicon resource pin when present
	TemporalMetadata   *TemporalMetadata // optional source-domain time; never runtime trace time
}

func (c Chunk) Validate() error {
	if err := c.ID.Validate(); err != nil {
		return err
	}
	if err := c.ProjectID.Validate(); err != nil {
		return err
	}
	if err := c.SourceID.Validate(); err != nil {
		return err
	}
	if err := c.ArtifactID.Validate(); err != nil {
		return err
	}
	if c.ChunkerVersion == "" {
		return fmt.Errorf("chunker_version: empty")
	}
	if err := c.Span.Validate(); err != nil {
		return err
	}
	if err := c.TextChecksum.Validate(); err != nil {
		return err
	}
	if err := c.ChunkHash.Validate(); err != nil {
		return err
	}
	if c.TemporalMetadata != nil {
		return c.TemporalMetadata.Validate()
	}
	return nil
}
