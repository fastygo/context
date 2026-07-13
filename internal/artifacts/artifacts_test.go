package artifacts_test

import (
	"testing"
	"time"

	"github.com/fastygo/context/internal/artifacts"
	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
)

func TestArtifactRejectsZeroValue(t *testing.T) {
	t.Parallel()
	if err := (artifacts.Artifact{}).Validate(); err == nil {
		t.Fatal("expected zero artifact to fail")
	}
}

func TestArtifactAcceptsMinimalBlob(t *testing.T) {
	t.Parallel()
	a := artifacts.Artifact{
		ID:           "a1",
		ProjectID:    "p1",
		MediaType:    "text/plain",
		ByteSize:     3,
		Checksum:     "abc",
		StorageURI:   "artifact://p1/a1",
		ArtifactType: artifacts.TypeBlob,
	}
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestStructuredRequiresSchemaID(t *testing.T) {
	t.Parallel()
	a := artifacts.Artifact{
		ID:           "a1",
		ProjectID:    "p1",
		MediaType:    "application/json",
		ByteSize:     2,
		Checksum:     "ab",
		StorageURI:   "artifact://p1/a1",
		ArtifactType: artifacts.TypeStructured,
	}
	if err := a.Validate(); err == nil {
		t.Fatal("expected structured without schema_id to fail")
	}
	a.SchemaID = "uxspec.screen.v1"
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestSchemaIDForbiddenOnBlob(t *testing.T) {
	t.Parallel()
	err := artifacts.ValidateTypeAndSchema(artifacts.TypeBlob, "uxspec.screen.v1")
	if err == nil {
		t.Fatal("expected schema_id on blob to fail")
	}
}

func TestApplyPutOptionsPromotesSchemaToStructured(t *testing.T) {
	t.Parallel()
	art := artifacts.ApplyPutOptions(artifacts.Artifact{}, &artifacts.PutOptions{
		SchemaID: "uxspec.screen.v1",
	})
	if art.ArtifactType != artifacts.TypeStructured || art.SchemaID != "uxspec.screen.v1" {
		t.Fatalf("got type=%q schema=%q", art.ArtifactType, art.SchemaID)
	}
}

func TestArtifactLineageValidatesMultipleInputs(t *testing.T) {
	t.Parallel()
	lineage := artifacts.ArtifactLineage{
		ProjectID:        "p1",
		OutputArtifactID: "derived",
		InputArtifactIDs: []ids.ArtifactID{"input-a", "input-b"},
		SourceRefs: []corpus.SourceRef{{
			ProjectID: "p1",
			SourceID:  "source-a",
			Span:      foundation.ByteSpan{Start: 0, End: 4},
			Checksum:  "abcd",
		}},
		ContextPackID:      "pack1",
		AgentRunID:         "run1",
		ToolCallID:         "tool1",
		GeneratorID:        "example-generator",
		GeneratorVersion:   "v1",
		TransformationKind: "aggregate",
		CreatedAt:          time.Unix(10, 0).UTC(),
	}
	if err := lineage.Validate(); err != nil {
		t.Fatal(err)
	}

	lineage.InputArtifactIDs = append(lineage.InputArtifactIDs, "input-a")
	if err := lineage.Validate(); err == nil {
		t.Fatal("expected duplicate lineage input to fail")
	}
}

func TestArtifactLineageRejectsOutputAsInput(t *testing.T) {
	t.Parallel()
	lineage := artifacts.ArtifactLineage{
		ProjectID:          "p1",
		OutputArtifactID:   "derived",
		InputArtifactIDs:   []ids.ArtifactID{"derived"},
		GeneratorID:        "example-generator",
		GeneratorVersion:   "v1",
		TransformationKind: "copy",
		CreatedAt:          time.Unix(10, 0).UTC(),
	}
	if err := lineage.Validate(); err == nil {
		t.Fatal("expected self-referential lineage to fail")
	}
}

func TestArtifactLineageRejectsDuplicateSourceRefs(t *testing.T) {
	t.Parallel()
	ref := corpus.SourceRef{
		ProjectID: "p1",
		SourceID:  "source-a",
		Span:      foundation.ByteSpan{Start: 0, End: 4},
		Checksum:  "abcd",
	}
	lineage := artifacts.ArtifactLineage{
		ProjectID:          "p1",
		OutputArtifactID:   "derived",
		SourceRefs:         []corpus.SourceRef{ref, ref},
		GeneratorID:        "example-generator",
		GeneratorVersion:   "v1",
		TransformationKind: "aggregate",
		CreatedAt:          time.Unix(10, 0).UTC(),
	}
	if err := lineage.Validate(); err == nil {
		t.Fatal("expected duplicate source refs to fail")
	}
}

func TestArtifactLineageRejectsContextPackOnlyProvenance(t *testing.T) {
	t.Parallel()
	lineage := artifacts.ArtifactLineage{
		ProjectID:          "p1",
		OutputArtifactID:   "derived",
		ContextPackID:      "pack1",
		GeneratorID:        "example-generator",
		GeneratorVersion:   "v1",
		TransformationKind: "summary",
		CreatedAt:          time.Unix(10, 0).UTC(),
	}
	if err := lineage.Validate(); err == nil {
		t.Fatal("ContextPack-only lineage must include an input artifact or source ref")
	}
}
