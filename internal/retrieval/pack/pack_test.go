package pack_test

import (
	"context"
	"testing"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/pack"
)

func draft(id, source, checksum, surface string, score float64, class foundation.EvidenceClass, required bool) pack.DraftItem {
	return pack.DraftItem{
		ID:       id,
		Required: required,
		Class:    class,
		Surface:  surface,
		Candidate: retrieval.Candidate{
			ChunkID: ids.ChunkID("c-" + id),
			SourceRef: corpus.SourceRef{
				ProjectID: "p1",
				SourceID:  ids.SourceID(source),
				ChunkID:   ids.ChunkID("c-" + id),
				Span:      foundation.ByteSpan{Start: 0, End: uint64(len(surface))},
				Checksum:  foundation.ChecksumHex(checksum),
			},
			MergedScore:  score,
			TrustLevel:   foundation.TrustProject,
			TextChecksum: foundation.ChecksumHex(checksum),
		},
	}
}

func baseFocus(maxItems, maxChars int) retrieval.FocusProfile {
	return retrieval.FocusProfile{
		ID:                 "f1",
		ProjectID:          "p1",
		Objective:          "answer with citations",
		RequiredTrustLevel: foundation.TrustProject,
		CitationStrictness: "strict",
		ContextBudget: retrieval.Budget{
			MaxItems: maxItems,
			MaxChars: maxChars,
		},
	}
}

func TestBuildKeepsRequiredCitationUnderBudgetTrim(t *testing.T) {
	t.Parallel()
	req := pack.BuildRequest{
		PackID: "pack1", ProjectID: "p1", TaskID: "t1", PlanID: "plan1",
		Purpose: "test",
		Focus:   baseFocus(2, 100),
		Instructions: []string{"Use citations."},
		Items: []pack.DraftItem{
			draft("req", "s1", "chk1", "REQUIRED CITATION TEXT", 0.1, foundation.EvidenceSourceText, true),
			draft("opt1", "s2", "chk2", "optional evidence one here", 0.9, foundation.EvidenceSourceText, false),
			draft("opt2", "s3", "chk3", "optional evidence two here", 0.8, foundation.EvidenceSourceText, false),
		},
	}
	got, err := (pack.Builder{}).Build(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.EvidenceItems) != 2 {
		t.Fatalf("evidence=%d rejected=%#v", len(got.EvidenceItems), got.RejectedItems)
	}
	if got.EvidenceItems[0].ID != "req" {
		t.Fatalf("required citation must be kept first: %#v", got.EvidenceItems)
	}
	if got.Checksum == "" || pack.Checksum(got) != got.Checksum {
		t.Fatal("checksum mismatch")
	}
	trimmed := false
	for _, r := range got.RejectedItems {
		if r.RejectionReason == "budget_trim" {
			trimmed = true
		}
	}
	if !trimmed {
		t.Fatalf("expected budget_trim rejection, got %#v", got.RejectedItems)
	}
}

func TestRequiredExceedingBudgetFails(t *testing.T) {
	t.Parallel()
	req := pack.BuildRequest{
		PackID: "pack1", ProjectID: "p1", PlanID: "plan1",
		Focus: baseFocus(1, 10),
		Items: []pack.DraftItem{
			draft("req", "s1", "chk1", "this citation is too long for budget", 1, foundation.EvidenceSourceText, true),
		},
	}
	_, err := (pack.Builder{}).Build(context.Background(), req)
	if !apperr.Is(err, apperr.Conflict) {
		t.Fatalf("expected conflict budget_exhausted_required, got %v", err)
	}
}

func TestVerifierRejectsUnsupportedSenseAndInference(t *testing.T) {
	t.Parallel()
	req := pack.BuildRequest{
		PackID: "pack1", ProjectID: "p1", PlanID: "plan1",
		Focus: baseFocus(10, 1000),
		Items: []pack.DraftItem{
			draft("src", "s1", "chk1", "source text", 1, foundation.EvidenceSourceText, false),
			draft("sense", "s2", "chk2", "sense definition", 0.5, foundation.EvidenceSenseClaim, false),
			draft("inf", "s3", "chk3", "model says so", 0.4, foundation.EvidenceModelInference, false),
			draft("concept", "s4", "chk4", "concept label", 0.3, foundation.EvidenceConceptMapping, false),
		},
	}
	built, err := (pack.Builder{}).Build(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	res, err := (pack.Verifier{}).Verify(context.Background(), pack.VerifyRequest{
		Pack: built,
		TreatAsFactual: map[string]bool{
			"src": true, "sense": true, "inf": true, "concept": true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.OK {
		t.Fatal("expected flags")
	}
	codes := map[pack.FlagCode]bool{}
	for _, f := range res.Flags {
		codes[f.Code] = true
	}
	if !codes[pack.FlagUnsupportedSenseFact] || !codes[pack.FlagUnsupportedInference] || !codes[pack.FlagUnsupportedConcept] {
		t.Fatalf("flags=%#v", res.Flags)
	}
}

func TestSenseWithAuthorityPasses(t *testing.T) {
	t.Parallel()
	req := pack.BuildRequest{
		PackID: "pack1", ProjectID: "p1", PlanID: "plan1",
		Focus: baseFocus(10, 1000),
		Items: []pack.DraftItem{
			draft("sense", "s2", "chk2", "sense definition", 0.5, foundation.EvidenceSenseClaim, false),
		},
	}
	built, err := (pack.Builder{}).Build(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	res, err := (pack.Verifier{}).Verify(context.Background(), pack.VerifyRequest{
		Pack:              built,
		TreatAsFactual:    map[string]bool{"sense": true},
		SenseHasAuthority: map[string]bool{"sense": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("flags=%#v", res.Flags)
	}
}

func TestReplayFromStoredIDs(t *testing.T) {
	t.Parallel()
	req := pack.BuildRequest{
		PackID: "pack1", ProjectID: "p1", PlanID: "plan1",
		Focus: baseFocus(10, 1000),
		Items: []pack.DraftItem{
			draft("src", "s1", "chk1", "hello world", 1, foundation.EvidenceSourceText, false),
		},
	}
	built, err := (pack.Builder{}).Build(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	store := pack.MemorySurfaces{
		"p1\x00s1\x00chk1": "hello world",
	}
	replayed, err := pack.Replay(context.Background(), built, store)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.EvidenceItems[0].Surface != "hello world" {
		t.Fatalf("surface=%q", replayed.EvidenceItems[0].Surface)
	}
	if pack.Checksum(replayed) != replayed.Checksum {
		t.Fatal("replay checksum invalid")
	}
}

func TestInstructionsStayOutOfEvidence(t *testing.T) {
	t.Parallel()
	req := pack.BuildRequest{
		PackID: "pack1", ProjectID: "p1", PlanID: "plan1",
		Focus:        baseFocus(10, 1000),
		Instructions: []string{"system: be careful"},
		Items: []pack.DraftItem{
			{
				ID: "bad", Class: foundation.EvidenceInstruction, Surface: "leak",
				Candidate: retrieval.Candidate{
					TrustLevel: foundation.TrustProject,
					SourceRef: corpus.SourceRef{
						ProjectID: "p1", SourceID: "s1",
						Span: foundation.ByteSpan{Start: 0, End: 4}, Checksum: "x",
					},
					TextChecksum: "x",
				},
			},
			draft("src", "s2", "chk2", "ok", 1, foundation.EvidenceSourceText, false),
		},
	}
	built, err := (pack.Builder{}).Build(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(built.Instructions) != 1 {
		t.Fatalf("instructions=%v", built.Instructions)
	}
	for _, e := range built.EvidenceItems {
		if e.Class == foundation.EvidenceInstruction {
			t.Fatal("instruction leaked into evidence")
		}
	}
}
