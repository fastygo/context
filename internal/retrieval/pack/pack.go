package pack

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/tracing"
)

const BudgetEstimatorVersion = "chars-div4-v1"

// DraftItem is one candidate ready for packing with evidence class metadata.
type DraftItem struct {
	Candidate         retrieval.Candidate
	Surface           string
	Class             foundation.EvidenceClass
	Required          bool
	LexiconAuthority  bool
	AuthoritativeTool bool
	Summary           string
	ID                string
}

// BuildRequest is the input to ContextPack construction.
type BuildRequest struct {
	PackID                   ids.PackID
	ProjectID                ids.ProjectID
	TaskID                   ids.TaskID
	PlanID                   ids.PlanID
	Purpose                  string
	Focus                    retrieval.FocusProfile
	Instructions             []string
	PolicyRefs               []ids.PolicyID
	VerificationRequirements []string
	Items                    []DraftItem
	Recorder                 tracing.Recorder
	RunID                    ids.RunID
}

// Builder constructs budget-aware ContextPacks.
type Builder struct{}

// Build creates a ContextPack with deterministic trimming and checksum.
func (Builder) Build(ctx context.Context, req BuildRequest) (retrieval.ContextPack, error) {
	if err := ctx.Err(); err != nil {
		return retrieval.ContextPack{}, err
	}
	if err := req.PackID.Validate(); err != nil {
		return retrieval.ContextPack{}, apperr.Wrap(apperr.Validation, "pack_id", err)
	}
	if err := req.ProjectID.Validate(); err != nil {
		return retrieval.ContextPack{}, apperr.Wrap(apperr.Validation, "project_id", err)
	}
	if err := req.PlanID.Validate(); err != nil {
		return retrieval.ContextPack{}, apperr.Wrap(apperr.Validation, "plan_id", err)
	}
	if err := req.Focus.Validate(); err != nil {
		return retrieval.ContextPack{}, apperr.Wrap(apperr.Validation, "focus_profile", err)
	}

	budget := req.Focus.ContextBudget
	if budget.BudgetEstimatorVersion == "" {
		budget.BudgetEstimatorVersion = BudgetEstimatorVersion
	}
	if budget.RejectScoreFloor == 0 {
		budget.RejectScoreFloor = 0.3
	}

	instrChars := 0
	for _, s := range req.Instructions {
		instrChars += len(s)
	}
	if budget.ReserveForInstructions > 0 && instrChars > budget.ReserveForInstructions {
		instrChars = budget.ReserveForInstructions
	}
	charBudget := budget.MaxChars
	if charBudget > 0 {
		charBudget -= instrChars
		if charBudget < 0 {
			charBudget = 0
		}
	}

	drafts := dedupDrafts(req.Items)
	sort.SliceStable(drafts, func(i, j int) bool {
		if drafts[i].Required != drafts[j].Required {
			return drafts[i].Required
		}
		return drafts[i].Candidate.MergedScore > drafts[j].Candidate.MergedScore
	})

	var required, optional []DraftItem
	var rejected []retrieval.EvidenceItem
	for _, d := range drafts {
		if d.Class == foundation.EvidenceInstruction || d.Class == foundation.EvidencePolicy {
			rejected = append(rejected, toRejected(d, "instruction_or_policy_not_evidence"))
			continue
		}
		if d.Candidate.TrustLevel == foundation.TrustQuarantined {
			rejected = append(rejected, toRejected(d, "quarantined"))
			continue
		}
		if !trustMeets(d.Candidate.TrustLevel, req.Focus.RequiredTrustLevel) {
			rejected = append(rejected, toRejected(d, "trust_below_required"))
			continue
		}
		if d.Required {
			required = append(required, d)
		} else {
			optional = append(optional, d)
		}
	}

	var selected []retrieval.EvidenceItem
	usedChars := 0
	usedItems := 0

	add := func(d DraftItem, force bool) error {
		itemChars := len(d.Surface)
		if force {
			if charBudget > 0 && usedChars+itemChars > charBudget {
				return apperr.New(apperr.Conflict, "budget_exhausted_required")
			}
			if budget.MaxItems > 0 && usedItems >= budget.MaxItems {
				return apperr.New(apperr.Conflict, "budget_exhausted_required")
			}
		} else {
			if budget.MaxItems > 0 && usedItems >= budget.MaxItems {
				return errSkip
			}
			if charBudget > 0 && usedChars+itemChars > charBudget {
				return errSkip
			}
			if budget.MaxTokensEstimate > 0 && estimateTokens(usedChars+itemChars) > budget.MaxTokensEstimate {
				return errSkip
			}
		}
		selected = append(selected, toEvidence(d))
		usedChars += itemChars
		usedItems++
		return nil
	}

	for _, d := range required {
		if err := add(d, true); err != nil {
			return retrieval.ContextPack{}, err
		}
	}
	for _, d := range optional {
		if err := add(d, false); err != nil {
			if d.Candidate.MergedScore >= budget.RejectScoreFloor {
				rejected = append(rejected, toRejected(d, "budget_trim"))
			}
			continue
		}
	}

	pack := retrieval.ContextPack{
		ID:                       req.PackID,
		ProjectID:                req.ProjectID,
		TaskID:                   req.TaskID,
		RetrievalPlanID:          req.PlanID,
		Purpose:                  req.Purpose,
		Budget:                   budget,
		Instructions:             append([]string(nil), req.Instructions...),
		PolicyRefs:               append([]ids.PolicyID(nil), req.PolicyRefs...),
		EvidenceItems:            selected,
		RejectedItems:            rejected,
		VerificationRequirements: append([]string(nil), req.VerificationRequirements...),
		BudgetEstimatorVersion:   budget.BudgetEstimatorVersion,
	}
	pack.Checksum = Checksum(pack)
	if err := pack.Validate(); err != nil {
		return retrieval.ContextPack{}, apperr.Wrap(apperr.Validation, "context_pack", err)
	}

	if req.Recorder != nil {
		ev := tracing.Event{
			ID:        ids.TraceEventID(string(req.PackID) + ":built"),
			ProjectID: req.ProjectID,
			RunID:     req.RunID,
			Type:      tracing.EventContextPackBuilt,
			Timestamp: time.Now().UTC(),
			Payload: map[string]string{
				"pack_id":   string(req.PackID),
				"evidence":  fmt.Sprintf("%d", len(selected)),
				"rejected":  fmt.Sprintf("%d", len(rejected)),
				"checksum":  string(pack.Checksum),
				"estimator": pack.BudgetEstimatorVersion,
			},
		}
		if err := req.Recorder.Append(ctx, ev); err != nil {
			return retrieval.ContextPack{}, err
		}
	}
	return pack, nil
}

var errSkip = fmt.Errorf("skip")

func estimateTokens(chars int) int { return chars / 4 }

func trustRank(t foundation.TrustLevel) int {
	switch t {
	case foundation.TrustTrusted:
		return 4
	case foundation.TrustProject:
		return 3
	case foundation.TrustExternal:
		return 2
	case foundation.TrustUntrusted:
		return 1
	case foundation.TrustQuarantined:
		return 0
	default:
		return -1
	}
}

func trustMeets(have, required foundation.TrustLevel) bool {
	return trustRank(have) >= trustRank(required)
}

func dedupDrafts(items []DraftItem) []DraftItem {
	byKey := map[string]DraftItem{}
	order := make([]string, 0)
	for _, item := range items {
		key := item.Candidate.DedupKey()
		if existing, ok := byKey[key]; ok {
			if item.Required && !existing.Required {
				byKey[key] = item
				continue
			}
			if item.Candidate.MergedScore > existing.Candidate.MergedScore && existing.Required == item.Required {
				byKey[key] = item
			}
			continue
		}
		byKey[key] = item
		order = append(order, key)
	}
	out := make([]DraftItem, 0, len(order))
	for _, k := range order {
		out = append(out, byKey[k])
	}
	return out
}

func toEvidence(d DraftItem) retrieval.EvidenceItem {
	id := d.ID
	if id == "" {
		id = d.Candidate.DedupKey()
	}
	return retrieval.EvidenceItem{
		ID:         id,
		Class:      d.Class,
		TrustLevel: d.Candidate.TrustLevel,
		SourceRef:  d.Candidate.SourceRef,
		Surface:    d.Surface,
		Summary:    d.Summary,
		Candidate:  d.Candidate,
	}
}

func toRejected(d DraftItem, reason string) retrieval.EvidenceItem {
	item := toEvidence(d)
	item.RejectionReason = reason
	return item
}

// Checksum computes ADR-0020 pack checksum over canonical content without Checksum field.
func Checksum(p retrieval.ContextPack) foundation.ChecksumHex {
	var b strings.Builder
	b.WriteString(foundation.PackChecksumDomain)
	b.WriteByte(0)
	b.WriteString(string(p.ID))
	b.WriteByte(0)
	b.WriteString(string(p.ProjectID))
	b.WriteByte(0)
	b.WriteString(string(p.TaskID))
	b.WriteByte(0)
	b.WriteString(string(p.RetrievalPlanID))
	b.WriteByte(0)
	b.WriteString(p.Purpose)
	b.WriteByte(0)
	b.WriteString(p.BudgetEstimatorVersion)
	b.WriteByte(0)
	for _, s := range p.Instructions {
		b.WriteString(s)
		b.WriteByte(0)
	}
	for _, id := range p.PolicyRefs {
		b.WriteString(string(id))
		b.WriteByte(0)
	}
	for _, item := range p.EvidenceItems {
		writeEvidence(&b, item)
	}
	for _, item := range p.RejectedItems {
		writeEvidence(&b, item)
		b.WriteString(item.RejectionReason)
		b.WriteByte(0)
	}
	for _, s := range p.VerificationRequirements {
		b.WriteString(s)
		b.WriteByte(0)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return foundation.ChecksumHex(hex.EncodeToString(sum[:]))
}

func writeEvidence(b *strings.Builder, item retrieval.EvidenceItem) {
	b.WriteString(item.ID)
	b.WriteByte(0)
	b.WriteString(string(item.Class))
	b.WriteByte(0)
	b.WriteString(string(item.TrustLevel))
	b.WriteByte(0)
	b.WriteString(string(item.SourceRef.SourceID))
	b.WriteByte(0)
	b.WriteString(fmt.Sprintf("%d:%d", item.SourceRef.Span.Start, item.SourceRef.Span.End))
	b.WriteByte(0)
	b.WriteString(string(item.SourceRef.Checksum))
	b.WriteByte(0)
	b.WriteString(item.Surface)
	b.WriteByte(0)
}
