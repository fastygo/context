// Package lexicon defines lexicographic context contracts and resource ports (ADR-0016).
package lexicon

import (
	"context"
	"fmt"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/linguistic"
)

// Sense is one meaning of a lexeme; never collapsed into a lemma.
type Sense struct {
	ID              ids.SenseID
	ProjectID       ids.ProjectID
	LexemeID        ids.LexemeID
	Language        linguistic.LanguageCode
	Definition      string
	ConceptID       ids.ConceptID
	Register        Register
	Region          DialectRegion
	TimePeriod      TimePeriod
	LexiconSourceID ids.LexiconSourceID
	SourceAuthority string
	Confidence      float64
	LicenseRef      string
	Metadata        map[string]string
}

func (s Sense) Validate() error {
	if err := s.ID.Validate(); err != nil {
		return err
	}
	if err := s.ProjectID.Validate(); err != nil {
		return err
	}
	if err := s.LexemeID.Validate(); err != nil {
		return err
	}
	return s.Language.Validate()
}

// Concept is a language-independent or domain concept with labels and relations.
type Concept struct {
	ID              ids.ConceptID
	ProjectID       ids.ProjectID
	PreferredLabel  string
	Labels          []string
	ConceptScheme   string
	Broader         []ids.ConceptID
	Narrower        []ids.ConceptID
	Related         []ids.ConceptID
	ExactMatches    []ids.ConceptID
	CloseMatches    []ids.ConceptID
	LexiconSourceID ids.LexiconSourceID
	LicenseRef      string
	Metadata        map[string]string
}

func (c Concept) Validate() error {
	if err := c.ID.Validate(); err != nil {
		return err
	}
	if err := c.ProjectID.Validate(); err != nil {
		return err
	}
	if c.PreferredLabel == "" {
		return fmt.Errorf("concept preferred_label: empty")
	}
	return nil
}

// Attestation is witnessed usage in a source with quote and provenance.
type Attestation struct {
	ID              ids.AttestationID
	ProjectID       ids.ProjectID
	SourceID        ids.SourceID
	ChunkID         ids.ChunkID
	Span            foundation.ByteSpan
	Quote           string
	Language        linguistic.LanguageCode
	LexemeID        ids.LexemeID
	SenseID         ids.SenseID
	ConceptID       ids.ConceptID
	VariantID       ids.VariantID
	AttestedAt      string // opaque date/era string for phase 1
	Region          DialectRegion
	Register        Register
	SourceAuthority string
	Confidence      float64
	ImportVersion   string
	Metadata        map[string]string
}

func (a Attestation) Validate() error {
	if err := a.ID.Validate(); err != nil {
		return err
	}
	if err := a.ProjectID.Validate(); err != nil {
		return err
	}
	if err := a.SourceID.Validate(); err != nil {
		return err
	}
	if a.Quote == "" {
		return fmt.Errorf("attestation quote: empty")
	}
	return a.Span.Validate()
}

// VariantType classifies non-canonical forms.
type VariantType string

const (
	VariantOrthographic     VariantType = "orthographic"
	VariantHistorical       VariantType = "historical"
	VariantRegional         VariantType = "regional"
	VariantSlang            VariantType = "slang"
	VariantSpelling         VariantType = "spelling"
	VariantScript           VariantType = "script"
	VariantTransliteration  VariantType = "transliteration"
)

// Variant is a non-canonical form meaningful for retrieval or history.
type Variant struct {
	ID           ids.VariantID
	ProjectID    ids.ProjectID
	CanonicalRef string
	Variant      string
	Type         VariantType
	Language     linguistic.LanguageCode
	Script       linguistic.ScriptCode
	Region       DialectRegion
	TimePeriod   TimePeriod
	SourceID     ids.SourceID
	Confidence   float64
}

func (v Variant) Validate() error {
	if err := v.ID.Validate(); err != nil {
		return err
	}
	if err := v.ProjectID.Validate(); err != nil {
		return err
	}
	if v.Variant == "" || v.CanonicalRef == "" {
		return fmt.Errorf("variant: form and canonical_ref required")
	}
	return nil
}

// MultiwordExpression spans multiple tokens or syntactic words.
type MultiwordExpression struct {
	ID              ids.MWEID
	ProjectID       ids.ProjectID
	Surface         string
	Normalized      string
	Language        linguistic.LanguageCode
	TokenIDs        []ids.TokenID
	Span            foundation.ByteSpan
	LexemeID        ids.LexemeID
	SenseID         ids.SenseID
	ExpressionType  string
	AnalyzerVersion string
	Confidence      float64
}

func (m MultiwordExpression) Validate() error {
	if err := m.ID.Validate(); err != nil {
		return err
	}
	if err := m.ProjectID.Validate(); err != nil {
		return err
	}
	if m.Surface == "" {
		return fmt.Errorf("mwe surface: empty")
	}
	return m.Span.Validate()
}

// Register, DialectRegion, and TimePeriod are metadata-rich references, not closed enums.
type (
	Register      string
	DialectRegion string
	TimePeriod    string
)

// LexiconSource describes a dictionary, corpus, glossary, or authority list.
type LexiconSource struct {
	ID            ids.LexiconSourceID
	Kind          string
	Title         string
	Authority     string
	License       string
	Version       string
	LanguageScope []linguistic.LanguageCode
	URI           string
	Metadata      map[string]string
}

func (s LexiconSource) Validate() error {
	if err := s.ID.Validate(); err != nil {
		return err
	}
	if s.Kind == "" || s.Title == "" {
		return fmt.Errorf("lexicon_source: kind and title required")
	}
	return nil
}

// ClaimOrigin distinguishes authority-backed claims from adapter hints/inference.
type ClaimOrigin string

const (
	ClaimFromLexicon  ClaimOrigin = "lexicon_source"
	ClaimAdapterHint  ClaimOrigin = "adapter_hint"
	ClaimInference    ClaimOrigin = "inference"
)

// ResourceAdapter maps external lexicon formats into neutral contracts.
// Implementations live outside the core (TEI/SKOS/dictionary adapters).
type ResourceAdapter interface {
	LookupSense(ctx context.Context, projectID ids.ProjectID, senseID ids.SenseID) (Sense, error)
	LookupConcept(ctx context.Context, projectID ids.ProjectID, conceptID ids.ConceptID) (Concept, error)
	LookupAttestation(ctx context.Context, projectID ids.ProjectID, attestationID ids.AttestationID) (Attestation, error)
	LicenseMetadata(ctx context.Context, sourceID ids.LexiconSourceID) (LexiconSource, error)
}

// AttestationSource provides witnessed usage records without owning parsers.
type AttestationSource interface {
	ListBySource(ctx context.Context, projectID ids.ProjectID, sourceID ids.SourceID) ([]Attestation, error)
}
