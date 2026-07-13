package manifest_test

import (
	"testing"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/indexing/manifest"
)

func TestDiffSourcesClassifiesChanges(t *testing.T) {
	t.Parallel()
	prev := map[string]foundation.ChecksumHex{"a": "1", "b": "2", "d": "4"}
	next := map[string]foundation.ChecksumHex{"b": "2", "c": "3", "a": "9"}
	diff := manifest.DiffSources(prev, next)
	kinds := map[manifest.ChangeKind]int{}
	for _, d := range diff {
		kinds[d.Kind]++
	}
	if kinds[manifest.Changed] != 1 || kinds[manifest.Added] != 1 || kinds[manifest.Removed] != 1 || kinds[manifest.Unchanged] != 1 {
		t.Fatalf("kinds=%v diff=%#v", kinds, diff)
	}
}
