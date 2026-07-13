package artifacts

import (
	"fmt"
	"strings"
	"time"

	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/ids"
)

// ArtifactLineage records durable many-to-many derivation provenance.
// Artifact.SourceID remains the optional immediate source origin.
type ArtifactLineage struct {
	ProjectID          ids.ProjectID      `json:"project_id"`
	OutputArtifactID   ids.ArtifactID     `json:"output_artifact_id"`
	InputArtifactIDs   []ids.ArtifactID   `json:"input_artifact_ids,omitempty"`
	SourceRefs         []corpus.SourceRef `json:"source_refs,omitempty"`
	ContextPackID      ids.PackID         `json:"context_pack_id,omitempty"`
	AgentRunID         ids.RunID          `json:"agent_run_id,omitempty"`
	ToolCallID         ids.ToolCallID     `json:"tool_call_id,omitempty"`
	GeneratorID        string             `json:"generator_id"`
	GeneratorVersion   string             `json:"generator_version"`
	TransformationKind string             `json:"transformation_kind"`
	CreatedAt          time.Time          `json:"created_at"`
}

func (l ArtifactLineage) Validate() error {
	if err := l.ProjectID.Validate(); err != nil {
		return err
	}
	if err := l.OutputArtifactID.Validate(); err != nil {
		return err
	}
	if len(l.InputArtifactIDs) == 0 && len(l.SourceRefs) == 0 {
		return fmt.Errorf("artifact_lineage: at least one input artifact or source ref required")
	}
	seen := make(map[ids.ArtifactID]struct{}, len(l.InputArtifactIDs))
	for _, inputID := range l.InputArtifactIDs {
		if err := inputID.Validate(); err != nil {
			return err
		}
		if inputID == l.OutputArtifactID {
			return fmt.Errorf("artifact_lineage: output artifact cannot be an input")
		}
		if _, exists := seen[inputID]; exists {
			return fmt.Errorf("artifact_lineage: duplicate input artifact %q", inputID)
		}
		seen[inputID] = struct{}{}
	}
	seenRefs := make(map[corpus.SourceRef]struct{}, len(l.SourceRefs))
	for _, ref := range l.SourceRefs {
		if err := ref.Validate(); err != nil {
			return fmt.Errorf("artifact_lineage source_ref: %w", err)
		}
		if ref.ProjectID != l.ProjectID {
			return fmt.Errorf("artifact_lineage: source_ref project does not match")
		}
		if _, exists := seenRefs[ref]; exists {
			return fmt.Errorf("artifact_lineage: duplicate source_ref for source %q", ref.SourceID)
		}
		seenRefs[ref] = struct{}{}
	}
	if strings.TrimSpace(l.GeneratorID) == "" {
		return fmt.Errorf("artifact_lineage generator_id: empty")
	}
	if strings.TrimSpace(l.GeneratorVersion) == "" {
		return fmt.Errorf("artifact_lineage generator_version: empty")
	}
	if strings.TrimSpace(l.TransformationKind) == "" {
		return fmt.Errorf("artifact_lineage transformation_kind: empty")
	}
	if l.CreatedAt.IsZero() {
		return fmt.Errorf("artifact_lineage created_at: zero")
	}
	return nil
}
