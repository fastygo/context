package graph_test

import (
	"testing"

	"github.com/fastygo/context/internal/graph"
)

func TestNodeRefRejectsEmpty(t *testing.T) {
	t.Parallel()
	if err := (graph.NodeRef{}).Validate(); err == nil {
		t.Fatal("expected zero node ref to fail")
	}
}
