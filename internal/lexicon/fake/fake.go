// Package fake provides fixture lexicon resource doubles for unit tests.
package fake

import (
	"context"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/lexicon"
)

// Resource is an in-memory Lexicon ResourceAdapter.
type Resource struct {
	Senses       map[ids.SenseID]lexicon.Sense
	Concepts     map[ids.ConceptID]lexicon.Concept
	Attestations map[ids.AttestationID]lexicon.Attestation
	Sources      map[ids.LexiconSourceID]lexicon.LexiconSource
}

func NewResource() *Resource {
	return &Resource{
		Senses:       make(map[ids.SenseID]lexicon.Sense),
		Concepts:     make(map[ids.ConceptID]lexicon.Concept),
		Attestations: make(map[ids.AttestationID]lexicon.Attestation),
		Sources:      make(map[ids.LexiconSourceID]lexicon.LexiconSource),
	}
}

func (r *Resource) LookupSense(ctx context.Context, projectID ids.ProjectID, senseID ids.SenseID) (lexicon.Sense, error) {
	if err := ctx.Err(); err != nil {
		return lexicon.Sense{}, err
	}
	s, ok := r.Senses[senseID]
	if !ok || s.ProjectID != projectID {
		return lexicon.Sense{}, apperr.New(apperr.NotFound, "sense not found")
	}
	return s, nil
}

func (r *Resource) LookupConcept(ctx context.Context, projectID ids.ProjectID, conceptID ids.ConceptID) (lexicon.Concept, error) {
	if err := ctx.Err(); err != nil {
		return lexicon.Concept{}, err
	}
	c, ok := r.Concepts[conceptID]
	if !ok || c.ProjectID != projectID {
		return lexicon.Concept{}, apperr.New(apperr.NotFound, "concept not found")
	}
	return c, nil
}

func (r *Resource) LookupAttestation(ctx context.Context, projectID ids.ProjectID, attestationID ids.AttestationID) (lexicon.Attestation, error) {
	if err := ctx.Err(); err != nil {
		return lexicon.Attestation{}, err
	}
	a, ok := r.Attestations[attestationID]
	if !ok || a.ProjectID != projectID {
		return lexicon.Attestation{}, apperr.New(apperr.NotFound, "attestation not found")
	}
	return a, nil
}

func (r *Resource) LicenseMetadata(ctx context.Context, sourceID ids.LexiconSourceID) (lexicon.LexiconSource, error) {
	if err := ctx.Err(); err != nil {
		return lexicon.LexiconSource{}, err
	}
	s, ok := r.Sources[sourceID]
	if !ok {
		return lexicon.LexiconSource{}, apperr.New(apperr.NotFound, "lexicon source not found")
	}
	return s, nil
}

// AttestationsBySource lists attestations for a source id.
type AttestationsBySource struct {
	BySource map[ids.SourceID][]lexicon.Attestation
}

func (a AttestationsBySource) ListBySource(ctx context.Context, projectID ids.ProjectID, sourceID ids.SourceID) ([]lexicon.Attestation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var out []lexicon.Attestation
	for _, att := range a.BySource[sourceID] {
		if att.ProjectID == projectID {
			out = append(out, att)
		}
	}
	return out, nil
}

var (
	_ lexicon.ResourceAdapter   = (*Resource)(nil)
	_ lexicon.AttestationSource = AttestationsBySource{}
)
