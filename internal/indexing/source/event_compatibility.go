package source

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
)

// EventOrdering describes the ordering guarantee exposed by an event source.
type EventOrdering string

const (
	EventOrderingTotal      EventOrdering = "total"
	EventOrderingPartition  EventOrdering = "partition"
	EventOrderingBestEffort EventOrdering = "best_effort"
	EventOrderingNone       EventOrdering = "none"
)

func (o EventOrdering) Validate() error {
	switch o {
	case EventOrderingTotal, EventOrderingPartition, EventOrderingBestEffort, EventOrderingNone:
		return nil
	default:
		return fmt.Errorf("event ordering: unsupported %q", o)
	}
}

// LateEventHandling declares how late or out-of-order source events are handled.
type LateEventHandling string

const (
	LateEventReject     LateEventHandling = "reject"
	LateEventNewVersion LateEventHandling = "new_source_version"
)

func (h LateEventHandling) Validate() error {
	switch h {
	case LateEventReject, LateEventNewVersion:
		return nil
	default:
		return fmt.Errorf("late event handling: unsupported %q", h)
	}
}

// EventSourceCompatibilityDescriptor declares the minimum guarantees required
// from a time-oriented SourceAdapter. Payload schemas remain adapter-owned.
type EventSourceCompatibilityDescriptor struct {
	AdapterID                    string
	AdapterVersion               string
	SchemaID                     string
	SchemaVersion                string
	ProducerID                   string
	ProducerVersion              string
	TimePrecision                string
	ClockSource                  string
	Ordering                     EventOrdering
	LateEventHandling            LateEventHandling
	StableEventIdentity          bool
	IncludesOccurredTime         bool
	IncludesIngestedTime         bool
	AssignsTrust                 bool
	IdempotentIngest             bool
	DeterministicBatchChecksums  bool
	DeterministicWindowChecksums bool
}

// Validate rejects descriptors that cannot preserve deterministic event-source
// identity, temporal metadata, trust, or replayable source versions.
func (d EventSourceCompatibilityDescriptor) Validate() error {
	required := []struct {
		name  string
		value string
	}{
		{"adapter_id", d.AdapterID},
		{"adapter_version", d.AdapterVersion},
		{"schema_id", d.SchemaID},
		{"schema_version", d.SchemaVersion},
		{"producer_id", d.ProducerID},
		{"producer_version", d.ProducerVersion},
		{"time_precision", d.TimePrecision},
		{"clock_source", d.ClockSource},
	}
	for _, field := range required {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("event source compatibility %s: empty", field.name)
		}
	}
	if err := d.Ordering.Validate(); err != nil {
		return err
	}
	if err := d.LateEventHandling.Validate(); err != nil {
		return err
	}
	capabilities := []struct {
		name      string
		supported bool
	}{
		{"stable_event_identity", d.StableEventIdentity},
		{"includes_occurred_time", d.IncludesOccurredTime},
		{"includes_ingested_time", d.IncludesIngestedTime},
		{"assigns_trust", d.AssignsTrust},
		{"idempotent_ingest", d.IdempotentIngest},
		{"deterministic_batch_checksums", d.DeterministicBatchChecksums},
		{"deterministic_window_checksums", d.DeterministicWindowChecksums},
	}
	for _, capability := range capabilities {
		if !capability.supported {
			return fmt.Errorf("event source compatibility %s: required", capability.name)
		}
	}
	return nil
}

// EventAdapter is a SourceAdapter that declares event-corpus compatibility.
// Runtime tracing.Event is intentionally absent from this contract.
type EventAdapter interface {
	Adapter
	EventSourceCompatibility() EventSourceCompatibilityDescriptor
}

// EventIdentity is the neutral identity material used to prove idempotent batch
// ingest without defining an adapter payload schema.
type EventIdentity struct {
	StableID        string
	PayloadChecksum foundation.ChecksumHex
}

// CanonicalEventIdentities sorts by stable id, collapses exact duplicates, and
// rejects conflicting payloads for the same id.
func CanonicalEventIdentities(events []EventIdentity) ([]EventIdentity, error) {
	canonical := append([]EventIdentity(nil), events...)
	for _, event := range canonical {
		if strings.TrimSpace(event.StableID) == "" {
			return nil, fmt.Errorf("event stable_id: empty")
		}
		if err := event.PayloadChecksum.Validate(); err != nil {
			return nil, fmt.Errorf("event %q payload checksum: %w", event.StableID, err)
		}
	}
	sort.Slice(canonical, func(i, j int) bool {
		if canonical[i].StableID == canonical[j].StableID {
			return canonical[i].PayloadChecksum < canonical[j].PayloadChecksum
		}
		return canonical[i].StableID < canonical[j].StableID
	})
	out := canonical[:0]
	for _, event := range canonical {
		if len(out) == 0 || out[len(out)-1].StableID != event.StableID {
			out = append(out, event)
			continue
		}
		if out[len(out)-1].PayloadChecksum != event.PayloadChecksum {
			return nil, fmt.Errorf("event %q: conflicting payload checksum", event.StableID)
		}
	}
	return out, nil
}

// EventBatchChecksum hashes the canonical event identity set. Input order and
// duplicate delivery do not affect the result.
func EventBatchChecksum(events []EventIdentity) (foundation.ChecksumHex, error) {
	canonical, err := CanonicalEventIdentities(events)
	if err != nil {
		return "", err
	}
	if len(canonical) == 0 {
		return "", fmt.Errorf("event batch: empty")
	}
	h := sha256.New()
	_, _ = h.Write([]byte("context/event-batch/v1"))
	writeEventIdentities(h, canonical)
	return foundation.ChecksumHex(hex.EncodeToString(h.Sum(nil))), nil
}

// EventWindowChecksum hashes a validated temporal window and its canonical
// event identity set. Changing either boundary or basis changes the checksum.
func EventWindowChecksum(window corpus.TemporalRange, events []EventIdentity) (foundation.ChecksumHex, error) {
	if err := window.Validate(); err != nil {
		return "", fmt.Errorf("event window: %w", err)
	}
	canonical, err := CanonicalEventIdentities(events)
	if err != nil {
		return "", err
	}
	if len(canonical) == 0 {
		return "", fmt.Errorf("event window: empty")
	}
	h := sha256.New()
	_, _ = h.Write([]byte("context/event-window/v1"))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(window.Start.UTC().Format(time.RFC3339Nano)))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(window.End.UTC().Format(time.RFC3339Nano)))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(window.Basis))
	writeEventIdentities(h, canonical)
	return foundation.ChecksumHex(hex.EncodeToString(h.Sum(nil))), nil
}

func writeEventIdentities(h interface{ Write([]byte) (int, error) }, canonical []EventIdentity) {
	for _, event := range canonical {
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(event.StableID))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(event.PayloadChecksum))
	}
}
