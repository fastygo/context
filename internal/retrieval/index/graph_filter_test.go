package index_test

import (
	"testing"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/index"
)

// ADR-0040: GraphNodeID must not silently change recall until enforced.
func TestGraphNodeIDFilterIgnored(t *testing.T) {
	t.Parallel()
	rec := index.ChunkRecord{
		ProjectID: "p1", SnapshotID: "s1", ChunkID: "c1", SourceID: "src",
		Span: foundation.ByteSpan{Start: 0, End: 4}, Text: "test",
		TextChecksum: "h1", TrustLevel: foundation.TrustProject,
	}
	with := retrieval.RetrievalFilters{GraphNodeID: "node-1"}
	without := retrieval.RetrievalFilters{}
	if index.MatchesFilters(rec, with) != index.MatchesFilters(rec, without) {
		t.Fatal("GraphNodeID must be ignored by MatchesFilters (ADR-0040)")
	}
	if !index.MatchesFilters(rec, with) {
		t.Fatal("expected match")
	}
}
