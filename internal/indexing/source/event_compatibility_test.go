package source_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing/source"
)

type fixtureEventAdapter struct{}

func (fixtureEventAdapter) List(context.Context, ids.ProjectID, string) ([]source.Discovered, error) {
	return []source.Discovered{{
		RelativePath: "events.ndjson",
		Bytes:        []byte("{\"event_id\":\"event-a\"}\n"),
		MediaType:    "application/x-ndjson",
	}}, nil
}

func (fixtureEventAdapter) EventSourceCompatibility() source.EventSourceCompatibilityDescriptor {
	return validDescriptor()
}

func validDescriptor() source.EventSourceCompatibilityDescriptor {
	return source.EventSourceCompatibilityDescriptor{
		AdapterID:                    "fixture-events",
		AdapterVersion:               "v1",
		SchemaID:                     "fixture.event",
		SchemaVersion:                "fixture.event.v1",
		ProducerID:                   "fixture-producer",
		ProducerVersion:              "v1",
		TimePrecision:                "nanosecond",
		ClockSource:                  "producer",
		Ordering:                     source.EventOrderingBestEffort,
		LateEventHandling:            source.LateEventNewVersion,
		StableEventIdentity:          true,
		IncludesOccurredTime:         true,
		IncludesIngestedTime:         true,
		AssignsTrust:                 true,
		IdempotentIngest:             true,
		DeterministicBatchChecksums:  true,
		DeterministicWindowChecksums: true,
	}
}

func TestEventSourceCompatibilityDescriptor(t *testing.T) {
	t.Parallel()
	var adapter source.EventAdapter = fixtureEventAdapter{}
	if err := adapter.EventSourceCompatibility().Validate(); err != nil {
		t.Fatal(err)
	}

	invalid := validDescriptor()
	invalid.IdempotentIngest = false
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected non-idempotent event source to be incompatible")
	}
}

func TestDuplicateEventFixtureIngestIsIdempotent(t *testing.T) {
	t.Parallel()
	once := []source.EventIdentity{
		{StableID: "event-b", PayloadChecksum: "bb"},
		{StableID: "event-a", PayloadChecksum: "aa"},
	}
	duplicatedAndReordered := []source.EventIdentity{
		{StableID: "event-a", PayloadChecksum: "aa"},
		{StableID: "event-b", PayloadChecksum: "bb"},
		{StableID: "event-a", PayloadChecksum: "aa"},
	}
	originalOrder := append([]source.EventIdentity(nil), duplicatedAndReordered...)
	firstChecksum, err := source.EventBatchChecksum(once)
	if err != nil {
		t.Fatal(err)
	}
	secondChecksum, err := source.EventBatchChecksum(duplicatedAndReordered)
	if err != nil {
		t.Fatal(err)
	}
	if firstChecksum != secondChecksum {
		t.Fatalf("duplicate delivery changed checksum: %s != %s", firstChecksum, secondChecksum)
	}
	canonical, err := source.CanonicalEventIdentities(duplicatedAndReordered)
	if err != nil {
		t.Fatal(err)
	}
	if len(canonical) != 2 || canonical[0].StableID != "event-a" {
		t.Fatalf("canonical events=%#v", canonical)
	}
	if !reflect.DeepEqual(duplicatedAndReordered, originalOrder) {
		t.Fatalf("canonicalization mutated caller slice: %#v", duplicatedAndReordered)
	}
}

func TestConflictingDuplicateEventIsRejected(t *testing.T) {
	t.Parallel()
	_, err := source.CanonicalEventIdentities([]source.EventIdentity{
		{StableID: "event-a", PayloadChecksum: "aa"},
		{StableID: "event-a", PayloadChecksum: "changed"},
	})
	if err == nil {
		t.Fatal("expected stable event id conflict")
	}
}

func TestEventWindowChecksumIncludesHalfOpenBoundaries(t *testing.T) {
	t.Parallel()
	base := time.Unix(100, 0).UTC()
	window := corpus.TemporalRange{
		Start: base,
		End:   base.Add(time.Minute),
		Basis: corpus.TimeBasisOccurred,
	}
	events := []source.EventIdentity{{StableID: "event-a", PayloadChecksum: "aa"}}
	first, err := source.EventWindowChecksum(window, events)
	if err != nil {
		t.Fatal(err)
	}
	window.End = window.End.Add(time.Nanosecond)
	second, err := source.EventWindowChecksum(window, events)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("changing the exclusive end boundary must change the window checksum")
	}
}
