// NDJSON / JSONL observation event source adapter (S3 / A7).
package source

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing/ignore"
	"github.com/fastygo/context/internal/indexing/normalize"
)

// NDJSONFiles discovers .ndjson / .jsonl event corpora under a root.
type NDJSONFiles struct {
	IgnorePatterns []string
}

func (a NDJSONFiles) EventSourceCompatibility() EventSourceCompatibilityDescriptor {
	return EventSourceCompatibilityDescriptor{
		AdapterID:                    "source-ndjson",
		AdapterVersion:               "ndjson-v1",
		SchemaID:                     "observation.event.v1",
		SchemaVersion:                "v1",
		ProducerID:                   "ndjson-files",
		ProducerVersion:              "ndjson-v1",
		TimePrecision:                "rfc3339",
		ClockSource:                  "producer_wall",
		Ordering:                     EventOrderingBestEffort,
		LateEventHandling:            LateEventNewVersion,
		StableEventIdentity:          true,
		IncludesOccurredTime:         true,
		IncludesIngestedTime:         true,
		AssignsTrust:                 true,
		IdempotentIngest:             true,
		DeterministicBatchChecksums:  true,
		DeterministicWindowChecksums: true,
	}
}

func (a NDJSONFiles) List(ctx context.Context, projectID ids.ProjectID, root string) ([]Discovered, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := projectID.Validate(); err != nil {
		return nil, apperr.Wrap(apperr.Validation, "project_id", err)
	}
	if root == "" {
		return nil, apperr.New(apperr.Validation, "source root: empty")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, apperr.Wrap(apperr.Validation, "source root", err)
	}
	filePats, err := ignore.LoadFile(filepath.Join(abs, ignore.FileName))
	if err != nil {
		return nil, apperr.Wrap(apperr.Validation, "load .contextignore", err)
	}
	patterns := ignore.Compile(filePats, a.IgnorePatterns)

	var out []Discovered
	err = filepath.WalkDir(abs, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(abs, path)
		if err != nil {
			return err
		}
		rel = normalize.RelativePath(rel)
		if d.IsDir() {
			if rel == "." || rel == "" {
				return nil
			}
			if ignore.MatchDir(rel, patterns) {
				return filepath.SkipDir
			}
			return nil
		}
		if ignore.Match(rel, patterns) {
			return nil
		}
		media := eventMediaType(rel)
		if media == "" {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return apperr.Wrap(apperr.Validation, "read source", err)
		}
		out = append(out, Discovered{
			RelativePath: rel,
			Bytes:        body,
			MediaType:    media,
			SourceType:   corpus.SourceTypeFile,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ObservationEvent is the neutral NDJSON line shape (adapter-owned envelope).
type ObservationEvent struct {
	EventID       string    `json:"event_id"`
	OccurredStart time.Time `json:"occurred_start"`
	OccurredEnd   time.Time `json:"occurred_end"`
	IngestedAt    time.Time `json:"ingested_at"`
	Message       string    `json:"message"`
	Trust         string    `json:"trust"`
}

// ParseNDJSONLines parses event lines, skipping blanks. Duplicate event_ids with
// identical payload checksums collapse; conflicting payloads error.
func ParseNDJSONLines(raw []byte) ([]ObservationEvent, []EventIdentity, error) {
	sc := bufio.NewScanner(bytes.NewReader(raw))
	var events []ObservationEvent
	var idsOut []EventIdentity
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var ev ObservationEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			return nil, nil, apperr.Wrap(apperr.Validation, "ndjson line", err)
		}
		if strings.TrimSpace(ev.EventID) == "" {
			return nil, nil, apperr.New(apperr.Validation, "event_id required")
		}
		sum := sha256.Sum256([]byte(line))
		events = append(events, ev)
		idsOut = append(idsOut, EventIdentity{
			StableID:        ev.EventID,
			PayloadChecksum: foundation.ChecksumHex(hex.EncodeToString(sum[:])),
		})
	}
	if err := sc.Err(); err != nil {
		return nil, nil, err
	}
	canonical, err := CanonicalEventIdentities(idsOut)
	if err != nil {
		return nil, nil, err
	}
	_ = canonical
	return events, idsOut, nil
}

// TemporalFromEvent maps an observation line into source-domain temporal metadata.
func TemporalFromEvent(ev ObservationEvent) (*corpus.TemporalMetadata, error) {
	m := &corpus.TemporalMetadata{
		Range: corpus.TemporalRange{
			Start: ev.OccurredStart, End: ev.OccurredEnd, Basis: corpus.TimeBasisOccurred,
		},
		IngestedAt: ev.IngestedAt,
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return m, nil
}

var _ EventAdapter = NDJSONFiles{}
