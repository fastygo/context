// Package artifacts defines stored artifact metadata and the ArtifactStore port.
package artifacts

import (
	"context"
	"fmt"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
)

// Artifact is stored source material or generated intermediate output.
type Artifact struct {
	ID          ids.ArtifactID
	ProjectID   ids.ProjectID
	SourceID    ids.SourceID // optional for generated outputs
	MediaType   string
	ByteSize    int64
	Checksum    foundation.ChecksumHex
	StorageURI  string // adapter-specific; never an unscoped host path in indexes
	ArtifactType string
}

func (a Artifact) Validate() error {
	if err := a.ID.Validate(); err != nil {
		return err
	}
	if err := a.ProjectID.Validate(); err != nil {
		return err
	}
	if a.MediaType == "" {
		return fmt.Errorf("media_type: empty")
	}
	if a.ByteSize < 0 {
		return fmt.Errorf("byte_size: negative")
	}
	if err := a.Checksum.Validate(); err != nil {
		return err
	}
	if a.StorageURI == "" {
		return fmt.Errorf("storage_uri: empty")
	}
	return nil
}

// ArtifactStore is the replaceable blob/content port (localfs first per ADR-0003).
type ArtifactStore interface {
	Put(ctx context.Context, projectID ids.ProjectID, artifactID ids.ArtifactID, mediaType string, body []byte) (Artifact, error)
	Get(ctx context.Context, projectID ids.ProjectID, artifactID ids.ArtifactID) (Artifact, []byte, error)
	Delete(ctx context.Context, projectID ids.ProjectID, artifactID ids.ArtifactID) error
}
