// Package artifacts defines stored artifact metadata and the ArtifactStore port.
package artifacts

import (
	"context"
	"fmt"
	"strings"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
)

// Closed artifact_type vocabulary (ADR-0022). Empty type normalizes to TypeBlob on Put.
const (
	TypeBlob       = "blob"
	TypeSpill      = "spill"
	TypeToolOutput = "tool_output"
	TypeStructured = "structured"
)

// Artifact is stored source material or generated intermediate output.
type Artifact struct {
	ID           ids.ArtifactID
	ProjectID    ids.ProjectID
	SourceID     ids.SourceID // optional for generated outputs
	MediaType    string
	ByteSize     int64
	Checksum     foundation.ChecksumHex
	StorageURI   string // adapter-specific; never an unscoped host path in indexes
	ArtifactType string // TypeBlob | TypeSpill | TypeToolOutput | TypeStructured
	SchemaID     string // required when ArtifactType == TypeStructured (ADR-0022)
}

// PutOptions carries optional metadata on ArtifactStore.Put (ADR-0022).
type PutOptions struct {
	ArtifactType string
	SchemaID     string
	SourceID     ids.SourceID
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
	if err := ValidateTypeAndSchema(a.ArtifactType, a.SchemaID); err != nil {
		return err
	}
	return nil
}

// NormalizeType returns a canonical artifact_type (empty → TypeBlob).
func NormalizeType(artifactType string) string {
	t := strings.TrimSpace(artifactType)
	if t == "" {
		return TypeBlob
	}
	return t
}

// ValidateTypeAndSchema enforces ADR-0022 invariants.
func ValidateTypeAndSchema(artifactType, schemaID string) error {
	t := NormalizeType(artifactType)
	switch t {
	case TypeBlob, TypeSpill, TypeToolOutput, TypeStructured:
		// ok
	default:
		return fmt.Errorf("artifact_type: unsupported %q", t)
	}
	sid := strings.TrimSpace(schemaID)
	if t == TypeStructured {
		if sid == "" {
			return fmt.Errorf("schema_id: required for artifact_type %s", TypeStructured)
		}
		if strings.ContainsAny(sid, " \t\n\r") {
			return fmt.Errorf("schema_id: must not contain whitespace")
		}
	}
	if sid != "" && t != TypeStructured {
		return fmt.Errorf("schema_id: only allowed when artifact_type is %s", TypeStructured)
	}
	return nil
}

// ApplyPutOptions merges opts into a base artifact before Validate.
func ApplyPutOptions(art Artifact, opts *PutOptions) Artifact {
	if opts == nil {
		art.ArtifactType = NormalizeType(art.ArtifactType)
		return art
	}
	if opts.SourceID != "" {
		art.SourceID = opts.SourceID
	}
	art.SchemaID = strings.TrimSpace(opts.SchemaID)
	art.ArtifactType = NormalizeType(opts.ArtifactType)
	if art.SchemaID != "" && art.ArtifactType == TypeBlob {
		art.ArtifactType = TypeStructured
	}
	return art
}

// ArtifactStore is the replaceable blob/content port (localfs first per ADR-0003).
type ArtifactStore interface {
	Put(ctx context.Context, projectID ids.ProjectID, artifactID ids.ArtifactID, mediaType string, body []byte, opts *PutOptions) (Artifact, error)
	Get(ctx context.Context, projectID ids.ProjectID, artifactID ids.ArtifactID) (Artifact, []byte, error)
	Delete(ctx context.Context, projectID ids.ProjectID, artifactID ids.ArtifactID) error
	// DeleteProject removes all artifacts for a project (stabilization C7).
	DeleteProject(ctx context.Context, projectID ids.ProjectID) error
}
