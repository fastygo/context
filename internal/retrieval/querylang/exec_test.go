package querylang_test

import (
	"context"
	"testing"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/linguistic/registry"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/index"
	"github.com/fastygo/context/internal/retrieval/querylang"
)

const (
	projID = ids.ProjectID("p_ql")
	snapID = ids.SnapshotID("snap_ql")
)

func fixtureIndex() *index.Memory {
	rec := func(chunk, source, text string) index.ChunkRecord {
		return index.ChunkRecord{
			ProjectID: projID, SnapshotID: snapID,
			ChunkID: ids.ChunkID(chunk), SourceID: ids.SourceID(source),
			Span: foundation.ByteSpan{Start: 0, End: uint64(len(text))},
			Text: text, TextChecksum: foundation.ChecksumHex("h-" + chunk),
			TrustLevel: foundation.TrustProject, Language: "ru",
		}
	}
	return index.NewMemory(
		rec("c-road", "s1", "Вдоль железной дороги шли люди"),
		rec("c-road2", "s2", "Новая дорога построена за год"),
		rec("c-home", "s3", "Домашний уют и тепло"),
		rec("c-dom", "s4", "Старый дом стоит у реки"),
		rec("c-chat", "s5", "чат и переписка о погоде"),
		rec("c-mem", "s6", "память проекта хранит решения"),
		rec("c-memchat", "s7", "память диалога в чат приложении"),
	)
}

func ruRunner(idx *index.Memory) querylang.Runner {
	ports, _ := registry.ForLanguage("ru")
	return querylang.Runner{
		Index:      idx,
		Normalizer: ports.Normalizer,
		Analyzer:   ports.Analyzer,
		Expander:   ports.Expander,
		Language:   "ru",
		AdapterID:  string(ports.Version.AdapterID),
	}
}

func plan() retrieval.RetrievalPlan {
	return retrieval.RetrievalPlan{
		ID: "plan-ql", ProjectID: projID, SnapshotID: snapID, TopNRawPool: 20,
		Strategies: []retrieval.RetrieverStrategy{{RetrieverID: "term"}},
	}
}

func chunkIDs(cands []retrieval.Candidate) map[string]bool {
	out := map[string]bool{}
	for _, c := range cands {
		out[string(c.ChunkID)] = true
	}
	return out
}

func run(t *testing.T, r querylang.Runner, q string) querylang.Result {
	t.Helper()
	res, err := r.Run(context.Background(), plan(), ids.QueryID("q-"+q), q)
	if err != nil {
		t.Fatalf("Run(%q): %v", q, err)
	}
	return res
}

func TestTermMorphExpansionFindsInflectedForm(t *testing.T) {
	t.Parallel()
	r := ruRunner(fixtureIndex())
	res := run(t, r, "дорога")
	got := chunkIDs(res.Candidates)
	// "дороги" (c-road, gen.) and "дорога" (c-road2, nom.) must both match.
	if !got["c-road"] || !got["c-road2"] {
		t.Fatalf("morph expansion missed inflected forms: %v", got)
	}
	// Expansions must be visible in explain.
	if len(res.Explain.Leaves) != 1 || len(res.Explain.Leaves[0].Expansions) == 0 {
		t.Fatalf("explain missing expansions: %#v", res.Explain)
	}
}

func TestTermTokenBoundaryPrecision(t *testing.T) {
	t.Parallel()
	r := ruRunner(fixtureIndex())
	res := run(t, r, "дом")
	got := chunkIDs(res.Candidates)
	if !got["c-dom"] {
		t.Fatalf("token term must match 'дом': %v", got)
	}
	// Substring semantics would match "Домашний"; token boundaries must not.
	if got["c-home"] {
		t.Fatalf("token term must not match inside 'Домашний': %v", got)
	}
}

func TestAndIntersection(t *testing.T) {
	t.Parallel()
	r := ruRunner(fixtureIndex())
	res := run(t, r, "память чат")
	got := chunkIDs(res.Candidates)
	if !got["c-memchat"] || len(got) != 1 {
		t.Fatalf("AND intersection wrong: %v", got)
	}
}

func TestOrUnion(t *testing.T) {
	t.Parallel()
	r := ruRunner(fixtureIndex())
	res := run(t, r, "память OR чат")
	got := chunkIDs(res.Candidates)
	if !got["c-mem"] || !got["c-chat"] || !got["c-memchat"] {
		t.Fatalf("OR union wrong: %v", got)
	}
}

func TestNotExclusion(t *testing.T) {
	t.Parallel()
	r := ruRunner(fixtureIndex())
	res := run(t, r, "память -чат")
	got := chunkIDs(res.Candidates)
	if !got["c-mem"] || got["c-memchat"] {
		t.Fatalf("NOT exclusion wrong: %v", got)
	}
}

func TestExactPhraseLiteral(t *testing.T) {
	t.Parallel()
	r := ruRunner(fixtureIndex())
	res := run(t, r, `"память проекта"`)
	got := chunkIDs(res.Candidates)
	if !got["c-mem"] || len(got) != 1 {
		t.Fatalf("exact phrase wrong: %v", got)
	}
	if res.Candidates[0].Snippet == nil {
		t.Fatal("exact phrase must attach snippet")
	}
}

func TestMorphPhraseMatchesInflectedPhrase(t *testing.T) {
	t.Parallel()
	r := ruRunner(fixtureIndex())
	// Query in citation form; text contains "железной дороги" (genitive).
	res := run(t, r, `~"железная дорога"`)
	got := chunkIDs(res.Candidates)
	if !got["c-road"] || len(got) != 1 {
		t.Fatalf("morph phrase wrong: %v", got)
	}
	cand := res.Candidates[0]
	if cand.Snippet == nil {
		t.Fatal("morph phrase must attach snippet")
	}
	hasReason := false
	for _, contrib := range cand.Contributions {
		for _, reason := range contrib.Reasons {
			if reason == foundation.ReasonMorphPhrase {
				hasReason = true
			}
		}
	}
	if !hasReason {
		t.Fatalf("missing morph_phrase reason: %#v", cand.Contributions)
	}
}

func TestMorphPhraseKeepsWordOrder(t *testing.T) {
	t.Parallel()
	r := ruRunner(fixtureIndex())
	res := run(t, r, `~"дорога железная"`)
	if len(res.Candidates) != 0 {
		t.Fatalf("reversed phrase must not match consecutive sequence: %v", chunkIDs(res.Candidates))
	}
}

func TestGroupingWithParens(t *testing.T) {
	t.Parallel()
	r := ruRunner(fixtureIndex())
	res := run(t, r, "(память OR переписка) -диалога")
	got := chunkIDs(res.Candidates)
	if !got["c-mem"] || !got["c-chat"] || got["c-memchat"] {
		t.Fatalf("grouped query wrong: %v", got)
	}
}

func TestLangDirectiveOverridesRunnerLanguage(t *testing.T) {
	t.Parallel()
	idx := fixtureIndex()
	ports, _ := registry.ForLanguage("ru")
	r := querylang.Runner{
		Index:      idx,
		Normalizer: ports.Normalizer,
		Analyzer:   ports.Analyzer,
		Expander:   ports.Expander,
		Language:   "", // no default; query directive must activate morphology
	}
	res := run(t, r, "lang:ru дорога")
	if !chunkIDs(res.Candidates)["c-road"] {
		t.Fatalf("lang: directive did not enable morphology: %v", chunkIDs(res.Candidates))
	}
	if res.Explain.Language != "ru" {
		t.Fatalf("explain language=%q", res.Explain.Language)
	}
}

func TestRejectedExpansionsExcluded(t *testing.T) {
	t.Parallel()
	r := ruRunner(fixtureIndex())
	r.RejectExp = map[string]bool{"дороги": true}
	res := run(t, r, "дорога")
	leaf := res.Explain.Leaves[0]
	found := false
	for _, rej := range leaf.RejectedExpansions {
		if rej == "дороги" {
			found = true
		}
	}
	if !found {
		t.Fatalf("rejected expansion must be listed: %#v", leaf)
	}
	for _, e := range leaf.Expansions {
		if e == "дороги" {
			t.Fatal("rejected expansion leaked into accepted list")
		}
	}
}

func TestProjectSnapshotIsolation(t *testing.T) {
	t.Parallel()
	idx := fixtureIndex()
	idx.Add(index.ChunkRecord{
		ProjectID: "p_other", SnapshotID: snapID, ChunkID: "c-leak", SourceID: "s-leak",
		Span: foundation.ByteSpan{Start: 0, End: 10}, Text: "память проекта",
		TextChecksum: "h-leak", TrustLevel: foundation.TrustProject, Language: "ru",
	})
	r := ruRunner(idx)
	res := run(t, r, "память")
	if chunkIDs(res.Candidates)["c-leak"] {
		t.Fatal("cross-project leak in querylang")
	}
}

func TestFiltersApplied(t *testing.T) {
	t.Parallel()
	idx := fixtureIndex()
	idx.Add(index.ChunkRecord{
		ProjectID: projID, SnapshotID: snapID, ChunkID: "c-en", SourceID: "s-en",
		Span: foundation.ByteSpan{Start: 0, End: 12}, Text: "память test en",
		TextChecksum: "h-en", TrustLevel: foundation.TrustProject, Language: "en",
	})
	r := ruRunner(idx)
	p := plan()
	p.Filters.Language = "ru"
	res, err := r.Run(context.Background(), p, "q-filter", "память")
	if err != nil {
		t.Fatal(err)
	}
	if chunkIDs(res.Candidates)["c-en"] {
		t.Fatal("language filter ignored")
	}
}

func TestTombstoneExcluded(t *testing.T) {
	t.Parallel()
	idx := fixtureIndex()
	idx.Add(index.ChunkRecord{
		ProjectID: projID, SnapshotID: snapID, ChunkID: "c-dead", SourceID: "s-dead",
		Span: foundation.ByteSpan{Start: 0, End: 10}, Text: "память умершего чанка",
		TextChecksum: "h-dead", TrustLevel: foundation.TrustProject, Language: "ru",
		Tombstoned: true,
	})
	r := ruRunner(idx)
	res := run(t, r, "память")
	if chunkIDs(res.Candidates)["c-dead"] {
		t.Fatal("tombstoned chunk leaked into operator search")
	}
}

func TestDeterministicOrdering(t *testing.T) {
	t.Parallel()
	r := ruRunner(fixtureIndex())
	a := run(t, r, "память OR чат")
	b := run(t, r, "память OR чат")
	if len(a.Candidates) != len(b.Candidates) {
		t.Fatal("nondeterministic candidate count")
	}
	for i := range a.Candidates {
		if a.Candidates[i].ChunkID != b.Candidates[i].ChunkID {
			t.Fatalf("nondeterministic order at %d", i)
		}
	}
}

func TestNoLinguisticPortsDegradesGracefully(t *testing.T) {
	t.Parallel()
	r := querylang.Runner{Index: fixtureIndex()}
	res := run(t, r, "память -чат")
	got := chunkIDs(res.Candidates)
	if !got["c-mem"] || got["c-memchat"] {
		t.Fatalf("plain token search must work without adapters: %v", got)
	}
	// Morph phrase without analyzer degrades to folded token sequence.
	res2 := run(t, r, `~"память проекта"`)
	if !chunkIDs(res2.Candidates)["c-mem"] {
		t.Fatalf("morph phrase fallback failed: %v", chunkIDs(res2.Candidates))
	}
}

func TestExplainCanonicalAndTraceEvents(t *testing.T) {
	t.Parallel()
	r := ruRunner(fixtureIndex())
	res := run(t, r, `память -чат`)
	if res.Explain.Canonical != "(AND память (NOT чат))" {
		t.Fatalf("canonical=%q", res.Explain.Canonical)
	}
	if len(res.Events) < 2 {
		t.Fatalf("expected trace events, got %d", len(res.Events))
	}
	if res.Events[0].Payload["layer"] != "querylang" {
		t.Fatalf("trace payload: %#v", res.Events[0].Payload)
	}
}
