// Package quota evaluates soft project quotas outside the model (ADR-0025 / Chunk 28).
package quota

import (
	"fmt"

	"github.com/fastygo/context/internal/policy"
)

// Limits are project-scoped soft caps. Zero means unlimited.
type Limits struct {
	MaxChunks int `json:"max_chunks,omitempty"`
	MaxPacks  int `json:"max_packs,omitempty"`
	MaxRuns   int `json:"max_runs,omitempty"`
	// SoftAskPercent is 1–100; when usage reaches this percent of a limit the
	// decision becomes ask (warn). At 100% (or used >= max) it becomes deny.
	// Zero defaults to 80 when any max is set.
	SoftAskPercent int `json:"soft_ask_percent,omitempty"`
}

// Usage is current workspace counters (from metrics/state).
type Usage struct {
	Chunks int `json:"chunks"`
	Packs  int `json:"packs"`
	Runs   int `json:"runs"`
}

// Breach describes one resource that hit soft or hard quota.
type Breach struct {
	Resource string          `json:"resource"` // chunks|packs|runs
	Used     int             `json:"used"`
	Limit    int             `json:"limit"`
	Decision policy.Decision `json:"decision"` // ask|deny
	Reason   string          `json:"reason"`
}

// Status is the Lab-facing quota evaluation result.
type Status struct {
	OK       bool            `json:"ok"`
	Decision policy.Decision `json:"decision"`
	Limits   Limits          `json:"limits"`
	Usage    Usage           `json:"usage"`
	Breaches []Breach        `json:"breaches,omitempty"`
	Notes    []string        `json:"notes,omitempty"`
}

// Enabled reports whether any hard cap is configured.
func (l Limits) Enabled() bool {
	return l.MaxChunks > 0 || l.MaxPacks > 0 || l.MaxRuns > 0
}

// Evaluate returns allow/ask/deny for current usage vs limits.
func Evaluate(limits Limits, usage Usage) Status {
	st := Status{
		OK:       true,
		Decision: policy.DecisionAllow,
		Limits:   limits,
		Usage:    usage,
	}
	if !limits.Enabled() {
		st.Notes = append(st.Notes, "no project quotas configured")
		return st
	}
	soft := limits.SoftAskPercent
	if soft <= 0 {
		soft = 80
	}
	if soft > 100 {
		soft = 100
	}

	check := func(name string, used, max int) {
		if max <= 0 {
			return
		}
		if used >= max {
			st.Breaches = append(st.Breaches, Breach{
				Resource: name, Used: used, Limit: max, Decision: policy.DecisionDeny,
				Reason: fmt.Sprintf("%s at or over hard limit (%d/%d)", name, used, max),
			})
			return
		}
		askAt := (max * soft) / 100
		if askAt < 1 {
			askAt = 1
		}
		if used >= askAt {
			st.Breaches = append(st.Breaches, Breach{
				Resource: name, Used: used, Limit: max, Decision: policy.DecisionAsk,
				Reason: fmt.Sprintf("%s at soft threshold (%d/%d, ask>=%d%%)", name, used, max, soft),
			})
		}
	}
	check("chunks", usage.Chunks, limits.MaxChunks)
	check("packs", usage.Packs, limits.MaxPacks)
	check("runs", usage.Runs, limits.MaxRuns)

	for _, b := range st.Breaches {
		if b.Decision == policy.DecisionDeny {
			st.Decision = policy.DecisionDeny
			st.OK = false
			return st
		}
	}
	for _, b := range st.Breaches {
		if b.Decision == policy.DecisionAsk {
			st.Decision = policy.DecisionAsk
			return st
		}
	}
	return st
}

// AllowWrite reports whether a mutating operation of the given resource class
// should proceed. Deny blocks; ask and allow proceed (ask is advisory).
func (s Status) AllowWrite() bool {
	return s.Decision != policy.DecisionDeny
}

// BlocksResource is true when the named counter is already at or over its hard
// limit (used >= max). Soft ask does not block. Call before ingest/pack/run.
func BlocksResource(limits Limits, usage Usage, resource string) bool {
	switch resource {
	case "chunks":
		return limits.MaxChunks > 0 && usage.Chunks >= limits.MaxChunks
	case "packs":
		return limits.MaxPacks > 0 && usage.Packs >= limits.MaxPacks
	case "runs":
		return limits.MaxRuns > 0 && usage.Runs >= limits.MaxRuns
	default:
		return false
	}
}
