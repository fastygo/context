// Package foundation defines shared enums, spans, and checksum conventions
// from ADR-0018, ADR-0019, ADR-0020, and ADR-0021.
package foundation

import "fmt"

// ChecksumHex is a lowercase hex-encoded digest string (typically SHA-256).
type ChecksumHex string

func (c ChecksumHex) Validate() error {
	if c == "" {
		return fmt.Errorf("checksum: empty")
	}
	return nil
}

// ByteSpan is a half-open byte range [Start, End) into newline-normalized UTF-8 text.
type ByteSpan struct {
	Start uint64 `json:"start"`
	End   uint64 `json:"end"`
}

// Validate rejects empty or inverted spans (ADR-0018 phase-1 chunks).
func (s ByteSpan) Validate() error {
	if s.Start >= s.End {
		return fmt.Errorf("byte span: start %d must be < end %d", s.Start, s.End)
	}
	return nil
}

// Len returns the number of bytes in the span.
func (s ByteSpan) Len() uint64 { return s.End - s.Start }

// TrustLevel labels source trust for retrieval and packing (ADR-0020).
type TrustLevel string

const (
	TrustTrusted     TrustLevel = "trusted"
	TrustProject     TrustLevel = "project"
	TrustExternal    TrustLevel = "external"
	TrustUntrusted   TrustLevel = "untrusted"
	TrustQuarantined TrustLevel = "quarantined"
)

func (t TrustLevel) Validate() error {
	switch t {
	case TrustTrusted, TrustProject, TrustExternal, TrustUntrusted, TrustQuarantined:
		return nil
	case "":
		return fmt.Errorf("trust_level: empty")
	default:
		return fmt.Errorf("trust_level: unknown %q", t)
	}
}

// EvidenceClass distinguishes instruction/data and claim kinds (ADR-0020).
type EvidenceClass string

const (
	EvidenceSourceText       EvidenceClass = "source_text"
	EvidenceLexicalAnalysis  EvidenceClass = "lexical_analysis"
	EvidenceSenseClaim       EvidenceClass = "sense_claim"
	EvidenceConceptMapping   EvidenceClass = "concept_mapping"
	EvidenceAttestation      EvidenceClass = "attestation"
	EvidenceToolOutput       EvidenceClass = "tool_output"
	EvidenceModelInference   EvidenceClass = "model_inference"
	EvidenceInstruction      EvidenceClass = "instruction"
	EvidencePolicy           EvidenceClass = "policy"
)

func (c EvidenceClass) Validate() error {
	switch c {
	case EvidenceSourceText, EvidenceLexicalAnalysis, EvidenceSenseClaim,
		EvidenceConceptMapping, EvidenceAttestation, EvidenceToolOutput,
		EvidenceModelInference, EvidenceInstruction, EvidencePolicy:
		return nil
	case "":
		return fmt.Errorf("evidence_class: empty")
	default:
		return fmt.Errorf("evidence_class: unknown %q", c)
	}
}

// MayJustifyFactualClaims reports whether the class may back factual claims in phase 1.
func (c EvidenceClass) MayJustifyFactualClaims() bool {
	switch c {
	case EvidenceSourceText, EvidenceAttestation, EvidenceToolOutput:
		return true
	default:
		return false
	}
}

// SnapshotStatus is the IndexSnapshot lifecycle state (ADR-0021).
type SnapshotStatus string

const (
	SnapshotBuilding   SnapshotStatus = "building"
	SnapshotReady      SnapshotStatus = "ready"
	SnapshotFailed     SnapshotStatus = "failed"
	SnapshotSuperseded SnapshotStatus = "superseded"
)

func (s SnapshotStatus) Validate() error {
	switch s {
	case SnapshotBuilding, SnapshotReady, SnapshotFailed, SnapshotSuperseded:
		return nil
	case "":
		return fmt.Errorf("snapshot_status: empty")
	default:
		return fmt.Errorf("snapshot_status: unknown %q", s)
	}
}

// IsSearchableAsActive reports whether the status may be pointed to by active_snapshot_id.
func (s SnapshotStatus) IsSearchableAsActive() bool {
	return s == SnapshotReady
}

// ScoreReason is a phase-1 retrieval explanation code (ADR-0019).
type ScoreReason string

const (
	ReasonExactPhrase      ScoreReason = "exact_phrase"
	ReasonExactSpan        ScoreReason = "exact_span"
	ReasonSparseTerm       ScoreReason = "sparse_term"
	ReasonDenseSimilarity  ScoreReason = "dense_similarity"
	ReasonLemmaMatch       ScoreReason = "lemma_match"
	ReasonWordformExpand   ScoreReason = "wordform_expand"
	// ReasonTokenTerm marks a token-boundary term hit (querylang leaf).
	ReasonTokenTerm ScoreReason = "token_term"
	// ReasonMorphPhrase marks a lemma-sequence phrase hit (querylang leaf).
	ReasonMorphPhrase ScoreReason = "morph_phrase"
	ReasonSenseFilter      ScoreReason = "sense_filter"
	ReasonConceptFilter    ScoreReason = "concept_filter"
	ReasonAttestationFilter ScoreReason = "attestation_filter"
	ReasonTrustBoost       ScoreReason = "trust_boost"
	ReasonRecencyBoost     ScoreReason = "recency_boost"
	ReasonCitationBoost    ScoreReason = "citation_boost"
)

func (r ScoreReason) Validate() error {
	switch r {
	case ReasonExactPhrase, ReasonExactSpan, ReasonSparseTerm, ReasonDenseSimilarity,
		ReasonLemmaMatch, ReasonWordformExpand, ReasonTokenTerm, ReasonMorphPhrase,
		ReasonSenseFilter, ReasonConceptFilter,
		ReasonAttestationFilter, ReasonTrustBoost, ReasonRecencyBoost, ReasonCitationBoost:
		return nil
	case "":
		return fmt.Errorf("score_reason: empty")
	default:
		return fmt.Errorf("score_reason: unknown %q", r)
	}
}

// Hash domain labels pinned for deterministic Merkle leaves (ADR-0018).
const (
	SourceLeafDomain = "context/source-leaf/v1"
	ChunkHashDomain  = "context/chunk/v1"
	PackChecksumDomain = "context/context-pack/v1"
	SourceMerkleAlgo = "source_merkle_v1"
	ChunkSetMerkleAlgo = "chunk_set_merkle_v1"
)
