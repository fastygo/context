package devcli

import (
	"context"
	"os"
	"strings"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/config"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing"
	modelfake "github.com/fastygo/context/internal/models/fake"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/dense"
	"github.com/fastygo/context/internal/retrieval/dense/postgresvector"
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
	Backend    string                `json:"dense_backend,omitempty"`
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
			ChunkID:       ch.ChunkID,
			SourceID:     ch.SourceID,
			Span:         foundation.ByteSpan{Start: ch.SpanStart, End: ch.SpanEnd},
			Text:         ch.Text,
			TextChecksum: ch.TextChecksum,
			TrustLevel:   ch.TrustLevel,
		})
	}
	return mem, snap, nil
}

// Search runs exact/sparse/dense/hybrid retrieval over the local workspace index.
// Modes dense and hybrid-dense require PostgreSQL (CONTEXT_PG_DSN or compose defaults).
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
		ID: "cli-plan", ProjectID: st.Project.ID, SnapshotID: snap, TopNRawPool: 20,
		Strategies: []retrieval.RetrieverStrategy{},
	}

	eng := hybrid.Engine{
		Exact:  exact.Retriever{Index: idx},
		Sparse: fake.SparseRetriever{Client: fake.SparseClient{Index: idx}, Index: idx},
	}
	backend := ""

	wantDense := mode == "dense" || mode == "hybrid-dense" || (mode == "hybrid" && denseEnabledByEnv())
	if wantDense {
		store, ns, emb, err := openDenseStore(context.Background())
		if err != nil {
			return SearchResult{}, err
		}
		defer store.Close()
		if err := ensureDenseIndex(context.Background(), store, ns, emb, idx, st.Project.ID, snap); err != nil {
			return SearchResult{}, err
		}
		eng.Dense = dense.Retriever{
			Store: store, Embedder: emb, Index: idx, Namespace: ns,
		}
		backend = store.Capabilities().BackendID
	}

	switch mode {
	case "exact":
		plan.Strategies = []retrieval.RetrieverStrategy{{RetrieverID: "exact"}}
	case "sparse":
		plan.Strategies = []retrieval.RetrieverStrategy{{RetrieverID: "sparse"}}
	case "dense":
		plan.Strategies = []retrieval.RetrieverStrategy{{RetrieverID: "dense"}}
	case "hybrid":
		plan.Strategies = []retrieval.RetrieverStrategy{
			{RetrieverID: "exact"},
			{RetrieverID: "sparse"},
		}
		if eng.Dense != nil {
			plan.Strategies = append(plan.Strategies, retrieval.RetrieverStrategy{RetrieverID: "dense"})
		}
	case "hybrid-dense":
		plan.Strategies = []retrieval.RetrieverStrategy{
			{RetrieverID: "exact"},
			{RetrieverID: "sparse"},
			{RetrieverID: "dense"},
		}
	default:
		return SearchResult{}, apperr.New(apperr.Validation, "mode must be exact|sparse|hybrid|dense|hybrid-dense")
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
		Backend:    backend,
	}, nil
}

func denseEnabledByEnv() bool {
	v := strings.TrimSpace(os.Getenv("CONTEXT_ENABLE_DENSE"))
	return v == "1" || strings.EqualFold(v, "true")
}

func openDenseStore(ctx context.Context) (*postgresvector.Store, indexing.VectorNamespace, modelfake.Embedder, error) {
	cfg, err := config.LoadStorageConfigFromEnv()
	if err != nil {
		return nil, indexing.VectorNamespace{}, modelfake.Embedder{}, err
	}
	if cfg.Vector.Kind != config.StoreKindPostgresVector && cfg.Vector.Kind != "" {
		// Still allow explicit postgres DSN for CLI dense modes.
	}
	store, err := postgresvector.Open(ctx, cfg.Vector.DSN, postgresvector.Config{
		Collection: cfg.Vector.Collection,
		Dimension:  cfg.Vector.Dimension,
		Metric:     cfg.Vector.Metric,
	})
	if err != nil {
		return nil, indexing.VectorNamespace{}, modelfake.Embedder{}, err
	}
	if err := store.EnsureSchema(ctx); err != nil {
		store.Close()
		return nil, indexing.VectorNamespace{}, modelfake.Embedder{}, err
	}
	ns := indexing.VectorNamespace{
		Name:             cfg.Vector.Collection,
		EmbeddingVersion: cfg.Vector.EmbeddingVersion,
	}
	emb := modelfake.Embedder{Dim: cfg.Vector.Dimension}
	return store, ns, emb, nil
}

func ensureDenseIndex(
	ctx context.Context,
	store *postgresvector.Store,
	ns indexing.VectorNamespace,
	emb modelfake.Embedder,
	idx *index.Memory,
	projectID ids.ProjectID,
	snapshotID ids.SnapshotID,
) error {
	ns.ProjectID = projectID
	ns.SnapshotID = snapshotID
	if ns.Name == "" {
		ns.Name = config.DefaultVectorCollection
	}
	if ns.EmbeddingVersion == "" {
		ns.EmbeddingVersion = config.DefaultEmbeddingVersion
	}
	recs := idx.List(projectID, snapshotID)
	if len(recs) == 0 {
		return nil
	}
	texts := make([]string, len(recs))
	for i, rec := range recs {
		texts[i] = rec.Text
	}
	vecs, modelVer, err := emb.Embed(ctx, texts)
	if err != nil {
		return err
	}
	points := make([]retrieval.VectorPoint, 0, len(recs))
	for i, rec := range recs {
		points = append(points, retrieval.VectorPoint{
			ChunkID:          rec.ChunkID,
			ProjectID:        rec.ProjectID,
			SnapshotID:       rec.SnapshotID,
			EmbeddingVersion: firstNonEmpty(ns.EmbeddingVersion, modelVer),
			ChunkerVersion:   "cli-para-v1",
			MorphVersion:     "simple-v1",
			Language:         rec.Language,
			Span:             rec.Span,
			Vector:           vecs[i],
		})
	}
	return store.Upsert(ctx, ns, points)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
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
			ID:        string(c.ChunkID),
			Class:     foundation.EvidenceSourceText,
			Surface:   surface,
			Required:  i == 0,
			Candidate: c,
		})
	}
	built, err := (pack.Builder{}).Build(context.Background(), pack.BuildRequest{
		PackID:    ids.PackID("pack_" + string(search.SnapshotID)),
		ProjectID: st.Project.ID,
		TaskID:    "cli-task",
		PlanID:    "cli-plan",
		Purpose:   "cli-context-pack",
		Focus: retrieval.FocusProfile{
			ID: "cli-focus", ProjectID: st.Project.ID, Objective: query,
			RequiredTrustLevel: foundation.TrustProject,
			CitationStrictness: "strict",
			ContextBudget:      retrieval.Budget{MaxItems: 8, MaxChars: 4000},
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
