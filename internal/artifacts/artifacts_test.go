package artifacts_test

import (
	"testing"

	"github.com/fastygo/context/internal/artifacts"
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
