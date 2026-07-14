// Package jsonres loads a curated JSON lexicon into a ResourceAdapter (S3 / A3).
package jsonres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/lexicon"
	"github.com/fastygo/context/internal/linguistic"
)

const (
	AdapterID      = "lexicon-json"
	AdapterVersion = "json-v1"
)

// Bundle is the curated JSON document shape.
type Bundle struct {
	Senses       []lexicon.Sense         `json:"senses"`
	Concepts     []lexicon.Concept       `json:"concepts"`
	Attestations []lexicon.Attestation   `json:"attestations"`
	Sources      []lexicon.LexiconSource `json:"sources"`
}

// Adapter is an in-memory ResourceAdapter seeded from curated JSON.
type Adapter struct {
	senses       map[ids.SenseID]lexicon.Sense
	concepts     map[ids.ConceptID]lexicon.Concept
	attestations map[ids.AttestationID]lexicon.Attestation
	sources      map[ids.LexiconSourceID]lexicon.LexiconSource
}

// Load parses curated JSON bytes into an Adapter.
func Load(raw []byte) (*Adapter, error) {
	var b Bundle
	if err := json.Unmarshal(raw, &b); err != nil {
		return nil, apperr.Wrap(apperr.Validation, "lexicon json", err)
	}
	a := &Adapter{
		senses:       make(map[ids.SenseID]lexicon.Sense, len(b.Senses)),
		concepts:     make(map[ids.ConceptID]lexicon.Concept, len(b.Concepts)),
		attestations: make(map[ids.AttestationID]lexicon.Attestation, len(b.Attestations)),
		sources:      make(map[ids.LexiconSourceID]lexicon.LexiconSource, len(b.Sources)),
	}
	for _, s := range b.Senses {
		if err := s.Validate(); err != nil {
			return nil, fmt.Errorf("sense %s: %w", s.ID, err)
		}
		a.senses[s.ID] = s
	}
	for _, c := range b.Concepts {
		if err := c.Validate(); err != nil {
			return nil, fmt.Errorf("concept %s: %w", c.ID, err)
		}
		a.concepts[c.ID] = c
	}
	for _, att := range b.Attestations {
		if err := att.Validate(); err != nil {
			return nil, fmt.Errorf("attestation %s: %w", att.ID, err)
		}
		a.attestations[att.ID] = att
	}
	for _, src := range b.Sources {
		if err := src.Validate(); err != nil {
			return nil, fmt.Errorf("lexicon source %s: %w", src.ID, err)
		}
		a.sources[src.ID] = src
	}
	return a, nil
}

func (a *Adapter) LookupSense(ctx context.Context, projectID ids.ProjectID, senseID ids.SenseID) (lexicon.Sense, error) {
	if err := ctx.Err(); err != nil {
		return lexicon.Sense{}, err
	}
	s, ok := a.senses[senseID]
	if !ok || s.ProjectID != projectID {
		return lexicon.Sense{}, apperr.New(apperr.NotFound, "sense not found")
	}
	return s, nil
}

func (a *Adapter) LookupConcept(ctx context.Context, projectID ids.ProjectID, conceptID ids.ConceptID) (lexicon.Concept, error) {
	if err := ctx.Err(); err != nil {
		return lexicon.Concept{}, err
	}
	c, ok := a.concepts[conceptID]
	if !ok || c.ProjectID != projectID {
		return lexicon.Concept{}, apperr.New(apperr.NotFound, "concept not found")
	}
	return c, nil
}

func (a *Adapter) LookupAttestation(ctx context.Context, projectID ids.ProjectID, attestationID ids.AttestationID) (lexicon.Attestation, error) {
	if err := ctx.Err(); err != nil {
		return lexicon.Attestation{}, err
	}
	att, ok := a.attestations[attestationID]
	if !ok || att.ProjectID != projectID {
		return lexicon.Attestation{}, apperr.New(apperr.NotFound, "attestation not found")
	}
	return att, nil
}

func (a *Adapter) LicenseMetadata(ctx context.Context, sourceID ids.LexiconSourceID) (lexicon.LexiconSource, error) {
	if err := ctx.Err(); err != nil {
		return lexicon.LexiconSource{}, err
	}
	s, ok := a.sources[sourceID]
	if !ok {
		return lexicon.LexiconSource{}, apperr.New(apperr.NotFound, "lexicon source not found")
	}
	return s, nil
}

// HarnessSeedJSON returns curated JSON aligned with lexicon/harness.DefaultSeed IDs.
func HarnessSeedJSON() []byte {
	b := Bundle{
		Senses: []lexicon.Sense{{
			ID: "sense-run-sport", ProjectID: "p_harness", LexemeID: "lex-run",
			Language: "en", Definition: "to jog / compete on foot", ConceptID: "concept-running",
			Register: "sport", Region: "us", TimePeriod: "2020s",
			LexiconSourceID: "lex-proof-1", SourceAuthority: "proof-fixture",
		}},
		Concepts: []lexicon.Concept{{
			ID: "concept-running", ProjectID: "p_harness", PreferredLabel: "running",
			Labels: []string{"running", "jogging"}, LexiconSourceID: "lex-proof-1",
		}},
		Attestations: []lexicon.Attestation{{
			ID: "att-runners-1", ProjectID: "p_harness", SourceID: "s_lex", ChunkID: "c_lex",
			Span: foundation.ByteSpan{Start: 0, End: 7}, Quote: "runners", Language: "en",
			LexemeID: "lex-run", SenseID: "sense-run-sport", ConceptID: "concept-running",
			Region: "us", Register: "sport", SourceAuthority: "proof-fixture",
			ImportVersion: "fixture-v1",
		}},
		Sources: []lexicon.LexiconSource{{
			ID: "lex-proof-1", Kind: "curated_json", Title: "Harness Lexicon",
			Authority: "proof-fixture", License: "CC0", Version: AdapterVersion,
			LanguageScope: []linguistic.LanguageCode{"en"},
		}},
	}
	raw, err := json.Marshal(b)
	if err != nil {
		panic(err)
	}
	return raw
}

var _ lexicon.ResourceAdapter = (*Adapter)(nil)
