package pack

import (
	"context"
	"fmt"
	"time"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/tracing"
)

// FlagCode classifies verifier findings.
type FlagCode string

const (
	FlagMissingSourceRef     FlagCode = "missing_source_ref"
	FlagMissingChecksum      FlagCode = "missing_checksum"
	FlagUnsupportedSenseFact FlagCode = "unsupported_sense_as_fact"
	FlagUnsupportedConcept   FlagCode = "unsupported_concept_as_fact"
	FlagUnsupportedInference FlagCode = "unsupported_model_inference_as_fact"
	FlagChecksumMismatch     FlagCode = "pack_checksum_mismatch"
	FlagReplaySurfaceMismatch FlagCode = "replay_surface_mismatch"
)

// Flag is one verifier finding against an evidence item or the pack.
type Flag struct {
	Code      FlagCode
	EvidenceID string
	Message   string
}

// VerifyRequest configures verification and optional tracing.
type VerifyRequest struct {
	Pack                 retrieval.ContextPack
	TreatAsFactual       map[string]bool // evidence IDs claimed as facts by the caller
	SenseHasAuthority    map[string]bool // evidence IDs with lexicon authority
	ToolAuthoritative    map[string]bool
	Recorder             tracing.Recorder
	RunID                ids.RunID
}

// VerifyResult is the baseline verifier output.
type VerifyResult struct {
	OK     bool
	Flags  []Flag
	PackOK bool // checksum matches recomputation
}

// Verifier checks source-backed factual evidence rules (ADR-0020).
type Verifier struct{}

func (Verifier) Verify(ctx context.Context, req VerifyRequest) (VerifyResult, error) {
	if err := ctx.Err(); err != nil {
		return VerifyResult{}, err
	}
	var out VerifyResult
	recomputed := Checksum(req.Pack)
	if recomputed != req.Pack.Checksum {
		out.Flags = append(out.Flags, Flag{
			Code:    FlagChecksumMismatch,
			Message: fmt.Sprintf("checksum want %s got %s", recomputed, req.Pack.Checksum),
		})
	} else {
		out.PackOK = true
	}

	for _, item := range req.Pack.EvidenceItems {
		asFact := req.TreatAsFactual[item.ID]
		if !asFact {
			// Default: source_text and attestation are factual; others are not unless claimed.
			asFact = item.Class == foundation.EvidenceSourceText || item.Class == foundation.EvidenceAttestation
		}
		if !asFact {
			continue
		}
		switch item.Class {
		case foundation.EvidenceSenseClaim:
			if !req.SenseHasAuthority[item.ID] {
				out.Flags = append(out.Flags, Flag{
					Code: FlagUnsupportedSenseFact, EvidenceID: item.ID,
					Message: "sense_claim used as fact without lexicon authority",
				})
			}
		case foundation.EvidenceConceptMapping:
			out.Flags = append(out.Flags, Flag{
				Code: FlagUnsupportedConcept, EvidenceID: item.ID,
				Message: "concept_mapping cannot justify factual claims",
			})
		case foundation.EvidenceModelInference:
			out.Flags = append(out.Flags, Flag{
				Code: FlagUnsupportedInference, EvidenceID: item.ID,
				Message: "model_inference cannot justify factual claims",
			})
		case foundation.EvidenceToolOutput:
			if !req.ToolAuthoritative[item.ID] {
				out.Flags = append(out.Flags, Flag{
					Code: FlagUnsupportedInference, EvidenceID: item.ID,
					Message: "tool_output used as fact without authoritative flag",
				})
			}
		}
		if err := item.SourceRef.Validate(); err != nil {
			out.Flags = append(out.Flags, Flag{
				Code: FlagMissingSourceRef, EvidenceID: item.ID, Message: err.Error(),
			})
			continue
		}
		if item.SourceRef.Checksum == "" {
			out.Flags = append(out.Flags, Flag{
				Code: FlagMissingChecksum, EvidenceID: item.ID, Message: "source checksum empty",
			})
		}
	}

	out.OK = len(out.Flags) == 0
	if req.Recorder != nil {
		status := "ok"
		if !out.OK {
			status = "flagged"
		}
		ev := tracing.Event{
			ID:        ids.TraceEventID(string(req.Pack.ID) + ":verified"),
			ProjectID: req.Pack.ProjectID,
			RunID:     req.RunID,
			Type:      tracing.EventContextPackVerified,
			Timestamp: time.Now().UTC(),
			Payload: map[string]string{
				"pack_id": string(req.Pack.ID),
				"status":  status,
				"flags":   fmt.Sprintf("%d", len(out.Flags)),
			},
		}
		if err := req.Recorder.Append(ctx, ev); err != nil {
			return VerifyResult{}, err
		}
	}
	return out, nil
}

// SurfaceStore loads original surfaces for replay.
type SurfaceStore interface {
	GetSurface(ctx context.Context, projectID ids.ProjectID, sourceID ids.SourceID, checksum foundation.ChecksumHex) (surface string, err error)
}

// Replay reloads evidence surfaces by source identity and verifies checksum continuity.
func Replay(ctx context.Context, pack retrieval.ContextPack, store SurfaceStore) (retrieval.ContextPack, error) {
	if err := ctx.Err(); err != nil {
		return retrieval.ContextPack{}, err
	}
	if store == nil {
		return retrieval.ContextPack{}, apperr.New(apperr.Validation, "surface store required")
	}
	out := pack
	out.EvidenceItems = make([]retrieval.EvidenceItem, len(pack.EvidenceItems))
	copy(out.EvidenceItems, pack.EvidenceItems)
	for i, item := range out.EvidenceItems {
		if item.Class == foundation.EvidenceInstruction || item.Class == foundation.EvidencePolicy {
			continue
		}
		surface, err := store.GetSurface(ctx, pack.ProjectID, item.SourceRef.SourceID, item.SourceRef.Checksum)
		if err != nil {
			return retrieval.ContextPack{}, err
		}
		if surface != item.Surface && item.Surface != "" {
			return retrieval.ContextPack{}, apperr.New(apperr.Conflict, string(FlagReplaySurfaceMismatch)+": "+item.ID)
		}
		out.EvidenceItems[i].Surface = surface
	}
	out.Checksum = Checksum(out)
	return out, nil
}

// MemorySurfaces is a test double SurfaceStore.
type MemorySurfaces map[string]string // key: project|source|checksum

func surfaceKey(projectID ids.ProjectID, sourceID ids.SourceID, checksum foundation.ChecksumHex) string {
	return string(projectID) + "\x00" + string(sourceID) + "\x00" + string(checksum)
}

func (m MemorySurfaces) GetSurface(ctx context.Context, projectID ids.ProjectID, sourceID ids.SourceID, checksum foundation.ChecksumHex) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s, ok := m[surfaceKey(projectID, sourceID, checksum)]
	if !ok {
		return "", apperr.New(apperr.NotFound, "surface not found")
	}
	return s, nil
}
