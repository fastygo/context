package foundation_test

import (
	"testing"

	"github.com/fastygo/context/internal/foundation"
)

func TestByteSpanRejectsEmptyAndInverted(t *testing.T) {
	t.Parallel()
	cases := []foundation.ByteSpan{
		{Start: 0, End: 0},
		{Start: 5, End: 5},
		{Start: 10, End: 3},
	}
	for _, span := range cases {
		if err := span.Validate(); err == nil {
			t.Fatalf("expected invalid span %#v", span)
		}
	}
}

func TestByteSpanAcceptsHalfOpen(t *testing.T) {
	t.Parallel()
	span := foundation.ByteSpan{Start: 0, End: 4}
	if err := span.Validate(); err != nil {
		t.Fatal(err)
	}
	if span.Len() != 4 {
		t.Fatalf("len=%d", span.Len())
	}
}

func TestSnapshotStatusActiveGate(t *testing.T) {
	t.Parallel()
	if !foundation.SnapshotReady.IsSearchableAsActive() {
		t.Fatal("ready must be searchable as active")
	}
	for _, s := range []foundation.SnapshotStatus{
		foundation.SnapshotBuilding,
		foundation.SnapshotFailed,
		foundation.SnapshotSuperseded,
	} {
		if s.IsSearchableAsActive() {
			t.Fatalf("%s must not be active", s)
		}
	}
}

func TestEvidenceClassFactualGate(t *testing.T) {
	t.Parallel()
	if !foundation.EvidenceSourceText.MayJustifyFactualClaims() {
		t.Fatal("source_text should justify facts")
	}
	if foundation.EvidenceModelInference.MayJustifyFactualClaims() {
		t.Fatal("model_inference must not justify facts")
	}
	if err := foundation.EvidenceClass("").Validate(); err == nil {
		t.Fatal("empty evidence class should fail")
	}
}

func TestTrustLevelRejectsUnknown(t *testing.T) {
	t.Parallel()
	if err := foundation.TrustLevel("mystery").Validate(); err == nil {
		t.Fatal("expected unknown trust level to fail")
	}
}
