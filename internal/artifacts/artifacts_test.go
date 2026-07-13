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

func TestArtifactAcceptsMinimal(t *testing.T) {
	t.Parallel()
	a := artifacts.Artifact{
		ID:         "a1",
		ProjectID:  "p1",
		MediaType:  "text/plain",
		ByteSize:   3,
		Checksum:   "abc",
		StorageURI: "artifact://p1/a1",
	}
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
}
