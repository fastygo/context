package hybrid_test

import (
	"context"
	"testing"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/lexicon"
	lexfake "github.com/fastygo/context/internal/lexicon/fake"
	"github.com/fastygo/context/internal/linguistic"
	"github.com/fastygo/context/internal/linguistic/simple"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/exact"
	"github.com/fastygo/context/internal/retrieval/fake"
	"github.com/fastygo/context/internal/retrieval/hybrid"
	"github.com/fastygo/context/internal/retrieval/index"
	"github.com/fastygo/context/internal/indexing"
)

func corpus() *index.Memory {
	return index.NewMemory(
		index.ChunkRecord{
			ProjectID: "p1", SnapshotID: "snap1", ChunkID: "c-run", SourceID: "s1",
			Span: foundation.ByteSpan{Start: 0, End: 30},
			Text: "runners run in the park", TextChecksum: "h1", TrustLevel: foundation.TrustProject,
			Lemmas: []string{"run"}, Wordforms: []string{"runners", "run"},
			SenseIDs: []ids.SenseID{"sense-run-sport"}, ConceptIDs: []ids.ConceptID{"concept-running"},
			AttestationIDs: []ids.AttestationID{"att-1"},
			Register: "sport", Region: "us", TimePeriod: "2020s",
			LexiconSourceID: "lex-1", SourceAuthority: "fixture",
		},
		index.ChunkRecord{
			ProjectID: "p1", SnapshotID: "snap1", ChunkID: "c-bank", SourceID: "s2",
			Span: foundation.ByteSpan{Start: 0, End: 24},
			Text: "river bank erosion", TextChecksum: "h2", TrustLevel: foundation.TrustProject,
			SenseIDs: []ids.SenseID{"sense-bank-river"},
		},
		index.ChunkRecord{
			ProjectID: "p1", SnapshotID: "snap1", ChunkID: "c-cite", SourceID: "s3",
			Span: foundation.ByteSpan{Start: 0, End: 40},
			Text: `see "ContextPack" in ADR-0020`, TextChecksum: "h3", TrustLevel: foundation.TrustTrusted,
		},
	)
}

func TestExactAndSparseWithoutVectors(t *testing.T) {
	t.Parallel()
	idx := corpus()
	eng := hybrid.Engine{
		Exact:  exact.Retriever{Index: idx},
		Sparse: fake.SparseRetriever{Client: fake.SparseClient{Index: idx}, Index: idx},
	}
	plan := retrieval.RetrievalPlan{
		ID: "plan1", ProjectID: "p1", SnapshotID: "snap1",
		Strategies: []retrieval.RetrieverStrategy{
			{RetrieverID: "exact"},
			{RetrieverID: "sparse"},
		},
	}
	res, err := eng.Search(context.Background(), plan, "q1", "runners")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Candidates) == 0 {
		t.Fatal("expected candidates")
	}
	if res.Candidates[0].Contributions[0].Explanation == "" && res.Candidates[0].Contributions[0].Reasons == nil {
		t.Fatal("expected explainable contribution")
	}
}

func TestLemmaVsWordformExpansion(t *testing.T) {
	t.Parallel()
	idx := corpus()
	eng := hybrid.Engine{
		Exact: exact.Retriever{Index: idx},
		Expander: simple.Expander{
			LemmaMap:    map[string]string{"running": "run"},
			WordformMap: map[string][]string{"run": {"runners"}},
		},
	}
	plan := retrieval.RetrievalPlan{
		ID: "plan1", ProjectID: "p1", SnapshotID: "snap1",
		Strategies: []retrieval.RetrieverStrategy{{RetrieverID: "exact"}},
	}
	res, err := eng.Search(context.Background(), plan, "q2", "run")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Expansions) == 0 {
		t.Fatal("expected wordform expansion")
	}
	found := false
	for _, c := range res.Candidates {
		if c.ChunkID == "c-run" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected c-run via expansion, got %#v", res.Candidates)
	}
}

func TestAmbiguousWordformAndSenseFilter(t *testing.T) {
	t.Parallel()
	idx := corpus()
	eng := hybrid.Engine{Exact: exact.Retriever{Index: idx}}
	plan := retrieval.RetrievalPlan{
		ID: "plan1", ProjectID: "p1", SnapshotID: "snap1",
		Strategies: []retrieval.RetrieverStrategy{{RetrieverID: "exact"}},
		Filters:    retrieval.RetrievalFilters{SenseID: "sense-bank-river"},
	}
	res, err := eng.Search(context.Background(), plan, "q3", "bank")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Candidates) != 1 || res.Candidates[0].ChunkID != "c-bank" {
		t.Fatalf("sense filter failed: %#v", res.Candidates)
	}
	hasSense := false
	for _, r := range res.Candidates[0].Contributions[0].Reasons {
		if r == foundation.ReasonSenseFilter {
			hasSense = true
		}
	}
	if !hasSense {
		t.Fatal("expected sense_filter reason")
	}
}

func TestConceptAndAttestationFilters(t *testing.T) {
	t.Parallel()
	idx := corpus()
	eng := hybrid.Engine{Exact: exact.Retriever{Index: idx}}
	plan := retrieval.RetrievalPlan{
		ID: "plan1", ProjectID: "p1", SnapshotID: "snap1",
		Strategies: []retrieval.RetrieverStrategy{{RetrieverID: "exact"}},
		Filters: retrieval.RetrievalFilters{
			ConceptID:     "concept-running",
			AttestationID: "att-1",
			Register:      "sport",
			DialectRegion: "us",
			TimePeriod:    "2020s",
		},
	}
	res, err := eng.Search(context.Background(), plan, "q4", "park")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Candidates) != 1 || res.Candidates[0].ChunkID != "c-run" {
		t.Fatalf("got %#v", res.Candidates)
	}
}

func TestQueryExpansionFalsePositiveRejected(t *testing.T) {
	t.Parallel()
	idx := corpus()
	eng := hybrid.Engine{
		Exact: exact.Retriever{Index: idx},
		Expander: simple.Expander{
			WordformMap: map[string][]string{"park": {"bank"}},
		},
		RejectExp: map[string]bool{"bank": true},
	}
	plan := retrieval.RetrievalPlan{
		ID: "plan1", ProjectID: "p1", SnapshotID: "snap1",
		Strategies: []retrieval.RetrieverStrategy{{RetrieverID: "exact"}},
	}
	res, err := eng.Search(context.Background(), plan, "q5", "park")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Rejected) != 1 {
		t.Fatalf("expected rejected expansion, got %#v", res.Rejected)
	}
	for _, c := range res.Candidates {
		if c.ChunkID == "c-bank" {
			t.Fatal("rejected expansion must not retrieve bank chunk")
		}
	}
}

func TestCitationLikeLookup(t *testing.T) {
	t.Parallel()
	idx := corpus()
	eng := hybrid.Engine{Exact: exact.Retriever{Index: idx}}
	plan := retrieval.RetrievalPlan{
		ID: "plan1", ProjectID: "p1", SnapshotID: "snap1",
		Strategies: []retrieval.RetrieverStrategy{{RetrieverID: "exact"}},
	}
	res, err := eng.Search(context.Background(), plan, "q6", `"ContextPack"`)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Candidates) != 1 || res.Candidates[0].ChunkID != "c-cite" {
		t.Fatalf("citation lookup failed: %#v", res.Candidates)
	}
}

func TestFakeVectorStoreRequiresNamespace(t *testing.T) {
	t.Parallel()
	store := fake.NewVectorStore()
	ns := indexing.VectorNamespace{Name: "ns", ProjectID: "p1", SnapshotID: "snap1", EmbeddingVersion: "emb-v1"}
	vec := fake.HashEmbed("runners run in the park", 8)
	err := store.Upsert(context.Background(), ns, []retrieval.VectorPoint{{
		ChunkID: "c-run", ProjectID: "p1", SnapshotID: "snap1", EmbeddingVersion: "emb-v1", Vector: vec,
	}})
	if err != nil {
		t.Fatal(err)
	}
	hits, err := store.Search(context.Background(), ns, vec, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].ChunkID != "c-run" {
		t.Fatalf("hits=%#v", hits)
	}
}

func TestLexiconFakeAndSimpleMorph(t *testing.T) {
	t.Parallel()
	res := lexfake.NewResource()
	res.Senses["sense-run-sport"] = lexicon.Sense{
		ID: "sense-run-sport", ProjectID: "p1", LexemeID: "lex-run", Language: "en", Definition: "to jog",
	}
	res.Concepts["concept-running"] = lexicon.Concept{
		ID: "concept-running", ProjectID: "p1", PreferredLabel: "running",
	}
	got, err := res.LookupSense(context.Background(), "p1", "sense-run-sport")
	if err != nil || got.Definition == "" {
		t.Fatalf("sense=%#v err=%v", got, err)
	}
	analyses, err := (simple.Analyzer{}).Analyze(context.Background(), linguistic.TokenOccurrence{
		ID: "t1", ProjectID: "p1", SourceID: "s1", ChunkID: "c1", Language: "en",
		Surface: "Runners", Normalized: "Runners", Span: foundation.ByteSpan{Start: 0, End: 7},
	})
	if err != nil || len(analyses) != 1 || analyses[0].Lemma != "runners" {
		t.Fatalf("analyses=%#v err=%v", analyses, err)
	}
}

func TestRetrievalTraceEvents(t *testing.T) {
	t.Parallel()
	idx := corpus()
	eng := hybrid.Engine{
		Exact: exact.Retriever{Index: idx},
		Expander: simple.Expander{
			WordformMap: map[string][]string{"run": {"runners"}},
		},
	}
	plan := retrieval.RetrievalPlan{
		ID: "plan1", ProjectID: "p1", SnapshotID: "snap1",
		Strategies: []retrieval.RetrieverStrategy{{RetrieverID: "exact"}},
	}
	res, err := eng.Search(context.Background(), plan, "q7", "run")
	if err != nil {
		t.Fatal(err)
	}
	types := map[string]bool{}
	for _, ev := range res.Events {
		types[string(ev.Type)] = true
		if ev.SnapshotID != "snap1" {
			t.Fatalf("missing snapshot on event %#v", ev)
		}
	}
	if !types["retrieval_query"] || !types["query_expansion"] || !types["retrieval_candidates"] {
		t.Fatalf("trace types=%v", types)
	}
}
