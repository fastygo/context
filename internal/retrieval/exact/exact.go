// Package exact implements deterministic source/span phrase lookup.
package exact

import (
	"context"
	"strings"

	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/index"
	"github.com/fastygo/context/internal/retrieval/merge"
)

const RetrieverID = "exact"

// Retriever performs exact phrase lookup over an in-memory chunk index.
type Retriever struct {
	Index *index.Memory
}

func (r Retriever) ID() string { return RetrieverID }

func (r Retriever) Retrieve(ctx context.Context, plan retrieval.RetrievalPlan, query string) ([]retrieval.Candidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := plan.Validate(); err != nil {
		return nil, err
	}
	query = strings.TrimSpace(query)
	if query == "" || r.Index == nil {
		return nil, nil
	}
	var out []retrieval.Candidate
	for _, rec := range r.Index.List(plan.ProjectID, plan.SnapshotID) {
		if !index.MatchesFilters(rec, plan.Filters) {
			continue
		}
		if !index.ContainsPhrase(rec.Text, query) {
			continue
		}
		out = append(out, retrieval.Candidate{
			ChunkID: rec.ChunkID,
			SourceRef: corpus.SourceRef{
				ProjectID: rec.ProjectID,
				SourceID:  rec.SourceID,
				ChunkID:   rec.ChunkID,
				Span:      rec.Span,
				Checksum:  rec.TextChecksum,
			},
			TrustLevel:   rec.TrustLevel,
			TextChecksum: rec.TextChecksum,
			Contributions: []retrieval.ScoreContribution{{
				RetrieverID:     RetrieverID,
				RawScore:        1,
				NormalizedScore: 1,
				Weight:          merge.DefaultWeight(RetrieverID),
				Reasons:         []foundation.ScoreReason{foundation.ReasonExactPhrase},
				Explanation:     "exact phrase match in chunk text",
				SnapshotID:      plan.SnapshotID,
				ProjectID:       plan.ProjectID,
				SenseID:         firstSense(rec),
				ConceptID:       firstConcept(rec),
				AttestationID:   firstAttestation(rec),
			}},
		})
	}
	merge.ApplyExactNormalization(out)
	return merge.DedupAndMerge(out), nil
}

func firstSense(rec index.ChunkRecord) ids.SenseID {
	if len(rec.SenseIDs) == 0 {
		return ""
	}
	return rec.SenseIDs[0]
}

func firstConcept(rec index.ChunkRecord) ids.ConceptID {
	if len(rec.ConceptIDs) == 0 {
		return ""
	}
	return rec.ConceptIDs[0]
}

func firstAttestation(rec index.ChunkRecord) ids.AttestationID {
	if len(rec.AttestationIDs) == 0 {
		return ""
	}
	return rec.AttestationIDs[0]
}

var _ retrieval.Retriever = Retriever{}
