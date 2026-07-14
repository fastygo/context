// Package retrieval defines planners, candidates, context packs, and search ports.
package retrieval

import (
	"context"
	"fmt"

	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing"
	"github.com/fastygo/context/internal/lexicon"
)

// FocusProfile is the task-specific retrieval and packing lens.
type FocusProfile struct {
	ID                   ids.FocusID           `json:"id"`
	ProjectID            ids.ProjectID         `json:"project_id"`
	TaskID               ids.TaskID            `json:"task_id,omitempty"`
	Objective            string                `json:"objective"`
	Scope                string                `json:"scope,omitempty"`
	PreferredSourceTypes []string              `json:"preferred_source_types,omitempty"`
	ForbiddenSourceTypes []string              `json:"forbidden_source_types,omitempty"`
	RequiredTrustLevel   foundation.TrustLevel `json:"required_trust_level"`
	FreshnessWindow      string                `json:"freshness_window,omitempty"`
	ExactnessLevel       string                `json:"exactness_level,omitempty"`
	CitationStrictness   string                `json:"citation_strictness,omitempty"`
	ContextBudget        Budget                `json:"context_budget"`
	AllowedTools         []string              `json:"allowed_tools,omitempty"`
	AllowedSubagents     []string              `json:"allowed_subagents,omitempty"`
	NegativeAssumptions  []string              `json:"negative_assumptions,omitempty"`
}

func (f FocusProfile) Validate() error {
	if err := f.ID.Validate(); err != nil {
		return err
	}
	if err := f.ProjectID.Validate(); err != nil {
		return err
	}
	if f.Objective == "" {
		return fmt.Errorf("focus_profile objective: empty")
	}
	return f.RequiredTrustLevel.Validate()
}

// Budget constrains ContextPack construction (ADR-0020).
type Budget struct {
	MaxItems               int     `json:"max_items,omitempty"`
	MaxChars               int     `json:"max_chars,omitempty"`
	MaxTokensEstimate      int     `json:"max_tokens_estimate,omitempty"`
	ReserveForInstructions int     `json:"reserve_for_instructions,omitempty"`
	BudgetEstimatorVersion string  `json:"budget_estimator_version,omitempty"`
	RejectScoreFloor       float64 `json:"reject_score_floor,omitempty"`
	AllowSpanTruncate      bool    `json:"allow_span_truncate,omitempty"`
}

// RetrievalPlan chooses retriever paths, filters, and budgets for a task.
type RetrievalPlan struct {
	ID          ids.PlanID
	ProjectID   ids.ProjectID
	TaskID      ids.TaskID
	FocusID     ids.FocusID
	SnapshotID  ids.SnapshotID
	Strategies  []RetrieverStrategy
	Filters     RetrievalFilters
	TopNRawPool int
}

func (p RetrievalPlan) Validate() error {
	if err := p.ID.Validate(); err != nil {
		return err
	}
	if err := p.ProjectID.Validate(); err != nil {
		return err
	}
	if err := p.SnapshotID.Validate(); err != nil {
		return err
	}
	if len(p.Strategies) == 0 {
		return fmt.Errorf("retrieval_plan: strategies required")
	}
	return p.Filters.Validate()
}

// RetrieverStrategy names a retriever family and weight override.
type RetrieverStrategy struct {
	RetrieverID string
	Weight      float64
}

// RetrievalFilters are explainable constraints; they do not replace source text.
type RetrievalFilters struct {
	SenseID         ids.SenseID
	ConceptID       ids.ConceptID
	AttestationID   ids.AttestationID
	Register        lexicon.Register
	DialectRegion   lexicon.DialectRegion
	TimePeriod      lexicon.TimePeriod
	LexiconSourceID ids.LexiconSourceID
	SourceAuthority string
	Language        string
	TemporalRange *corpus.TemporalRange
	// GraphNodeID is reserved (ADR-0040). MatchesFilters ignores it until a
	// superseding ADR implements graph-backed enforcement.
	GraphNodeID ids.GraphNodeID
}

// Validate checks optional generic filters without conflating event time with
// lexicographic TimePeriod.
func (f RetrievalFilters) Validate() error {
	if f.TemporalRange != nil {
		return f.TemporalRange.Validate()
	}
	return nil
}

// MatchesTemporal applies deterministic half-open overlap semantics. Sources
// without temporal metadata do not match an explicit temporal filter.
func (f RetrievalFilters) MatchesTemporal(metadata *corpus.TemporalMetadata) bool {
	if f.TemporalRange == nil {
		return true
	}
	if metadata == nil || metadata.Validate() != nil {
		return false
	}
	return f.TemporalRange.Overlaps(metadata.Range)
}

// ScoreContribution records one retriever's contribution (ADR-0019).
type ScoreContribution struct {
	RetrieverID     string                   `json:"retriever_id"`
	RawScore        float64                  `json:"raw_score"`
	NormalizedScore float64                  `json:"normalized_score"`
	Weight          float64                  `json:"weight"`
	Reasons         []foundation.ScoreReason `json:"reasons"`
	Explanation     string                   `json:"explanation"`
	SnapshotID      ids.SnapshotID           `json:"snapshot_id"`
	ProjectID       ids.ProjectID            `json:"project_id"`
	ExpansionIDs    []ids.ExpansionID        `json:"expansion_ids,omitempty"`
	SenseID         ids.SenseID              `json:"sense_id,omitempty"`
	ConceptID       ids.ConceptID            `json:"concept_id,omitempty"`
	AttestationID   ids.AttestationID        `json:"attestation_id,omitempty"`
	AnalyzerVersion string                   `json:"analyzer_version,omitempty"`
	EmbedVersion    string                   `json:"embed_version,omitempty"`
}

// Candidate is a merged retrieval hit before packing.
type Candidate struct {
	ChunkID       ids.ChunkID            `json:"chunk_id"`
	SourceRef     corpus.SourceRef       `json:"source_ref"`
	MergedScore   float64                `json:"merged_score"`
	Contributions []ScoreContribution    `json:"contributions"`
	TrustLevel    foundation.TrustLevel  `json:"trust_level"`
	TextChecksum  foundation.ChecksumHex `json:"text_checksum"`
	// Snippet is an optional citation window with absolute offsets into chunk text (C4).
	Snippet *Snippet `json:"snippet,omitempty"`
}

// Snippet is a citation window into chunk text (see retrieval/snippet for builders).
type Snippet struct {
	Text          string                 `json:"text"`
	ChunkSpan     foundation.ByteSpan    `json:"chunk_span"`
	Highlights    []foundation.ByteSpan  `json:"highlights,omitempty"`
	ChunkChecksum foundation.ChecksumHex `json:"chunk_checksum"`
	Query         string                 `json:"query,omitempty"`
}

func (c Candidate) DedupKey() string {
	return fmt.Sprintf("%s:%d:%d:%s", c.SourceRef.SourceID, c.SourceRef.Span.Start, c.SourceRef.Span.End, c.TextChecksum)
}

// EvidenceItem is one selected or rejected pack entry.
type EvidenceItem struct {
	ID              string                   `json:"id"`
	Class           foundation.EvidenceClass `json:"class"`
	TrustLevel      foundation.TrustLevel    `json:"trust_level"`
	SourceRef       corpus.SourceRef         `json:"source_ref"`
	Surface         string                   `json:"surface"`
	Summary         string                   `json:"summary,omitempty"`
	Candidate       Candidate                `json:"candidate"`
	RejectionReason string                   `json:"rejection_reason,omitempty"`
}

func (e EvidenceItem) Validate() error {
	if err := e.Class.Validate(); err != nil {
		return err
	}
	if err := e.TrustLevel.Validate(); err != nil {
		return err
	}
	if e.Class == foundation.EvidenceInstruction || e.Class == foundation.EvidencePolicy {
		return nil
	}
	return e.SourceRef.Validate()
}

// ContextPack is the central runtime handoff object.
type ContextPack struct {
	ID                       ids.PackID             `json:"id"`
	ProjectID                ids.ProjectID          `json:"project_id"`
	TaskID                   ids.TaskID             `json:"task_id,omitempty"`
	RetrievalPlanID          ids.PlanID             `json:"retrieval_plan_id"`
	Purpose                  string                 `json:"purpose,omitempty"`
	Budget                   Budget                 `json:"budget"`
	Instructions             []string               `json:"instructions"`
	PolicyRefs               []ids.PolicyID         `json:"policy_refs,omitempty"`
	EvidenceItems            []EvidenceItem         `json:"evidence_items"`
	RejectedItems            []EvidenceItem         `json:"rejected_items,omitempty"`
	VerificationRequirements []string               `json:"verification_requirements,omitempty"`
	Checksum                 foundation.ChecksumHex `json:"checksum"`
	BudgetEstimatorVersion   string                 `json:"budget_estimator_version,omitempty"`
}

func (p ContextPack) Validate() error {
	if err := p.ID.Validate(); err != nil {
		return err
	}
	if err := p.ProjectID.Validate(); err != nil {
		return err
	}
	if err := p.RetrievalPlanID.Validate(); err != nil {
		return err
	}
	for _, item := range p.EvidenceItems {
		if item.Class == foundation.EvidenceInstruction {
			return fmt.Errorf("context_pack: instruction class must not appear in evidence_items")
		}
		if err := item.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Retriever is a single retrieval path.
type Retriever interface {
	ID() string
	Retrieve(ctx context.Context, plan RetrievalPlan, query string) ([]Candidate, error)
}

// VectorStore is the dense embedding port (ADR-0004, ADR-0017).
type VectorStore interface {
	Upsert(ctx context.Context, ns indexing.VectorNamespace, points []VectorPoint) error
	Search(ctx context.Context, ns indexing.VectorNamespace, vector []float32, limit int) ([]VectorHit, error)
}

// VectorPoint is one dense row keyed by chunk.
type VectorPoint struct {
	ChunkID          ids.ChunkID
	ProjectID        ids.ProjectID
	SnapshotID       ids.SnapshotID
	EmbeddingVersion string
	ChunkerVersion   string
	MorphVersion     string
	ContextRef       ids.ContextRefID
	Span             foundation.ByteSpan
	Language         string
	Vector           []float32
}

// VectorHit is a dense search result with provenance echoed from the store.
type VectorHit struct {
	ChunkID          ids.ChunkID
	Score            float64
	EmbeddingVersion string
	ChunkerVersion   string
	MorphVersion     string
	ContextRef       ids.ContextRefID
	SnapshotID       ids.SnapshotID
}

// BackendCapabilities declares what a dense/sparse backend can enforce
// server-side so planners can keep client-side filters where needed.
type BackendCapabilities struct {
	BackendID                string   `json:"backend_id"`
	Kind                     string   `json:"kind"`
	SupportsProjectFilter    bool     `json:"supports_project_filter"`
	SupportsSnapshotFilter   bool     `json:"supports_snapshot_filter"`
	SupportsTemporalFilter   bool     `json:"supports_temporal_filter"`
	SupportsMetadataFilter   bool     `json:"supports_metadata_filter"`
	SupportsPayloadNamespace bool     `json:"supports_payload_namespace"`
	MaxDimension             int      `json:"max_dimension,omitempty"`
	Dimension                int      `json:"dimension,omitempty"`
	Metrics                  []string `json:"metrics,omitempty"`
	NamespaceModel           string   `json:"namespace_model,omitempty"`
	ManagedService           bool     `json:"managed_service"`
}

// CapabilityReporter is implemented by live vector/sparse adapters.
type CapabilityReporter interface {
	Capabilities() BackendCapabilities
}

// SparseSearchClient is the sparse/FTS port.
type SparseSearchClient interface {
	Search(ctx context.Context, projectID ids.ProjectID, snapshotID ids.SnapshotID, query string, limit int) ([]SparseHit, error)
}

// SparseHit is a sparse search result.
type SparseHit struct {
	ChunkID ids.ChunkID
	Score   float64
}

// Reranker is an optional post-merge stage (C11). Hybrid merge remains the
// default; Identity / WeightedResort document the intentional no-op path.
type Reranker interface {
	Rerank(ctx context.Context, query string, candidates []Candidate) ([]Candidate, error)
}
