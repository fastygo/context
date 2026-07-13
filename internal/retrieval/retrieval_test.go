package retrieval_test

import (
	"testing"

	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/retrieval"
)

func TestContextPackRejectsInstructionInEvidence(t *testing.T) {
	t.Parallel()
	pack := retrieval.ContextPack{
		ID:              "pack1",
		ProjectID:       "p1",
		RetrievalPlanID: "plan1",
		EvidenceItems: []retrieval.EvidenceItem{{
			Class:      foundation.EvidenceInstruction,
			TrustLevel: foundation.TrustTrusted,
		}},
	}
	if err := pack.Validate(); err == nil {
		t.Fatal("instructions must not live in evidence_items")
	}
}

func TestCandidateDedupKey(t *testing.T) {
	t.Parallel()
	c := retrieval.Candidate{
		SourceRef: corpus.SourceRef{
			SourceID: "s1",
			Span:     foundation.ByteSpan{Start: 2, End: 8},
		},
		TextChecksum: "deadbeef",
	}
	if got := c.DedupKey(); got != "s1:2:8:deadbeef" {
		t.Fatalf("dedup key=%q", got)
	}
}

func TestRetrievalPlanRequiresStrategies(t *testing.T) {
	t.Parallel()
	p := retrieval.RetrievalPlan{
		ID:         "plan1",
		ProjectID:  "p1",
		SnapshotID: "snap1",
	}
	if err := p.Validate(); err == nil {
		t.Fatal("expected missing strategies to fail")
	}
}
