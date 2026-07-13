// Package hybrid runs phase-1 multi-retriever search with explainable expansions.
package hybrid

import (
	"context"
	"time"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/linguistic"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/exact"
	"github.com/fastygo/context/internal/retrieval/merge"
	"github.com/fastygo/context/internal/tracing"
)

// Engine coordinates exact/sparse/dense (and optional expansion) retrieval.
type Engine struct {
	Exact     exact.Retriever
	Sparse    retrieval.Retriever
	Dense     retrieval.Retriever
	Expander  linguistic.QueryExpander
	Recorder  tracing.Recorder
	RejectExp map[string]bool // expanded terms that must be ignored (false-positive control)
}

// Result captures candidates plus replayable trace events for one query.
type Result struct {
	Candidates []retrieval.Candidate
	Expansions []linguistic.QueryExpansion
	Rejected   []linguistic.QueryExpansion
	Events     []tracing.Event
}

// Search executes plan strategies against the query.
func (e Engine) Search(ctx context.Context, plan retrieval.RetrievalPlan, queryID ids.QueryID, query string) (Result, error) {
	var res Result
	now := time.Now().UTC()
	res.Events = append(res.Events, tracing.Event{
		ID:         ids.TraceEventID(string(queryID) + ":query"),
		ProjectID:  plan.ProjectID,
		RunID:      ids.RunID(queryID),
		Type:       tracing.EventRetrievalQuery,
		Timestamp:  now,
		SnapshotID: plan.SnapshotID,
		Payload:    map[string]string{"query": query, "snapshot_id": string(plan.SnapshotID)},
	})

	terms := []string{query}
	if e.Expander != nil {
		exps, err := e.Expander.Expand(ctx, queryID, query, "en")
		if err != nil {
			return Result{}, err
		}
		for _, exp := range exps {
			if e.RejectExp[exp.ExpandedTerm] {
				res.Rejected = append(res.Rejected, exp)
				continue
			}
			res.Expansions = append(res.Expansions, exp)
			terms = append(terms, exp.ExpandedTerm)
			res.Events = append(res.Events, tracing.Event{
				ID:                ids.TraceEventID(string(exp.ID)),
				ProjectID:         plan.ProjectID,
				RunID:             ids.RunID(queryID),
				Type:              tracing.EventQueryExpansion,
				Timestamp:         now,
				SnapshotID:        plan.SnapshotID,
				QueryExpansionVer: exp.AdapterVersion,
				Payload: map[string]string{
					"original": exp.OriginalTerm,
					"expanded": exp.ExpandedTerm,
					"type":     string(exp.Type),
					"reason":   exp.Reason,
				},
			})
		}
	}

	var all []retrieval.Candidate
	for _, strategy := range plan.Strategies {
		for _, term := range terms {
			cands, err := e.retrieveOne(ctx, strategy.RetrieverID, plan, term)
			if err != nil {
				return Result{}, err
			}
			for i := range cands {
				for j := range cands[i].Contributions {
					if strategy.Weight > 0 {
						cands[i].Contributions[j].Weight = strategy.Weight
					}
					if term != query {
						cands[i].Contributions[j].ExpansionIDs = append(cands[i].Contributions[j].ExpansionIDs, ids.ExpansionID("exp:"+term))
						cands[i].Contributions[j].Reasons = append(cands[i].Contributions[j].Reasons, foundation.ReasonWordformExpand)
					}
				}
			}
			all = append(all, cands...)
		}
	}

	// Annotate filter hits for sense/concept/attestation when filters set.
	for i := range all {
		for j := range all[i].Contributions {
			if plan.Filters.SenseID != "" {
				all[i].Contributions[j].Reasons = appendUniqueReason(all[i].Contributions[j].Reasons, foundation.ReasonSenseFilter)
				all[i].Contributions[j].SenseID = plan.Filters.SenseID
			}
			if plan.Filters.ConceptID != "" {
				all[i].Contributions[j].Reasons = appendUniqueReason(all[i].Contributions[j].Reasons, foundation.ReasonConceptFilter)
				all[i].Contributions[j].ConceptID = plan.Filters.ConceptID
			}
			if plan.Filters.AttestationID != "" {
				all[i].Contributions[j].Reasons = appendUniqueReason(all[i].Contributions[j].Reasons, foundation.ReasonAttestationFilter)
				all[i].Contributions[j].AttestationID = plan.Filters.AttestationID
			}
		}
	}

	res.Candidates = merge.DedupAndMerge(all)
	res.Events = append(res.Events, tracing.Event{
		ID:         ids.TraceEventID(string(queryID) + ":candidates"),
		ProjectID:  plan.ProjectID,
		RunID:      ids.RunID(queryID),
		Type:       tracing.EventRetrievalCandidates,
		Timestamp:  now,
		SnapshotID: plan.SnapshotID,
		Payload:    map[string]string{"count": itoa(len(res.Candidates))},
	})
	if e.Recorder != nil {
		for _, ev := range res.Events {
			if err := e.Recorder.Append(ctx, ev); err != nil {
				return Result{}, err
			}
		}
	}
	return res, nil
}

func (e Engine) retrieveOne(ctx context.Context, retrieverID string, plan retrieval.RetrievalPlan, query string) ([]retrieval.Candidate, error) {
	switch retrieverID {
	case exact.RetrieverID:
		return e.Exact.Retrieve(ctx, plan, query)
	case "sparse":
		if e.Sparse == nil {
			return nil, nil
		}
		return e.Sparse.Retrieve(ctx, plan, query)
	case "dense":
		if e.Dense == nil {
			return nil, nil
		}
		return e.Dense.Retrieve(ctx, plan, query)
	default:
		return nil, nil
	}
}

func appendUniqueReason(in []foundation.ScoreReason, r foundation.ScoreReason) []foundation.ScoreReason {
	for _, existing := range in {
		if existing == r {
			return in
		}
	}
	return append(in, r)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
