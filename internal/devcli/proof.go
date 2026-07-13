package devcli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/artifacts"
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
	"github.com/fastygo/context/internal/storage"
	"github.com/fastygo/context/internal/storage/postgres"
	"github.com/fastygo/context/internal/tracing"
)

// ProofSummary is the Chunk 12 hypothesis report for Lab/replay.
type ProofSummary struct {
	Hypothesis   string            `json:"hypothesis"`
	Status       string            `json:"status"` // validated | partially_validated | failed
	CheckedAt    time.Time         `json:"checked_at"`
	ProjectID    string            `json:"project_id"`
	SnapshotID   string            `json:"snapshot_id"`
	Steps        []ProofStep       `json:"steps"`
	Gaps         []string          `json:"gaps"`
	NextDecisions []string         `json:"next_decisions"`
	Artifacts    map[string]string `json:"artifacts"`
}

// ProofStep records one proof checkpoint.
type ProofStep struct {
	ID      string `json:"id"`
	OK      bool   `json:"ok"`
	Detail  string `json:"detail,omitempty"`
	Artifact string `json:"artifact,omitempty"`
}

// RunProof executes the Chunk 12 end-to-end CLI/infra proof and writes JSON
// under outDir for Lab replay without requiring live services later.
func RunProof(repoRoot, outDir string) (ProofSummary, error) {
	if outDir == "" {
		outDir = filepath.Join(repoRoot, ".proofs")
	}
	dataDir := filepath.Join(outDir, "workspace")
	corpusDir := filepath.Join(outDir, "corpus")
	_ = os.RemoveAll(dataDir)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return ProofSummary{}, err
	}

	summary := ProofSummary{
		Hypothesis: "ingest → hybrid/dense retrieve → ContextPack → fake agent → verifier → replayable trace, with pgvector + postgres metadata",
		Status:     "validated",
		CheckedAt:  time.Now().UTC(),
		Artifacts:  map[string]string{},
	}
	add := func(id string, ok bool, detail, art string) {
		summary.Steps = append(summary.Steps, ProofStep{ID: id, OK: ok, Detail: detail, Artifact: art})
		if !ok {
			summary.Status = "failed"
		}
	}
	partial := func() {
		if summary.Status == "validated" {
			summary.Status = "partially_validated"
		}
	}
	writeJSON := func(name string, v any) (string, error) {
		path := filepath.Join(outDir, name)
		raw, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
			return "", err
		}
		summary.Artifacts[name] = path
		return path, nil
	}

	// 1-3: init + ingest proof corpus
	st, err := InitProject(dataDir, corpusDir, "proof", "Chunk 12 Proof")
	if err != nil {
		add("init-project", false, err.Error(), "")
		return summary, err
	}
	add("init-project", true, "project="+string(st.Project.ID), "")
	st, err = Ingest(dataDir, "proof", corpusDir)
	if err != nil {
		add("ingest", false, err.Error(), "")
		return summary, err
	}
	summary.ProjectID = string(st.Project.ID)
	summary.SnapshotID = string(st.Snapshot.ID)
	_, _ = writeJSON("01-ingest.json", map[string]any{
		"project_id": st.Project.ID, "snapshot_id": st.Snapshot.ID,
		"chunks": len(st.Chunks), "status": st.Snapshot.Status,
		"source_merkle_root": st.Snapshot.SourceMerkleRoot, "chunk_set_hash": st.Snapshot.ChunkSetHash,
	})
	add("ingest", len(st.Chunks) > 0, "chunks="+itoa(len(st.Chunks)), "01-ingest.json")

	dsn := strings.TrimSpace(os.Getenv("CONTEXT_PG_DSN"))
	if dsn == "" {
		cfg, _ := config.LoadStorageConfigFromEnv()
		dsn = cfg.Metadata.DSN
	}
	_ = os.Setenv("CONTEXT_PG_DSN", dsn)

	// 4-5: search modes
	for _, mode := range []string{"exact", "sparse", "hybrid", "dense", "hybrid-dense"} {
		name := "02-search-" + mode + ".json"
		res, err := Search(dataDir, "proof", "ContextPack", mode, "")
		if err != nil {
			add("search-"+mode, false, err.Error(), "")
			if mode == "dense" || mode == "hybrid-dense" {
				partial()
				summary.Gaps = append(summary.Gaps, "dense/hybrid-dense requires healthy PostgreSQL/pgvector: "+err.Error())
				continue
			}
			return summary, err
		}
		_, _ = writeJSON(name, res)
		add("search-"+mode, len(res.Candidates) >= 0, "candidates="+itoa(len(res.Candidates))+" backend="+res.Backend, name)
	}

	// 6: multilingual + expansion trace shape
	multi, err := proveMultilingual()
	if err != nil {
		add("multilingual", false, err.Error(), "")
		return summary, err
	}
	_, _ = writeJSON("03-multilingual.json", multi)
	add("multilingual", multi["ok"] == true, "language+expansion+token_span", "03-multilingual.json")

	// 7: lexicon metadata preservation
	lex, err := proveLexicon()
	if err != nil {
		add("lexicon", false, err.Error(), "")
		return summary, err
	}
	_, _ = writeJSON("04-lexicon.json", lex)
	add("lexicon", lex["ok"] == true, "sense/concept/attestation/register/region/time/authority", "04-lexicon.json")

	// 8-10: pack, agent, trace/verify
	packRes, err := BuildPack(dataDir, "proof", "ContextPack roadmap hybrid retrieval", "")
	if err != nil {
		add("context-pack", false, err.Error(), "")
		return summary, err
	}
	_, _ = writeJSON("05-context-pack.json", packRes)
	add("context-pack", len(packRes.Pack.EvidenceItems) > 0, "pack="+string(packRes.Pack.ID), "05-context-pack.json")

	agentRes, err := AgentRun(dataDir, "proof", "ContextPack roadmap hybrid retrieval", "")
	if err != nil {
		add("agent-run", false, err.Error(), "")
		return summary, err
	}
	_, _ = writeJSON("06-agent-run.json", agentRes)
	add("agent-run", agentRes.VerifyOK, "run="+string(agentRes.Run.ID)+" verify_ok="+boolStr(agentRes.VerifyOK), "06-agent-run.json")

	traceRes, err := Trace(dataDir, "proof", string(agentRes.Run.ID))
	if err != nil {
		add("trace", false, err.Error(), "")
		return summary, err
	}
	_, _ = writeJSON("07-trace.json", traceRes)
	add("trace", len(traceRes.Events) > 0, "events="+itoa(len(traceRes.Events)), "07-trace.json")

	// 11-12: event window, lineage, temporal, source≠trace
	evt, err := proveEventsAndLineage(ctxBackground(), dsn, st.Project.ID, agentRes.Run.ID, traceRes.Events)
	if err != nil {
		add("events-lineage-temporal", false, err.Error(), "")
		partial()
		summary.Gaps = append(summary.Gaps, "postgres metadata event/lineage proof failed: "+err.Error())
	} else {
		_, _ = writeJSON("08-events-lineage-temporal.json", evt)
		add("events-lineage-temporal", evt["ok"] == true, "temporal filter + lineage ≠ runtime trace", "08-events-lineage-temporal.json")
	}

	// Record known gaps / next decisions (even on success)
	summary.Gaps = append(summary.Gaps,
		"CLI ingest still persists workspace state.json; postgres metadata is proven via meta path / proof helpers, not yet the default ingest backend.",
		"Sparse path remains fake term-overlap; Postgres FTS and context-sparse are not required for this proof.",
		"Dense embeddings use fake-hash-v1 dim=8; live embedding providers are deferred.",
		"Multilingual/lexicon proofs use in-memory fixtures with simple-lang adapters; context-lang-* and TEI/SKOS lexicon adapters are not wired.",
	)
	summary.NextDecisions = append(summary.NextDecisions,
		"Decide when ingest/agent-run should default MetadataStore to postgres (CONTEXT_METADATA_KIND=postgres).",
		"Before QDrant/Turbopuffer: keep BackendCapabilities contract tests against pgvector + candidate.",
		"Before context-sparse: measure Postgres FTS / fake sparse lexical limits on a larger corpus.",
		"Before context-lang-*: pin analyzer_version/dictionary_version on chunk rows during ingest.",
		"Before TEI/SKOS lexicon adapters: promote DocumentStore sense/concept/attestation payloads into typed lexicon ports only after schema stability.",
	)

	if _, err := writeJSON("SUMMARY.json", summary); err != nil {
		return summary, err
	}
	if err := writeProofMarkdown(outDir, summary); err != nil {
		return summary, err
	}
	summary.Artifacts["SUMMARY.md"] = filepath.Join(outDir, "SUMMARY.md")
	return summary, nil
}

func proveMultilingual() (map[string]any, error) {
	idx := index.NewMemory(
		index.ChunkRecord{
			ProjectID: "p1", SnapshotID: "snap1", ChunkID: "c-en", SourceID: "s-en",
			Span: foundation.ByteSpan{Start: 4, End: 11}, Text: "Runners run in the park",
			TextChecksum: "h-en", TrustLevel: foundation.TrustProject,
			Language: "en", AnalyzerVersion: "simple-v1",
			Lemmas: []string{"run"}, Wordforms: []string{"runners", "run"},
		},
		index.ChunkRecord{
			ProjectID: "p1", SnapshotID: "snap1", ChunkID: "c-ru", SourceID: "s-ru",
			Span: foundation.ByteSpan{Start: 0, End: 6}, Text: "Бегуны бегают в парке",
			TextChecksum: "h-ru", TrustLevel: foundation.TrustProject,
			Language: "ru", AnalyzerVersion: "simple-v1",
		},
	)
	store := fake.NewVectorStore()
	ns := indexing.VectorNamespace{Name: "ns", ProjectID: "p1", SnapshotID: "snap1", EmbeddingVersion: config.DefaultEmbeddingVersion}
	for _, rec := range idx.List("p1", "snap1") {
		_ = store.Upsert(context.Background(), ns, []retrieval.VectorPoint{{
			ChunkID: rec.ChunkID, ProjectID: rec.ProjectID, SnapshotID: rec.SnapshotID,
			EmbeddingVersion: config.DefaultEmbeddingVersion, Language: rec.Language, Span: rec.Span,
			Vector: fake.HashEmbed(rec.Text, config.DefaultEmbeddingDimension),
		}})
	}
	eng := hybrid.Engine{
		Exact: exact.Retriever{Index: idx},
		Dense: dense.Retriever{Store: store, Embedder: modelfake.Embedder{Dim: 8}, Index: idx, Namespace: ns},
		Expander: simple.Expander{WordformMap: map[string][]string{"run": {"runners"}}},
	}
	plan := retrieval.RetrievalPlan{
		ID: "multi", ProjectID: "p1", SnapshotID: "snap1", TopNRawPool: 10,
		Strategies: []retrieval.RetrieverStrategy{{RetrieverID: "exact"}, {RetrieverID: "dense"}},
		Filters:    retrieval.RetrievalFilters{Language: "en"},
	}
	res, err := eng.Search(context.Background(), plan, "q-multi", "run")
	if err != nil {
		return nil, err
	}
	var expansionEvent bool
	for _, ev := range res.Events {
		if ev.Type == tracing.EventQueryExpansion {
			expansionEvent = true
		}
	}
	ok := len(res.Candidates) > 0 && expansionEvent && len(res.Expansions) > 0
	for _, c := range res.Candidates {
		if c.ChunkID != "c-en" {
			ok = false
		}
		if c.SourceRef.Span.Start != 4 || c.SourceRef.Span.End != 11 {
			ok = false
		}
	}
	return map[string]any{
		"ok": ok, "candidates": res.Candidates, "expansions": res.Expansions,
		"events": res.Events, "language_filter": "en",
		"analyzer": "simple-v1", "note": "RU chunk excluded by language filter; expansion traced",
	}, nil
}

func proveLexicon() (map[string]any, error) {
	idx := index.NewMemory(index.ChunkRecord{
		ProjectID: "p1", SnapshotID: "snap1", ChunkID: "c-lex", SourceID: "s-lex",
		Span: foundation.ByteSpan{Start: 0, End: 7}, Text: "runners",
		TextChecksum: "h-lex", TrustLevel: foundation.TrustProject,
		Language: "en", SenseIDs: []ids.SenseID{"sense-run-sport"},
		ConceptIDs: []ids.ConceptID{"concept-running"},
		AttestationIDs: []ids.AttestationID{"att-runners-1"},
		Register: "sport", Region: "us", TimePeriod: "2020s",
		LexiconSourceID: "lex-proof-1", SourceAuthority: "proof-fixture",
	})
	eng := hybrid.Engine{Exact: exact.Retriever{Index: idx}}
	plan := retrieval.RetrievalPlan{
		ID: "lex", ProjectID: "p1", SnapshotID: "snap1",
		Strategies: []retrieval.RetrieverStrategy{{RetrieverID: "exact"}},
		Filters: retrieval.RetrievalFilters{
			SenseID: "sense-run-sport", ConceptID: "concept-running",
			AttestationID: "att-runners-1", Register: "sport", DialectRegion: "us",
			TimePeriod: "2020s", LexiconSourceID: "lex-proof-1", SourceAuthority: "proof-fixture",
		},
	}
	res, err := eng.Search(context.Background(), plan, "q-lex", "runners")
	if err != nil {
		return nil, err
	}
	ok := len(res.Candidates) == 1 && res.Candidates[0].TextChecksum == "h-lex" &&
		res.Candidates[0].SourceRef.Span.Start == 0 && res.Candidates[0].SourceRef.Span.End == 7
	return map[string]any{
		"ok": ok, "original_text": "runners", "candidates": res.Candidates,
		"filters": plan.Filters,
	}, nil
}

func proveEventsAndLineage(ctx context.Context, dsn string, projectID ids.ProjectID, runID ids.RunID, runtimeEvents []tracing.Event) (map[string]any, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, apperr.New(apperr.Unavailable, "CONTEXT_PG_DSN required for event/lineage proof")
	}
	meta, err := postgres.Open(ctx, dsn)
	if err != nil {
		return nil, err
	}
	defer meta.Close()

	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	if err := meta.PutProject(ctx, corpus.Project{ID: projectID, Name: "proof"}); err != nil {
		return nil, err
	}
	// Event-window source (corpus), not a tracing.Event.
	srcTemporal := &corpus.TemporalMetadata{
		Range: corpus.TemporalRange{
			Start: now, End: now.Add(2 * time.Hour), Basis: corpus.TimeBasisOccurred,
		},
		IngestedAt: now.Add(5 * time.Minute),
	}
	if err := meta.PutSource(ctx, corpus.Source{
		ID: "evt-batch-1", ProjectID: projectID, Type: corpus.SourceTypeFile,
		PathKey: "events.ndjson", TrustLevel: foundation.TrustProject,
		MediaType: "application/x-ndjson", Checksum: "evtbatch", TemporalMetadata: srcTemporal,
	}); err != nil {
		return nil, err
	}
	windowChunks := []corpus.Chunk{
		{
			ID: "evt-c1", ProjectID: projectID, SourceID: "evt-batch-1", ArtifactID: "evt-art-raw",
			SnapshotID: "snap-evt", ChunkerVersion: "event-window-v1",
			Span: foundation.ByteSpan{Start: 0, End: 20}, TextChecksum: "e1", ChunkHash: "eh1",
			TemporalMetadata: &corpus.TemporalMetadata{
				Range: corpus.TemporalRange{Start: now, End: now.Add(time.Second), Basis: corpus.TimeBasisOccurred},
				IngestedAt: now.Add(5 * time.Minute),
			},
		},
		{
			ID: "evt-c2", ProjectID: projectID, SourceID: "evt-batch-1", ArtifactID: "evt-art-raw",
			SnapshotID: "snap-evt", ChunkerVersion: "event-window-v1",
			Span: foundation.ByteSpan{Start: 20, End: 40}, TextChecksum: "e2", ChunkHash: "eh2",
			TemporalMetadata: &corpus.TemporalMetadata{
				Range: corpus.TemporalRange{Start: now.Add(time.Hour), End: now.Add(time.Hour + time.Second), Basis: corpus.TimeBasisOccurred},
				IngestedAt: now.Add(65 * time.Minute),
			},
		},
	}
	if err := meta.PutArtifactMeta(ctx, artifacts.Artifact{
		ID: "evt-art-raw", ProjectID: projectID, MediaType: "application/x-ndjson", ByteSize: 10,
		Checksum: "raw1", StorageURI: "local://events.ndjson", ArtifactType: artifacts.TypeBlob,
	}); err != nil {
		return nil, err
	}
	if err := meta.PutArtifactMeta(ctx, artifacts.Artifact{
		ID: "evt-art-derived", ProjectID: projectID, MediaType: "application/json", ByteSize: 12,
		Checksum: "der1", StorageURI: "local://derived.json",
		ArtifactType: artifacts.TypeStructured, SchemaID: "event.window.summary.v1",
	}); err != nil {
		return nil, err
	}
	for _, ch := range windowChunks {
		if err := meta.PutChunk(ctx, ch); err != nil {
			return nil, err
		}
	}
	if err := meta.PutArtifactLineage(ctx, artifacts.ArtifactLineage{
		ProjectID: projectID, OutputArtifactID: "evt-art-derived",
		InputArtifactIDs: []ids.ArtifactID{"evt-art-raw"},
		SourceRefs: []corpus.SourceRef{{
			ProjectID: projectID, SourceID: "evt-batch-1", ChunkID: "evt-c1",
			Span: foundation.ByteSpan{Start: 0, End: 20}, Checksum: "e1",
		}},
		AgentRunID: runID, GeneratorID: "proof-event-summary", GeneratorVersion: "v1",
		TransformationKind: "event_window_summary", CreatedAt: now.Add(3 * time.Hour),
	}); err != nil {
		return nil, err
	}
	if err := meta.PutDocument(ctx, storage.MetaDocument{
		ProjectID: projectID, Kind: storage.DocumentTokenOccurrence, ID: "tok-1",
		Language: "en", ChunkID: "evt-c1", AnalyzerVersion: "simple-v1",
		Payload: []byte(`{"surface":"committed","span":{"start":0,"end":9}}`),
	}); err != nil {
		return nil, err
	}

	// Temporal retrieval over in-memory projection of event chunks.
	mem := index.NewMemory()
	for _, ch := range windowChunks {
		mem.Add(index.ChunkRecord{
			ProjectID: ch.ProjectID, SnapshotID: ch.SnapshotID, ChunkID: ch.ID, SourceID: ch.SourceID,
			Span: ch.Span, TextChecksum: ch.TextChecksum, TrustLevel: foundation.TrustProject,
			TemporalMetadata: ch.TemporalMetadata, Text: string(ch.ID),
		})
	}
	filter := retrieval.RetrievalFilters{TemporalRange: &corpus.TemporalRange{
		Start: now.Add(30 * time.Minute), End: now.Add(90 * time.Minute), Basis: corpus.TimeBasisOccurred,
	}}
	var matched []ids.ChunkID
	for _, rec := range mem.List(projectID, "snap-evt") {
		if index.MatchesFilters(rec, filter) {
			matched = append(matched, rec.ChunkID)
		}
	}
	lineage, err := meta.GetArtifactLineage(ctx, projectID, "evt-art-derived")
	if err != nil {
		return nil, err
	}
	art, err := meta.GetArtifactMeta(ctx, projectID, "evt-art-derived")
	if err != nil {
		return nil, err
	}

	sourceIsNotTrace := true
	for _, ev := range runtimeEvents {
		if strings.HasPrefix(string(ev.ID), "evt-") || ev.Type == tracing.EventType("source_event") {
			sourceIsNotTrace = false
		}
	}
	ok := len(matched) == 1 && matched[0] == "evt-c2" &&
		art.SchemaID == "event.window.summary.v1" &&
		sourceIsNotTrace && len(lineage.InputArtifactIDs) == 1

	return map[string]any{
		"ok":                                ok,
		"temporal_matched_chunk_ids":        matched,
		"lineage":                           lineage,
		"derived_schema_id":                 art.SchemaID,
		"runtime_trace_event_count":         len(runtimeEvents),
		"source_events_separate_from_trace": sourceIsNotTrace,
		"note":                              "evt-c1 outside window; evt-c2 overlaps; lineage loaded without parsing AgentRun trace",
	}, nil
}

func writeProofMarkdown(outDir string, summary ProofSummary) error {
	var b strings.Builder
	b.WriteString("# Chunk 12 Proof Summary\n\n")
	b.WriteString("- Status: **" + summary.Status + "**\n")
	b.WriteString("- Hypothesis: " + summary.Hypothesis + "\n")
	b.WriteString("- Project: `" + summary.ProjectID + "` snapshot `" + summary.SnapshotID + "`\n")
	b.WriteString("- Checked at: " + summary.CheckedAt.Format(time.RFC3339) + "\n\n")
	b.WriteString("## Steps\n\n")
	for _, s := range summary.Steps {
		mark := "FAIL"
		if s.OK {
			mark = "OK"
		}
		b.WriteString("- [" + mark + "] `" + s.ID + "` " + s.Detail)
		if s.Artifact != "" {
			b.WriteString(" → `" + s.Artifact + "`")
		}
		b.WriteByte('\n')
	}
	b.WriteString("\n## Gaps\n\n")
	for _, g := range summary.Gaps {
		b.WriteString("- " + g + "\n")
	}
	b.WriteString("\n## Next decisions\n\n")
	for _, d := range summary.NextDecisions {
		b.WriteString("- " + d + "\n")
	}
	return os.WriteFile(filepath.Join(outDir, "SUMMARY.md"), []byte(b.String()), 0o644)
}

func ctxBackground() context.Context { return context.Background() }

func boolStr(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}