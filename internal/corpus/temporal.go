package corpus

import (
	"fmt"
	"strings"
	"time"
)

// TimeBasis identifies which source-domain clock a temporal range describes.
// The standard values cover common event corpora; adapters may use a
// namespaced value such as "adapter:recorded" when none applies.
type TimeBasis string

const (
	TimeBasisOccurred  TimeBasis = "occurred"
	TimeBasisObserved  TimeBasis = "observed"
	TimeBasisEffective TimeBasis = "effective"
)

// Validate accepts standard bases and explicit adapter-namespaced extensions.
func (b TimeBasis) Validate() error {
	switch b {
	case TimeBasisOccurred, TimeBasisObserved, TimeBasisEffective:
		return nil
	}
	value := string(b)
	if strings.HasPrefix(value, "adapter:") && len(value) > len("adapter:") &&
		!strings.ContainsAny(value, " \t\n\r") {
		return nil
	}
	if value == "" {
		return fmt.Errorf("time_basis: empty")
	}
	return fmt.Errorf("time_basis: unsupported %q", value)
}

// TemporalRange is a half-open source-time interval [Start, End).
type TemporalRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Basis TimeBasis `json:"basis"`
}

// Validate requires a non-empty interval and an explicit time basis.
func (r TemporalRange) Validate() error {
	if r.Start.IsZero() {
		return fmt.Errorf("temporal_range start: zero")
	}
	if r.End.IsZero() {
		return fmt.Errorf("temporal_range end: zero")
	}
	if !r.Start.Before(r.End) {
		return fmt.Errorf("temporal_range: start must be before end")
	}
	return r.Basis.Validate()
}

// Overlaps reports half-open overlap on the same time basis. Adjacent ranges
// do not overlap.
func (r TemporalRange) Overlaps(other TemporalRange) bool {
	if r.Validate() != nil || other.Validate() != nil || r.Basis != other.Basis {
		return false
	}
	return r.Start.Before(other.End) && other.Start.Before(r.End)
}

// TemporalMetadata carries source-domain time separately from ingestion time.
type TemporalMetadata struct {
	Range      TemporalRange `json:"range"`
	IngestedAt time.Time     `json:"ingested_at"`
}

func (m TemporalMetadata) Validate() error {
	if err := m.Range.Validate(); err != nil {
		return err
	}
	if m.IngestedAt.IsZero() {
		return fmt.Errorf("temporal_metadata ingested_at: zero")
	}
	return nil
}
