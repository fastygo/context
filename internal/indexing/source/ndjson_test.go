package source_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing/source"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/exact"
	"github.com/fastygo/context/internal/retrieval/index"
)

func TestNDJSONIdempotentBatchAndTemporalFilter(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"event_id":"evt-001","occurred_start":"2026-07-13T10:00:00Z","occurred_end":"2026-07-13T10:00:01Z","ingested_at":"2026-07-13T10:05:00Z","message":"index snapshot committed","trust":"project"}
{"event_id":"evt-002","occurred_start":"2026-07-13T11:00:00Z","occurred_end":"2026-07-13T11:00:01Z","ingested_at":"2026-07-13T11:05:00Z","message":"dense vectors upserted","trust":"project"}
`)
	if b, err := os.ReadFile(filepath.Join("..", "..", "..", ".proofs", "corpus", "events.ndjson")); err == nil {
		raw = b
	}
	events, ids1, err := source.ParseNDJSONLines(raw)
	if err != nil {
		t.Fatal(err)
	}
	_, ids2, err := source.ParseNDJSONLines(append(append([]byte{}, raw...), raw...))
	if err != nil {
		t.Fatal(err)
	}
	sum1, err := source.EventBatchChecksum(ids1)
	if err != nil {
		t.Fatal(err)
	}
	sum2, err := source.EventBatchChecksum(ids2)
	if err != nil {
		t.Fatal(err)
	}
	if sum1 != sum2 {
		t.Fatalf("idempotent re-ingest checksum drift: %s vs %s", sum1, sum2)
	}
	if err := (source.NDJSONFiles{}).EventSourceCompatibility().Validate(); err != nil {
		t.Fatal(err)
	}

	recs := make([]index.ChunkRecord, 0, len(events))
	for _, ev := range events {
		tm, err := source.TemporalFromEvent(ev)
		if err != nil {
			t.Fatal(err)
		}
		end := uint64(len(ev.Message))
		if end == 0 {
			end = 1
		}
		recs = append(recs, index.ChunkRecord{
			ProjectID: "p_evt", SnapshotID: "snap_evt",
			ChunkID: ids.ChunkID(ev.EventID), SourceID: "s_evt",
			Span: foundation.ByteSpan{Start: 0, End: end},
			Text: ev.Message, TextChecksum: foundation.ChecksumHex("h-" + ev.EventID),
			TrustLevel: foundation.TrustProject, TemporalMetadata: tm, Language: "en",
		})
	}
	idx := index.NewMemory(recs...)
	t0 := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	plan := retrieval.RetrievalPlan{
		ID: "p", ProjectID: "p_evt", SnapshotID: "snap_evt", TopNRawPool: 10,
		Strategies: []retrieval.RetrieverStrategy{{RetrieverID: "exact"}},
		Filters: retrieval.RetrievalFilters{
			TemporalRange: &corpus.TemporalRange{
				Start: t0, End: t0.Add(90 * time.Minute), Basis: corpus.TimeBasisOccurred,
			},
		},
	}
	cands, err := (exact.Retriever{Index: idx}).Retrieve(context.Background(), plan, "index")
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 1 || cands[0].ChunkID != "evt-001" {
		t.Fatalf("temporal window want evt-001 only, got %#v", cands)
	}
}

func TestNDJSONFilesList(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "events.ndjson")
	body := `{"event_id":"e1","occurred_start":"2026-07-13T10:00:00Z","occurred_end":"2026-07-13T10:00:01Z","ingested_at":"2026-07-13T10:05:00Z","message":"hi","trust":"project"}` + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := (source.NDJSONFiles{}).List(context.Background(), "p1", dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].MediaType != "application/x-ndjson" {
		t.Fatalf("got %#v", got)
	}
}
