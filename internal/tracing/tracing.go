// Package tracing defines append-only runtime events and the recorder port (ADR-0006).
package tracing

import (
	"context"
	"fmt"
	"time"

	"github.com/fastygo/context/internal/ids"
)

// EventType classifies append-only trace records.
type EventType string

const (
	EventRunStarted          EventType = "run_started"
	EventRunFinished         EventType = "run_finished"
	EventRetrievalQuery      EventType = "retrieval_query"
	EventRetrievalCandidates EventType = "retrieval_candidates"
	EventQueryExpansion      EventType = "query_expansion"
	EventSenseMapping        EventType = "sense_mapping"
	EventConceptMapping      EventType = "concept_mapping"
	EventAttestationUsed     EventType = "attestation_used"
	EventContextPackBuilt    EventType = "context_pack_built"
	EventContextPackVerified EventType = "context_pack_verified"
	EventModelCall           EventType = "model_call"
	EventToolDecision        EventType = "tool_decision"
	EventToolExecuted        EventType = "tool_executed"
	EventSnapshotTransition  EventType = "snapshot_transition"
	EventPolicyDecision      EventType = "policy_decision"
)

// Event is one append-only, replayable trace record.
type Event struct {
	ID        ids.TraceEventID
	ProjectID ids.ProjectID
	RunID     ids.RunID
	Type      EventType
	Timestamp time.Time
	Payload   map[string]string

	// Version pins for linguistic / lexicographic reproducibility.
	AnalyzerVersion     string
	DictionaryVersion   string
	FeatureScheme       string
	QueryExpansionVer   string
	SenseMappingVersion string
	ConceptMappingVer   string
	AttestationVersion  string
	SnapshotID          ids.SnapshotID
}

func (e Event) Validate() error {
	if err := e.ID.Validate(); err != nil {
		return err
	}
	if err := e.ProjectID.Validate(); err != nil {
		return err
	}
	if e.Type == "" {
		return fmt.Errorf("trace event type: empty")
	}
	if e.Timestamp.IsZero() {
		return fmt.Errorf("trace event timestamp: zero")
	}
	return nil
}

// Recorder appends immutable events for later replay.
type Recorder interface {
	Append(ctx context.Context, event Event) error
	ListByRun(ctx context.Context, projectID ids.ProjectID, runID ids.RunID) ([]Event, error)
}
