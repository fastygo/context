package runtime_test

import (
	"context"
	"fmt"

	memory "github.com/fastygo/context/pkg/contextkit/runtime"
)

func ExampleNew() {
	ctx := context.Background()
	rt, err := memory.New(ctx, memory.Config{ProjectID: "demo", Sources: []memory.Source{{
		SourceID: "ticket-1", Version: "v1", Text: "The account is locked.",
		TrustLevel: "project", EvidenceClass: "source_text",
	}}})
	if err != nil {
		panic(err)
	}
	result, err := rt.ContextPack(ctx, memory.PackRequest{ProjectID: "demo", Query: "account",
		Focus: memory.Focus{ID: "routing-v1", Objective: "Find account evidence", RequiredTrustLevel: "project",
			Budget: memory.Budget{MaxItems: 8, MaxChars: 4096}},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(result.FocusID, result.Snapshot.RuntimeVersion, len(result.Snapshot.Sources))
	// Output: routing-v1 memory-exact-v1 1
}
