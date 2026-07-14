// Package golden is the Chunk 19 offline retrieval eval harness.
// Reports are Lab-facing JSON (no Lab imports). Dense uses an in-memory
// VectorStore by default; set CONTEXT_PG_DSN + CONTEXT_EVAL_DENSE=postgres
// only when intentionally gating against live pgvector (optional).
package golden

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/fastygo/context/internal/config"
	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing"
	"github.com/fastygo/context/internal/linguistic/simple"
	modelfake "github.com/fastygo/context/internal/models/fake"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/dense"
	"github.com/fastygo/context/internal/retrieval/exact"
	"github.com/fastygo/context/internal/retrieval/fake"
	"github.com/fastygo/context/internal/retrieval/hybrid"
	"github.com/fastygo/context/internal/retrieval/index"
	"github.com/fastygo/context/internal/retrieval/merge"
	"github.com/fastygo/context/internal/retrieval/pack"
	"github.com/fastygo/context/internal/retrieval/rerank"
	"github.com/fastygo/context/internal/retrieval/sparse"
)

const (
	ProjectID  = "p_eval"
	SnapshotID = "snap_eval"
	SuiteID    = "eval-golden-v2"
)

// CaseSpec is one golden expectation (also serialized under .proofs/eval).
type CaseSpec struct {
	ID           string   `json:"id"`
	Kind         string   `json:"kind"` // exact|sparse|dense|hybrid|multilingual|lexicon|event_window|pack_verify
	Query        string   `json:"query"`
	WantChunkIDs []string `json:"want_chunk_ids"`
	Language     string   `json:"language,omitempty"`
	SenseID      string   `json:"sense_id,omitempty"`
	ConceptID    string   `json:"concept_id,omitempty"`
	Notes        string   `json:"notes,omitempty"`
}

// CaseResult is one executed case for Lab.
type CaseResult struct {
	ID           string   `json:"id"`
	Kind         string   `json:"kind"`
	Passed       bool     `json:"passed"`
	Query        string   `json:"query"`
	WantChunkIDs []string `json:"want_chunk_ids"`
	GotChunkIDs  []string `json:"got_chunk_ids"`
	Score        float64  `json:"score"`
	Reasons      []string `json:"reasons,omitempty"`
	Notes        string   `json:"notes,omitempty"`
}

// Report is the machine-readable eval output (Lab-consumable).
type Report struct {
	OK          bool         `json:"ok"`
	SuiteID     string       `json:"suite_id"`
	GeneratedAt time.Time    `json:"generated_at"`
	Cases       []CaseResult `json:"cases"`
	Summary     string       `json:"summary"`
}

// Specs returns the default golden set (neutral fixtures).
func Specs() []CaseSpec {
	return []CaseSpec{
		{
			ID: "exact-runners", Kind: "exact", Query: "runners",
			WantChunkIDs: []string{"c-run"},
			Notes:        "exact substring hit on sport running chunk",
		},
		{
			ID: "sparse-park", Kind: "sparse", Query: "park runners",
			WantChunkIDs: []string{"c-run"},
			Notes:        "fake sparse term overlap",
		},
		{
			ID: "dense-run", Kind: "dense", Query: "runners run in the park",
			WantChunkIDs: []string{"c-run"},
			Notes:        "offline fake-hash dense similarity",
		},
		{
			ID: "hybrid-run-expand", Kind: "hybrid", Query: "run",
			WantChunkIDs: []string{"c-run"},
			Notes:        "hybrid exact+sparse+dense with wordform expansion",
		},
		{
			ID: "morph-fake-expand", Kind: "hybrid", Query: "run",
			WantChunkIDs: []string{"c-run"},
			Notes:        "C12 morph-fake: simple expander maps run→runners",
		},
		{
			ID: "multilingual-en-filter", Kind: "multilingual", Query: "run",
			WantChunkIDs: []string{"c-en"}, Language: "en",
			Notes: "language filter keeps en; ru excluded (Chunk 18 fixture)",
		},
		{
			ID: "lexicon-sense-concept", Kind: "lexicon", Query: "runners",
			WantChunkIDs: []string{"c-lex"}, SenseID: "sense-run-sport", ConceptID: "concept-running",
			Notes: "sense+concept filters preserve original surface (Chunk 18)",
		},
		{
			ID: "sense-attestation-filter", Kind: "lexicon", Query: "runners",
			WantChunkIDs: []string{"c-lex"}, SenseID: "sense-run-sport", ConceptID: "concept-running",
			Notes: "C12 sense/attestation filter reasons required",
		},
		{
			ID: "event-window", Kind: "event_window", Query: "deploy",
			WantChunkIDs: []string{"c-evt-in"},
			Notes:        "C12 temporal event-window keeps in-range chunk only",
		},
		{
			ID: "pack-verify-source", Kind: "pack_verify", Query: "runners",
			WantChunkIDs: []string{"c-run"},
			Notes:        "ContextPack build + Verifier OK on source_text evidence",
		},
	}
}

// Run executes all default golden cases offline.
func Run(ctx context.Context) (Report, error) {
	return RunSpecs(ctx, Specs())
}

// RunSpecs executes the provided cases against the fixture corpus.
func RunSpecs(ctx context.Context, specs []CaseSpec) (Report, error) {
	idx := fixtureIndex()
	store, ns, emb, err := openDenseOffline(idx)
	if err != nil {
		return Report{}, err
	}
	eng := hybrid.Engine{
		Exact:  exact.Retriever{Index: idx},
		Sparse: sparse.Retriever{Client: fake.SparseClient{Index: idx}, Index: idx, Explanation: "eval fake sparse"},
		Dense: dense.Retriever{
			Store: store, Embedder: emb, Index: idx, Namespace: ns,
		},
		Expander: simple.Expander{WordformMap: map[string][]string{"run": {"runners"}}},
		Reranker: rerank.Identity{},
	}

	rep := Report{
		SuiteID:     SuiteID,
		GeneratedAt: time.Now().UTC(),
		Cases:       make([]CaseResult, 0, len(specs)),
	}
	allOK := true
	for _, spec := range specs {
		cr, err := runCase(ctx, eng, idx, spec)
		if err != nil {
			return Report{}, err
		}
		if !cr.Passed {
			allOK = false
		}
		rep.Cases = append(rep.Cases, cr)
	}
	rep.OK = allOK
	if allOK {
		rep.Summary = "all golden cases passed"
	} else {
		rep.Summary = "one or more golden cases failed"
	}
	return rep, nil
}

func runCase(ctx context.Context, eng hybrid.Engine, idx *index.Memory, spec CaseSpec) (CaseResult, error) {
	cr := CaseResult{
		ID: spec.ID, Kind: spec.Kind, Query: spec.Query,
		WantChunkIDs: append([]string(nil), spec.WantChunkIDs...),
		Notes:        spec.Notes,
	}
	if spec.Kind == "pack_verify" {
		return runPackVerify(ctx, eng, idx, spec, cr)
	}

	plan := retrieval.RetrievalPlan{
		ID: ids.PlanID("eval-" + spec.ID), ProjectID: ProjectID, SnapshotID: SnapshotID,
		TopNRawPool: 20,
	}
	switch spec.Kind {
	case "exact":
		plan.Strategies = []retrieval.RetrieverStrategy{{RetrieverID: "exact"}}
	case "sparse":
		plan.Strategies = []retrieval.RetrieverStrategy{{RetrieverID: "sparse"}}
	case "dense":
		plan.Strategies = []retrieval.RetrieverStrategy{{RetrieverID: "dense"}}
	case "hybrid", "multilingual":
		plan.Strategies = []retrieval.RetrieverStrategy{
			{RetrieverID: "exact"}, {RetrieverID: "sparse"}, {RetrieverID: "dense"},
		}
	case "lexicon":
		plan.Strategies = []retrieval.RetrieverStrategy{{RetrieverID: "exact"}}
		plan.Filters = retrieval.RetrievalFilters{
			SenseID: ids.SenseID(spec.SenseID), ConceptID: ids.ConceptID(spec.ConceptID),
			AttestationID: "att-runners-1", Register: "sport", DialectRegion: "us",
			TimePeriod: "2020s", LexiconSourceID: "lex-proof-1", SourceAuthority: "proof-fixture",
		}
	case "event_window":
		plan.Strategies = []retrieval.RetrieverStrategy{{RetrieverID: "exact"}}
		t0 := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
		plan.Filters = retrieval.RetrievalFilters{
			TemporalRange: &corpus.TemporalRange{
				Start: t0, End: t0.Add(2 * time.Hour), Basis: corpus.TimeBasisOccurred,
			},
		}
	default:
		cr.Notes = "unknown kind"
		return cr, nil
	}
	if spec.Language != "" {
		plan.Filters.Language = spec.Language
	}

	res, err := eng.Search(ctx, plan, ids.QueryID("q-"+spec.ID), spec.Query)
	if err != nil {
		return CaseResult{}, err
	}
	cands := merge.DedupAndMerge(res.Candidates)
	got := make([]string, 0, len(cands))
	reasonSet := map[string]struct{}{}
	var top float64
	for i, c := range cands {
		got = append(got, string(c.ChunkID))
		if i == 0 {
			top = c.MergedScore
		}
		for _, contrib := range c.Contributions {
			for _, r := range contrib.Reasons {
				reasonSet[string(r)] = struct{}{}
			}
		}
	}
	sort.Strings(got)
	cr.GotChunkIDs = got
	cr.Score = top
	for r := range reasonSet {
		cr.Reasons = append(cr.Reasons, r)
	}
	sort.Strings(cr.Reasons)
	cr.Passed = containsAll(got, spec.WantChunkIDs) && len(cands) > 0
	if spec.Kind == "multilingual" {
		for _, id := range got {
			if id == "c-ru" {
				cr.Passed = false
				cr.Notes += "; ru leak"
			}
		}
	}
	if spec.Kind == "lexicon" {
		_, hasSense := reasonSet[string(foundation.ReasonSenseFilter)]
		_, hasConcept := reasonSet[string(foundation.ReasonConceptFilter)]
		if !hasSense || !hasConcept {
			cr.Passed = false
			cr.Notes += "; missing sense/concept filter reasons"
		}
		if spec.ID == "sense-attestation-filter" {
			_, hasAtt := reasonSet[string(foundation.ReasonAttestationFilter)]
			if !hasAtt {
				cr.Passed = false
				cr.Notes += "; missing attestation filter reason"
			}
		}
		if len(cands) == 1 && cands[0].TextChecksum != "h-lex" {
			cr.Passed = false
			cr.Notes += "; checksum changed"
		}
	}
	if spec.Kind == "event_window" {
		for _, id := range got {
			if id == "c-evt-out" {
				cr.Passed = false
				cr.Notes += "; out-of-window leak"
			}
		}
	}
	return cr, nil
}

func runPackVerify(ctx context.Context, eng hybrid.Engine, idx *index.Memory, spec CaseSpec, cr CaseResult) (CaseResult, error) {
	plan := retrieval.RetrievalPlan{
		ID: "eval-pack", ProjectID: ProjectID, SnapshotID: SnapshotID, TopNRawPool: 10,
		Strategies: []retrieval.RetrieverStrategy{{RetrieverID: "exact"}, {RetrieverID: "sparse"}},
	}
	res, err := eng.Search(ctx, plan, "q-pack", spec.Query)
	if err != nil {
		return CaseResult{}, err
	}
	cands := merge.DedupAndMerge(res.Candidates)
	items := make([]pack.DraftItem, 0, len(cands))
	got := make([]string, 0, len(cands))
	for i, c := range cands {
		got = append(got, string(c.ChunkID))
		rec, _ := idx.Get(ProjectID, SnapshotID, c.ChunkID)
		items = append(items, pack.DraftItem{
			ID: string(c.ChunkID), Class: foundation.EvidenceSourceText,
			Surface: rec.Text, Required: i == 0, Candidate: c,
		})
	}
	cr.GotChunkIDs = got
	built, err := (pack.Builder{}).Build(ctx, pack.BuildRequest{
		PackID: "pack_eval", ProjectID: ProjectID, TaskID: "eval-task", PlanID: "eval-pack",
		Purpose: "eval-golden", Focus: retrieval.FocusProfile{
			ID: "focus_eval", ProjectID: ProjectID, Objective: spec.Query,
			RequiredTrustLevel: foundation.TrustProject, CitationStrictness: "strict",
			ContextBudget: retrieval.Budget{MaxItems: 4, MaxChars: 2000},
		},
		Instructions: []string{"Cite evidence only."}, Items: items,
	})
	if err != nil {
		cr.Notes = err.Error()
		return cr, nil
	}
	verify, err := (pack.Verifier{}).Verify(ctx, pack.VerifyRequest{Pack: built})
	if err != nil {
		return CaseResult{}, err
	}
	cr.Passed = verify.OK && verify.PackOK && containsAll(got, spec.WantChunkIDs)
	cr.Score = 1
	if verify.OK {
		cr.Reasons = []string{"verify_ok", "pack_checksum_ok"}
	} else {
		for _, f := range verify.Flags {
			cr.Reasons = append(cr.Reasons, string(f.Code))
		}
	}
	return cr, nil
}

func fixtureIndex() *index.Memory {
	recs := []index.ChunkRecord{
		{
			ProjectID: ProjectID, SnapshotID: SnapshotID, ChunkID: "c-run", SourceID: "s1",
			Span: foundation.ByteSpan{Start: 0, End: 23},
			Text: "runners run in the park", TextChecksum: "h-run", TrustLevel: foundation.TrustProject,
			Language: "en", Lemmas: []string{"run"}, Wordforms: []string{"runners", "run"},
			SenseIDs: []ids.SenseID{"sense-run-sport"}, ConceptIDs: []ids.ConceptID{"concept-running"},
			AttestationIDs: []ids.AttestationID{"att-runners-1"},
			Register: "sport", Region: "us", TimePeriod: "2020s",
			LexiconSourceID: "lex-proof-1", SourceAuthority: "proof-fixture",
		},
		{
			ProjectID: ProjectID, SnapshotID: SnapshotID, ChunkID: "c-bank", SourceID: "s2",
			Span: foundation.ByteSpan{Start: 0, End: 18},
			Text: "river bank erosion", TextChecksum: "h-bank", TrustLevel: foundation.TrustProject,
			Language: "en", SenseIDs: []ids.SenseID{"sense-bank-river"},
		},
		{
			ProjectID: ProjectID, SnapshotID: SnapshotID, ChunkID: "c-en", SourceID: "s-en",
			Span: foundation.ByteSpan{Start: 4, End: 11},
			Text: "Runners run in the park", TextChecksum: "h-en", TrustLevel: foundation.TrustProject,
			Language: "en", AnalyzerVersion: "simple-v1",
			Lemmas: []string{"run"}, Wordforms: []string{"runners", "run"},
		},
		{
			ProjectID: ProjectID, SnapshotID: SnapshotID, ChunkID: "c-ru", SourceID: "s-ru",
			Span: foundation.ByteSpan{Start: 0, End: 12},
			Text: "Бегуны бегают в парке", TextChecksum: "h-ru", TrustLevel: foundation.TrustProject,
			Language: "ru", AnalyzerVersion: "simple-v1",
		},
		{
			ProjectID: ProjectID, SnapshotID: SnapshotID, ChunkID: "c-lex", SourceID: "s-lex",
			Span: foundation.ByteSpan{Start: 0, End: 7}, Text: "runners",
			TextChecksum: "h-lex", TrustLevel: foundation.TrustProject, Language: "en",
			SenseIDs: []ids.SenseID{"sense-run-sport"}, ConceptIDs: []ids.ConceptID{"concept-running"},
			AttestationIDs: []ids.AttestationID{"att-runners-1"},
			Register: "sport", Region: "us", TimePeriod: "2020s",
			LexiconSourceID: "lex-proof-1", SourceAuthority: "proof-fixture",
		},
	}
	recs = append(recs, eventWindowRecords()...)
	return index.NewMemory(recs...)
}

func eventWindowRecords() []index.ChunkRecord {
	t0 := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	return []index.ChunkRecord{
		{
			ProjectID: ProjectID, SnapshotID: SnapshotID, ChunkID: "c-evt-in", SourceID: "s-evt",
			Span: foundation.ByteSpan{Start: 0, End: 18}, Text: "deploy completed ok",
			TextChecksum: "h-evt-in", TrustLevel: foundation.TrustProject, Language: "en",
			TemporalMetadata: &corpus.TemporalMetadata{
				Range: corpus.TemporalRange{
					Start: t0.Add(30 * time.Minute), End: t0.Add(time.Hour), Basis: corpus.TimeBasisOccurred,
				},
				IngestedAt: t0.Add(3 * time.Hour),
			},
		},
		{
			ProjectID: ProjectID, SnapshotID: SnapshotID, ChunkID: "c-evt-out", SourceID: "s-evt",
			Span: foundation.ByteSpan{Start: 0, End: 16}, Text: "deploy rolled back",
			TextChecksum: "h-evt-out", TrustLevel: foundation.TrustProject, Language: "en",
			TemporalMetadata: &corpus.TemporalMetadata{
				Range: corpus.TemporalRange{
					Start: t0.Add(5 * time.Hour), End: t0.Add(6 * time.Hour), Basis: corpus.TimeBasisOccurred,
				},
				IngestedAt: t0.Add(7 * time.Hour),
			},
		},
	}
}

func openDenseOffline(idx *index.Memory) (*fake.VectorStore, indexing.VectorNamespace, modelfake.Embedder, error) {
	store := fake.NewVectorStore()
	ns := indexing.VectorNamespace{
		Name: config.DefaultVectorCollection, ProjectID: ProjectID, SnapshotID: SnapshotID,
		EmbeddingVersion: config.DefaultEmbeddingVersion,
	}
	emb := modelfake.Embedder{Dim: config.DefaultEmbeddingDimension}
	recs := idx.List(ProjectID, SnapshotID)
	docs := make([]dense.ChunkDoc, 0, len(recs))
	for _, rec := range recs {
		docs = append(docs, dense.ChunkDoc{
			ProjectID: rec.ProjectID, SnapshotID: rec.SnapshotID, ChunkID: rec.ChunkID,
			Text: rec.Text, Language: rec.Language, ChunkerVersion: "eval-v1",
			EmbeddingVersion: ns.EmbeddingVersion, MorphVersion: "simple-v1", Span: rec.Span,
		})
	}
	if err := dense.UpsertEmbedded(context.Background(), store, emb, ns, docs); err != nil {
		return nil, indexing.VectorNamespace{}, modelfake.Embedder{}, err
	}
	return store, ns, emb, nil
}

func containsAll(got, want []string) bool {
	set := map[string]struct{}{}
	for _, g := range got {
		set[g] = struct{}{}
	}
	for _, w := range want {
		if _, ok := set[w]; !ok {
			return false
		}
	}
	return len(want) > 0
}

// MarshalReport encodes the Lab-facing JSON report.
func MarshalReport(rep Report) ([]byte, error) {
	return json.MarshalIndent(rep, "", "  ")
}
