package devcli

import (
	"strings"
	"unicode/utf8"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/policy/isolation"
	"github.com/fastygo/context/internal/redaction"
	"github.com/fastygo/context/internal/retrieval"
)

const inspectorSurfaceMax = 240

// InspectReport is Lab-facing explanation of search/pack decisions (Chunk 26).
// It omits host filesystem paths and raw DB rows.
type InspectReport struct {
	OK           bool              `json:"ok"`
	ProjectID    ids.ProjectID     `json:"project_id"`
	SnapshotID   ids.SnapshotID    `json:"snapshot_id,omitempty"`
	Query        string            `json:"query,omitempty"`
	Mode         string            `json:"mode"` // pack | pack_id
	FocusID      ids.FocusID       `json:"focus_id,omitempty"`
	PackID       ids.PackID        `json:"pack_id,omitempty"`
	Purpose      string            `json:"purpose,omitempty"`
	PlanID       ids.PlanID        `json:"retrieval_plan_id,omitempty"`
	Budget       InspectBudget     `json:"budget"`
	Instructions []string          `json:"instructions,omitempty"`
	Selected     []InspectEvidence `json:"selected"`
	Rejected     []InspectEvidence `json:"rejected,omitempty"`
	Candidates   []InspectCandidate `json:"candidates,omitempty"`
	Checksum     foundation.ChecksumHex `json:"pack_checksum,omitempty"`
	Notes        []string          `json:"notes,omitempty"`
	Redacted     bool              `json:"redacted,omitempty"`
}

// InspectBudget summarizes ContextPack budget for Lab.
type InspectBudget struct {
	MaxItems               int     `json:"max_items,omitempty"`
	MaxChars               int     `json:"max_chars,omitempty"`
	MaxTokensEstimate      int     `json:"max_tokens_estimate,omitempty"`
	BudgetEstimatorVersion string  `json:"budget_estimator_version,omitempty"`
	RejectScoreFloor       float64 `json:"reject_score_floor,omitempty"`
	SelectedCount          int     `json:"selected_count"`
	RejectedCount          int     `json:"rejected_count"`
	SelectedChars          int     `json:"selected_chars"`
}

// InspectEvidence is one selected/rejected pack item without host paths.
type InspectEvidence struct {
	ID              string                   `json:"id"`
	Class           foundation.EvidenceClass `json:"class"`
	TrustLevel      foundation.TrustLevel    `json:"trust_level"`
	ChunkID         ids.ChunkID              `json:"chunk_id,omitempty"`
	SourceID        ids.SourceID             `json:"source_id,omitempty"`
	PathKey         string                   `json:"path_key,omitempty"`
	SpanStart       uint64                   `json:"span_start,omitempty"`
	SpanEnd         uint64                   `json:"span_end,omitempty"`
	TextChecksum    foundation.ChecksumHex   `json:"text_checksum,omitempty"`
	MergedScore     float64                  `json:"merged_score,omitempty"`
	Reasons         []string                 `json:"reasons,omitempty"`
	SurfacePreview  string                   `json:"surface_preview,omitempty"`
	RejectionReason string                   `json:"rejection_reason,omitempty"`
}

// InspectCandidate is a retrieval hit summary for Lab.
type InspectCandidate struct {
	ChunkID      ids.ChunkID            `json:"chunk_id"`
	SourceID     ids.SourceID           `json:"source_id,omitempty"`
	PathKey      string                 `json:"path_key,omitempty"`
	MergedScore  float64                `json:"merged_score"`
	TrustLevel   foundation.TrustLevel  `json:"trust_level,omitempty"`
	TextChecksum foundation.ChecksumHex `json:"text_checksum,omitempty"`
	Reasons      []string               `json:"reasons,omitempty"`
}

// Inspect builds an inspector report from a query (search+pack) or an existing pack_id.
func Inspect(dataDir, projectID, query, focusID, packID string) (InspectReport, error) {
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return InspectReport{}, err
	}
	if err := isolation.RequireProjectMatch(st.Project.ID, ids.ProjectID(projectID)); err != nil {
		return InspectReport{}, err
	}

	if packID != "" {
		pack, ok := findPack(st.Packs, ids.PackID(packID))
		if !ok {
			return InspectReport{}, apperr.New(apperr.NotFound, "pack not found")
		}
		rep := inspectPack(st, pack, "", "pack_id", focusID)
		rep.Notes = append(rep.Notes, "inspected existing pack; candidates omitted")
		return rep, nil
	}
	if query == "" {
		return InspectReport{}, apperr.New(apperr.Validation, "query or pack_id required")
	}

	search, err := Search(dataDir, projectID, query, "hybrid", focusID)
	if err != nil {
		return InspectReport{}, err
	}
	packRes, err := BuildPack(dataDir, projectID, query, focusID)
	if err != nil {
		return InspectReport{}, err
	}
	rep := inspectPack(st, packRes.Pack, query, "pack", string(packRes.FocusID))
	rep.SnapshotID = search.SnapshotID
	rep.Candidates = inspectCandidates(st, search.Candidates)
	rep.Notes = append(rep.Notes,
		"built via hybrid search + context-pack",
		"surface_preview is truncated; full surface stays on pack evidence",
	)
	return rep, nil
}

func findPack(packs []retrieval.ContextPack, id ids.PackID) (retrieval.ContextPack, bool) {
	for i := len(packs) - 1; i >= 0; i-- {
		if packs[i].ID == id {
			return packs[i], true
		}
	}
	return retrieval.ContextPack{}, false
}

func inspectPack(st State, pack retrieval.ContextPack, query, mode, focusID string) InspectReport {
	selectedChars := 0
	selected := make([]InspectEvidence, 0, len(pack.EvidenceItems))
	for _, e := range pack.EvidenceItems {
		selectedChars += utf8.RuneCountInString(e.Surface)
		selected = append(selected, inspectEvidence(st, e))
	}
	rejected := make([]InspectEvidence, 0, len(pack.RejectedItems))
	for _, e := range pack.RejectedItems {
		rejected = append(rejected, inspectEvidence(st, e))
	}
	focus := ids.FocusID(focusID)
	if focus == "" {
		focus = packResFocusFromState(st, pack)
	}
	snap := st.Project.ActiveSnapshotID
	if snap == "" {
		snap = st.Snapshot.ID
	}
	return InspectReport{
		OK:           true,
		ProjectID:    pack.ProjectID,
		SnapshotID:   snap,
		Query:        query,
		Mode:         mode,
		FocusID:      focus,
		PackID:       pack.ID,
		Purpose:      pack.Purpose,
		PlanID:       pack.RetrievalPlanID,
		Budget: InspectBudget{
			MaxItems:               pack.Budget.MaxItems,
			MaxChars:               pack.Budget.MaxChars,
			MaxTokensEstimate:      pack.Budget.MaxTokensEstimate,
			BudgetEstimatorVersion: firstNonEmpty(pack.BudgetEstimatorVersion, pack.Budget.BudgetEstimatorVersion),
			RejectScoreFloor:       pack.Budget.RejectScoreFloor,
			SelectedCount:          len(selected),
			RejectedCount:          len(rejected),
			SelectedChars:          selectedChars,
		},
		Instructions: pack.Instructions,
		Selected:     selected,
		Rejected:     rejected,
		Checksum:     pack.Checksum,
		Redacted:     inspectHasRedaction(selected, rejected),
	}
}

func inspectHasRedaction(selected, rejected []InspectEvidence) bool {
	for _, e := range selected {
		if strings.Contains(e.SurfacePreview, redaction.Replacement) {
			return true
		}
	}
	for _, e := range rejected {
		if strings.Contains(e.SurfacePreview, redaction.Replacement) {
			return true
		}
	}
	return false
}

func packResFocusFromState(st State, pack retrieval.ContextPack) ids.FocusID {
	for _, f := range st.Focuses {
		if f.ProjectID == pack.ProjectID && f.TaskID == pack.TaskID {
			return f.ID
		}
	}
	return ""
}

func inspectEvidence(st State, e retrieval.EvidenceItem) InspectEvidence {
	chunkID := e.Candidate.ChunkID
	if chunkID == "" {
		chunkID = e.SourceRef.ChunkID
	}
	preview, _ := redaction.Apply(truncateRunes(e.Surface, inspectorSurfaceMax))
	out := InspectEvidence{
		ID:              e.ID,
		Class:           e.Class,
		TrustLevel:      e.TrustLevel,
		ChunkID:         chunkID,
		SourceID:        e.SourceRef.SourceID,
		PathKey:         pathKeyForChunk(st, chunkID),
		SpanStart:       e.SourceRef.Span.Start,
		SpanEnd:         e.SourceRef.Span.End,
		TextChecksum:    firstChecksum(e.Candidate.TextChecksum, e.SourceRef.Checksum),
		MergedScore:     e.Candidate.MergedScore,
		Reasons:         contributionReasons(e.Candidate.Contributions),
		SurfacePreview:  preview,
		RejectionReason: e.RejectionReason,
	}
	return out
}

func inspectCandidates(st State, cands []retrieval.Candidate) []InspectCandidate {
	out := make([]InspectCandidate, 0, len(cands))
	for _, c := range cands {
		out = append(out, InspectCandidate{
			ChunkID:      c.ChunkID,
			SourceID:     c.SourceRef.SourceID,
			PathKey:      pathKeyForChunk(st, c.ChunkID),
			MergedScore:  c.MergedScore,
			TrustLevel:   c.TrustLevel,
			TextChecksum: firstChecksum(c.TextChecksum, c.SourceRef.Checksum),
			Reasons:      contributionReasons(c.Contributions),
		})
	}
	return out
}

func pathKeyForChunk(st State, chunkID ids.ChunkID) string {
	if chunkID == "" {
		return ""
	}
	for _, ch := range st.Chunks {
		if ch.ChunkID == chunkID {
			return ch.PathKey
		}
	}
	return ""
}

func firstChecksum(vals ...foundation.ChecksumHex) foundation.ChecksumHex {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func contributionReasons(cs []retrieval.ScoreContribution) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		if c.Explanation != "" {
			out = append(out, c.Explanation)
			continue
		}
		for _, r := range c.Reasons {
			if r != "" {
				out = append(out, string(r))
			}
		}
	}
	return out
}

func truncateRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return s
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max]) + "…"
}
