// Package merge implements ADR-0019 score normalization, dedup, and weighted merge.
package merge

import (
	"sort"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/retrieval"
)

// DefaultWeight returns phase-1 default weights by retriever family.
func DefaultWeight(retrieverID string) float64 {
	switch retrieverID {
	case "exact":
		return 1.00
	case "term":
		return 0.95
	case "morphphrase":
		return 0.90
	case "sparse":
		return 0.75
	case "lemma", "wordform":
		return 0.70
	case "dense":
		return 0.55
	default:
		return 0.50
	}
}

// PriorityRank is used for stable tie-breaking (lower is better).
func PriorityRank(retrieverID string) int {
	switch retrieverID {
	case "exact":
		return 0
	case "term", "morphphrase":
		return 1
	case "sparse":
		return 2
	case "lemma", "wordform":
		return 3
	case "dense":
		return 4
	default:
		return 9
	}
}

// NormalizeScores applies min-max normalization within one retriever result set.
func NormalizeScores(raw []float64) []float64 {
	out := make([]float64, len(raw))
	if len(raw) == 0 {
		return out
	}
	minV, maxV := raw[0], raw[0]
	for _, v := range raw[1:] {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}
	if maxV == minV {
		return out // all zeros per ADR-0019
	}
	den := maxV - minV
	for i, v := range raw {
		out[i] = (v - minV) / den
	}
	return out
}

// DedupAndMerge merges candidates by ADR-0019 dedup key and recomputes merged scores.
func DedupAndMerge(candidates []retrieval.Candidate) []retrieval.Candidate {
	byKey := map[string]*retrieval.Candidate{}
	order := make([]string, 0)
	for _, c := range candidates {
		key := c.DedupKey()
		if existing, ok := byKey[key]; ok {
			existing.Contributions = append(existing.Contributions, c.Contributions...)
			if existing.ChunkID == "" {
				existing.ChunkID = c.ChunkID
			}
			continue
		}
		cp := c
		byKey[key] = &cp
		order = append(order, key)
	}
	out := make([]retrieval.Candidate, 0, len(order))
	for _, key := range order {
		c := byKey[key]
		c.MergedScore = mergedScore(*c)
		out = append(out, *c)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].MergedScore != out[j].MergedScore {
			return out[i].MergedScore > out[j].MergedScore
		}
		pi := bestPriority(out[i])
		pj := bestPriority(out[j])
		if pi != pj {
			return pi < pj
		}
		if out[i].ChunkID != out[j].ChunkID {
			return out[i].ChunkID < out[j].ChunkID
		}
		if out[i].SourceRef.SourceID != out[j].SourceRef.SourceID {
			return out[i].SourceRef.SourceID < out[j].SourceRef.SourceID
		}
		return out[i].SourceRef.Span.Start < out[j].SourceRef.Span.Start
	})
	return out
}

func mergedScore(c retrieval.Candidate) float64 {
	var sum float64
	for _, contrib := range c.Contributions {
		w := contrib.Weight
		if w == 0 {
			w = DefaultWeight(contrib.RetrieverID)
		}
		sum += w * contrib.NormalizedScore
	}
	return sum
}

func bestPriority(c retrieval.Candidate) int {
	best := 99
	for _, contrib := range c.Contributions {
		if p := PriorityRank(contrib.RetrieverID); p < best {
			best = p
		}
	}
	return best
}

// ApplyExactNormalization sets exact hits to normalized 1.0 (ADR-0019 {0,1} path).
func ApplyExactNormalization(cands []retrieval.Candidate) {
	for i := range cands {
		for j := range cands[i].Contributions {
			if cands[i].Contributions[j].RetrieverID == "exact" {
				cands[i].Contributions[j].NormalizedScore = 1
				if cands[i].Contributions[j].RawScore == 0 {
					cands[i].Contributions[j].RawScore = 1
				}
			}
		}
	}
}

// ReasonExact helpers keep call sites readable.
func ExactReasons(phrase bool) []foundation.ScoreReason {
	if phrase {
		return []foundation.ScoreReason{foundation.ReasonExactPhrase}
	}
	return []foundation.ScoreReason{foundation.ReasonExactSpan}
}
