package runtime

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/pack"
)

func TestExistingBuilderParity(t *testing.T) {
	ctx := context.Background()
	rt, err := New(ctx, Config{ProjectID: "p", Sources: []Source{{SourceID: "s", Version: "v1", Text: "one fact", TrustLevel: "project", EvidenceClass: "source_text"}}})
	if err != nil {
		t.Fatal(err)
	}
	req := PackRequest{ProjectID: "p", Query: "fact", Focus: Focus{ID: "f", Objective: "facts", RequiredTrustLevel: "project", Budget: Budget{MaxItems: 3, MaxChars: 100}}}
	result, err := rt.ContextPack(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	var actual retrieval.ContextPack
	if err = json.Unmarshal(result.ContextPack, &actual); err != nil {
		t.Fatal(err)
	}
	candidates, err := rt.retrieve(ctx, "p", "fact")
	if err != nil {
		t.Fatal(err)
	}
	expected, err := (pack.Builder{}).Build(ctx, pack.BuildRequest{
		PackID: actual.ID, PlanID: actual.RetrievalPlanID, ProjectID: "p", Purpose: "facts",
		Focus: retrieval.FocusProfile{ID: ids.FocusID("f"), ProjectID: "p", Objective: "facts", RequiredTrustLevel: foundation.TrustProject, ContextBudget: retrieval.Budget{MaxItems: 3, MaxChars: 100}},
		Items: []pack.DraftItem{{ID: string(candidates[0].ChunkID), Candidate: candidates[0], Surface: "one fact", Class: foundation.EvidenceSourceText}},
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(expected)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != string(result.ContextPack) {
		t.Fatalf("public adapter diverged from core\n%s\n%s", raw, result.ContextPack)
	}
	verified, err := (pack.Verifier{}).Verify(ctx, pack.VerifyRequest{Pack: actual})
	if err != nil || !verified.OK {
		t.Fatalf("verification: %+v %v", verified, err)
	}
}
