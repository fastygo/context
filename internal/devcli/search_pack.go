package devcli

import (
	"context"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/exact"
	"github.com/fastygo/context/internal/retrieval/fake"
	"github.com/fastygo/context/internal/retrieval/hybrid"
	"github.com/fastygo/context/internal/retrieval/index"
	"github.com/fastygo/context/internal/retrieval/merge"
	"github.com/fastygo/context/internal/retrieval/pack"
)

// SearchResult is CLI JSON for search.
type SearchResult struct {
	ProjectID  ids.ProjectID         `json:"project_id"`
	SnapshotID ids.SnapshotID        `json:"snapshot_id"`
	Query      string                `json:"query"`
	Mode       string                `json:"mode"`
	Candidates []retrieval.Candidate `json:"candidates"`
}

func loadIndex(st State) (*index.Memory, ids.SnapshotID, error) {
	if st.Project.ActiveSnapshotID == "" && st.Snapshot.ID == "" {
		return nil, "", apperr.New(apperr.Validation, "no active snapshot; run ingest")
	}
	snap := st.Project.ActiveSnapshotID
	if snap == "" {
		snap = st.Snapshot.ID
	}
	mem := index.NewMemory()
	for _, ch := range st.Chunks {
		if ch.SnapshotID != snap {
			continue
		}
		mem.Add(index.ChunkRecord{
			ProjectID:    st.Project.ID,
			SnapshotID:   ch.SnapshotID,
			ChunkID:      ch.ChunkID,
			SourceID:     ch.SourceID,
			Span:         foundation.ByteSpan{Start: ch.SpanStart, End: ch.SpanEnd},
			Text:         ch.Text,
			TextChecksum: ch.TextChecksum,
			TrustLevel:   ch.TrustLevel,
		})
	}
	return mem, snap, nil
}

// Search runs exact/sparse/hybrid retrieval over the local workspace index.
func Search(dataDir, projectID, query, mode string) (SearchResult, error) {
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return SearchResult{}, err
	}
	if projectID != "" && ids.ProjectID(projectID) != st.Project.ID {
		return SearchResult{}, apperr.New(apperr.Validation, "project id mismatch")
	}
	if mode == "" {
		mode = "hybrid"
	}
	idx, snap, err := loadIndex(st)
	if err != nil {
		return SearchResult{}, err
	}
	plan := retrieval.RetrievalPlan{
		ID: "cli-plan", ProjectID: st.Project.ID, SnapshotID: snap,
		Strategies: []retrieval.RetrieverStrategy{},
	}
	switch mode {
	case "exact":
		plan.Strategies = []retrieval.RetrieverStrategy{{RetrieverID: "exact"}}
	case "sparse":
		plan.Strategies = []retrieval.RetrieverStrategy{{RetrieverID: "sparse"}}
	case "hybrid":
		plan.Strategies = []retrieval.RetrieverStrategy{
			{RetrieverID: "exact"},
			{RetrieverID: "sparse"},
		}
	default:
		return SearchResult{}, apperr.New(apperr.Validation, "mode must be exact|sparse|hybrid")
	}

	eng := hybrid.Engine{
		Exact:  exact.Retriever{Index: idx},
		Sparse: fake.SparseRetriever{Client: fake.SparseClient{Index: idx}, Index: idx},
	}
	res, err := eng.Search(context.Background(), plan, "cli-q", query)
	if err != nil {
		return SearchResult{}, err
	}
	cands := merge.DedupAndMerge(res.Candidates)
	return SearchResult{
		ProjectID:  st.Project.ID,
		SnapshotID: snap,
		Query:      query,
		Mode:       mode,
		Candidates: cands,
	}, nil
}

// PackResult is CLI JSON for context-pack.
type PackResult struct {
	Pack retrieval.ContextPack `json:"context_pack"`
}

// BuildPack creates a ContextPack from search candidates for a query.
func BuildPack(dataDir, projectID, query string) (PackResult, error) {
	search, err := Search(dataDir, projectID, query, "hybrid")
	if err != nil {
		return PackResult{}, err
	}
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return PackResult{}, err
	}
	idx, _, err := loadIndex(st)
	if err != nil {
		return PackResult{}, err
	}
	items := make([]pack.DraftItem, 0, len(search.Candidates))
	for i, c := range search.Candidates {
		rec, ok := idx.Get(st.Project.ID, search.SnapshotID, c.ChunkID)
		surface := ""
		if ok {
			surface = rec.Text
		}
		items = append(items, pack.DraftItem{
			ID:       string(c.ChunkID),
			Class:    foundation.EvidenceSourceText,
			Surface:  surface,
			Required: i == 0,
			Candidate: c,
		})
	}
	built, err := (pack.Builder{}).Build(context.Background(), pack.BuildRequest{
		PackID:  ids.PackID("pack_" + string(search.SnapshotID)),
		ProjectID: st.Project.ID,
		TaskID:  "cli-task",
		PlanID:  "cli-plan",
		Purpose: "cli-context-pack",
		Focus: retrieval.FocusProfile{
			ID: "cli-focus", ProjectID: st.Project.ID, Objective: query,
			RequiredTrustLevel: foundation.TrustProject,
			CitationStrictness: "strict",
			ContextBudget: retrieval.Budget{MaxItems: 8, MaxChars: 4000},
		},
		Instructions: []string{"Answer using cited evidence only."},
		Items:        items,
	})
	if err != nil {
		return PackResult{}, err
	}
	st.Packs = append(st.Packs, built)
	if err := ws.Save(st); err != nil {
		return PackResult{}, err
	}
	return PackResult{Pack: built}, nil
}
