// Package runtime provides bounded, immutable, in-process evidence retrieval
// and ContextPack construction. It performs no network or filesystem I/O.
package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/exact"
	"github.com/fastygo/context/internal/retrieval/index"
	"github.com/fastygo/context/internal/retrieval/pack"
	"github.com/fastygo/context/pkg/contextkit"
)

// Version pins the embedded adapter's identity and capability profile.
const Version = "memory-exact-v1"
const MaxSources = 128
const MaxInputBytes = 2 << 20

// Source is one immutable UTF-8 source, indexed as one complete chunk.
// Trust and evidence labels must be assigned by the authorized embedding host.
type Source struct {
	SourceID      string `json:"source_id"`
	Version       string `json:"version"`
	Text          string `json:"text"`
	TrustLevel    string `json:"trust_level"`
	EvidenceClass string `json:"evidence_class"`
}

// Config is copied by New. Zero limits select the hard ceilings; positive
// limits may lower them. All metadata and text count against MaxBytes.
type Config struct {
	ProjectID  string
	Sources    []Source
	MaxBytes   int
	MaxSources int
}

// Snapshot is the complete caller-retainable source manifest. Its ID binds all
// source fields, sorted by source id, and the project and adapter version.
type Snapshot struct {
	ID             string   `json:"id"`
	ProjectID      string   `json:"project_id"`
	RuntimeVersion string   `json:"runtime_version"`
	Sources        []Source `json:"sources"`
}

// Runtime is immutable after New and safe for concurrent method calls.
// The zero value is not usable; callers must use New.
type Runtime struct {
	snapshot Snapshot
	index    *index.Memory
	maxBytes int
}

// Budget bounds selected evidence. MaxChars counts UTF-8 bytes, following the
// existing pack builder. Rejected evidence still occupies response space.
type Budget struct {
	MaxItems          int `json:"max_items"`
	MaxChars          int `json:"max_chars"`
	MaxTokensEstimate int `json:"max_tokens_estimate,omitempty"`
}

// Focus is the supported subset of the Context focus contract.
type Focus struct {
	ID                 string `json:"id"`
	Objective          string `json:"objective"`
	RequiredTrustLevel string `json:"required_trust_level"`
	Budget             Budget `json:"context_budget"`
}

// PackRequest supplies all controls directly; there is no mutable focus store.
type PackRequest struct {
	ProjectID                string   `json:"project_id"`
	TaskID                   string   `json:"task_id,omitempty"`
	Query                    string   `json:"query"`
	Focus                    Focus    `json:"focus"`
	Instructions             []string `json:"instructions,omitempty"`
	PolicyRefs               []string `json:"policy_refs,omitempty"`
	VerificationRequirements []string `json:"verification_requirements,omitempty"`
}

// PackResult preserves the HTTP pack representation and adds the complete
// snapshot needed by a consumer to retain original bytes and source versions.
type PackResult struct {
	contextkit.PackResult
	Snapshot Snapshot `json:"snapshot"`
}

func invalid(message string) error {
	return contextkit.APIError{Code: "validation", Message: message}
}

func validText(s string) bool { return utf8.ValidString(s) && !strings.ContainsRune(s, 0) }

func digest(domain string, value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(append([]byte(domain+"\x00"), raw...))
	return hex.EncodeToString(sum[:]), nil
}

// New validates and freezes sources without opening files, sockets, or stores.
func New(ctx context.Context, cfg Config) (*Runtime, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.ProjectID) == "" || !validText(cfg.ProjectID) {
		return nil, invalid("project_id required and must be valid UTF-8 without NUL")
	}
	if cfg.MaxBytes == 0 {
		cfg.MaxBytes = MaxInputBytes
	}
	if cfg.MaxSources == 0 {
		cfg.MaxSources = MaxSources
	}
	if cfg.MaxBytes < 1 || cfg.MaxBytes > MaxInputBytes || cfg.MaxSources < 1 || cfg.MaxSources > MaxSources {
		return nil, invalid("limits must be positive and within runtime ceilings")
	}
	if len(cfg.Sources) > cfg.MaxSources {
		return nil, invalid("source count exceeds limit")
	}
	used := len(cfg.ProjectID)
	seen := make(map[string]bool, len(cfg.Sources))
	for _, s := range cfg.Sources {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for _, field := range []string{s.SourceID, s.Version, s.Text, s.TrustLevel, s.EvidenceClass} {
			if len(field) > cfg.MaxBytes-used {
				return nil, invalid("source bytes exceed limit")
			}
			used += len(field)
			if !validText(field) {
				return nil, invalid("source fields must be valid UTF-8 without NUL")
			}
		}
		if strings.TrimSpace(s.SourceID) == "" || strings.TrimSpace(s.Version) == "" || s.Text == "" {
			return nil, invalid("source id, version and text required")
		}
		if seen[s.SourceID] {
			return nil, invalid("duplicate source id")
		}
		seen[s.SourceID] = true
		if err := foundation.TrustLevel(s.TrustLevel).Validate(); err != nil {
			return nil, invalid(err.Error())
		}
		if err := foundation.EvidenceClass(s.EvidenceClass).Validate(); err != nil {
			return nil, invalid(err.Error())
		}
	}
	if used > cfg.MaxBytes {
		return nil, invalid("source bytes exceed limit")
	}
	// Clone strings as well as the slice so substrings cannot retain large buffers.
	sources := make([]Source, len(cfg.Sources))
	for i, s := range cfg.Sources {
		sources[i] = Source{strings.Clone(s.SourceID), strings.Clone(s.Version), strings.Clone(s.Text), strings.Clone(s.TrustLevel), strings.Clone(s.EvidenceClass)}
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].SourceID < sources[j].SourceID })
	snapshot := Snapshot{ProjectID: strings.Clone(cfg.ProjectID), RuntimeVersion: Version, Sources: sources}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	if len(raw) > cfg.MaxBytes {
		return nil, invalid("encoded snapshot exceeds limit")
	}
	hash, err := digest("context/memory-snapshot/v1", snapshot)
	if err != nil {
		return nil, err
	}
	snapshot.ID = "snapshot_" + hash
	records := make([]index.ChunkRecord, len(sources))
	for i, s := range sources {
		sum := sha256.Sum256([]byte(s.Text))
		records[i] = index.ChunkRecord{
			ProjectID: ids.ProjectID(snapshot.ProjectID), SnapshotID: ids.SnapshotID(snapshot.ID),
			ChunkID: ids.ChunkID(fmt.Sprintf("chunk_%04d", i)), SourceID: ids.SourceID(s.SourceID),
			Span: foundation.ByteSpan{End: uint64(len(s.Text))}, Text: s.Text,
			TextChecksum: foundation.ChecksumHex(hex.EncodeToString(sum[:])), TrustLevel: foundation.TrustLevel(s.TrustLevel),
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &Runtime{snapshot: snapshot, index: index.NewMemory(records...), maxBytes: cfg.MaxBytes}, nil
}

func (r *Runtime) check(ctx context.Context, projectID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if r == nil || r.index == nil {
		return invalid("runtime must be constructed with New")
	}
	if projectID == "" || projectID != r.snapshot.ProjectID {
		return contextkit.APIError{Code: "permission", Message: "project_id does not match runtime"}
	}
	return nil
}

// Snapshot returns an independently owned manifest; string contents are immutable.
func (r *Runtime) Snapshot() Snapshot {
	if r == nil {
		return Snapshot{}
	}
	out := r.snapshot
	out.Sources = append([]Source(nil), r.snapshot.Sources...)
	return out
}

func (r *Runtime) retrieve(ctx context.Context, projectID, query string) ([]retrieval.Candidate, error) {
	if err := r.check(ctx, projectID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(query) == "" || !validText(query) || len(query) > r.maxBytes {
		return nil, invalid("query must be nonempty bounded UTF-8 without NUL")
	}
	return (exact.Retriever{Index: r.index}).Retrieve(ctx, retrieval.RetrievalPlan{
		ID: "memory_exact", ProjectID: ids.ProjectID(projectID), SnapshotID: ids.SnapshotID(r.snapshot.ID),
		Strategies: []retrieval.RetrieverStrategy{{RetrieverID: exact.RetrieverID}},
	}, query)
}

// Search supports exact case-sensitive phrase retrieval only. Unsupported
// options fail explicitly. Search candidates are not policy-admitted evidence.
func (r *Runtime) Search(ctx context.Context, req contextkit.SearchRequest) (contextkit.SearchResult, error) {
	if err := r.check(ctx, req.ProjectID); err != nil {
		return contextkit.SearchResult{}, err
	}
	if (req.Mode != "" && req.Mode != "exact") || req.Lang != "" || req.FocusID != "" {
		return contextkit.SearchResult{}, invalid("only exact search without stored focus or language options is supported")
	}
	candidates, err := r.retrieve(ctx, req.ProjectID, req.Query)
	if err != nil {
		return contextkit.SearchResult{}, err
	}
	out := contextkit.SearchResult{ProjectID: req.ProjectID, SnapshotID: r.snapshot.ID, Query: req.Query, Mode: "exact", Candidates: []contextkit.Candidate{}}
	raw, err := json.Marshal(candidates)
	if err != nil {
		return out, err
	}
	if err = json.Unmarshal(raw, &out.Candidates); err != nil {
		return out, err
	}
	return out, ctx.Err()
}

// ContextPack builds using the existing deterministic pack builder and its
// checksum contract. The host must enforce its own serialized response limit.
func (r *Runtime) ContextPack(ctx context.Context, req PackRequest) (PackResult, error) {
	if err := r.check(ctx, req.ProjectID); err != nil {
		return PackResult{}, err
	}
	// Bound individual controls and list counts before encoding or cloning.
	total := 0
	fields := []string{req.ProjectID, req.TaskID, req.Query, req.Focus.ID, req.Focus.Objective, req.Focus.RequiredTrustLevel}
	for _, list := range [][]string{req.Instructions, req.PolicyRefs, req.VerificationRequirements} {
		if len(list) > MaxSources {
			return PackResult{}, invalid("control list exceeds limit")
		}
		fields = append(fields, list...)
	}
	for _, field := range fields {
		if !validText(field) || len(field) > r.maxBytes-total {
			return PackResult{}, invalid("invalid or oversized control fields")
		}
		total += len(field)
	}
	b := req.Focus.Budget
	if b.MaxItems < 1 || b.MaxItems > MaxSources || b.MaxChars < 1 || b.MaxChars > r.maxBytes || b.MaxTokensEstimate < 0 || b.MaxTokensEstimate > r.maxBytes {
		return PackResult{}, invalid("invalid or unbounded pack budget")
	}
	instructionBytes := 0
	for _, s := range req.Instructions {
		instructionBytes += len(s)
	}
	if instructionBytes >= b.MaxChars {
		return PackResult{}, invalid("instructions exhaust character budget")
	}
	for _, id := range req.PolicyRefs {
		if strings.TrimSpace(id) == "" {
			return PackResult{}, invalid("empty policy reference")
		}
	}
	focus := retrieval.FocusProfile{
		ID: ids.FocusID(req.Focus.ID), ProjectID: ids.ProjectID(req.ProjectID), Objective: req.Focus.Objective,
		RequiredTrustLevel: foundation.TrustLevel(req.Focus.RequiredTrustLevel),
		ContextBudget:      retrieval.Budget{MaxItems: b.MaxItems, MaxChars: b.MaxChars, MaxTokensEstimate: b.MaxTokensEstimate},
	}
	if err := focus.Validate(); err != nil {
		return PackResult{}, invalid(err.Error())
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return PackResult{}, err
	}
	if len(raw) > r.maxBytes {
		return PackResult{}, invalid("encoded request exceeds limit")
	}
	var frozen PackRequest
	if err := json.Unmarshal(raw, &frozen); err != nil {
		return PackResult{}, err
	}
	req = frozen
	hash, err := digest("context/memory-pack-request/v1", struct {
		SnapshotID string      `json:"snapshot_id"`
		Request    PackRequest `json:"request"`
	}{r.snapshot.ID, req})
	if err != nil {
		return PackResult{}, err
	}
	candidates, err := r.retrieve(ctx, req.ProjectID, req.Query)
	if err != nil {
		return PackResult{}, err
	}
	byID := make(map[string]Source, len(r.snapshot.Sources))
	for _, s := range r.snapshot.Sources {
		byID[s.SourceID] = s
	}
	drafts := make([]pack.DraftItem, 0, len(candidates))
	for _, c := range candidates {
		s := byID[string(c.SourceRef.SourceID)]
		drafts = append(drafts, pack.DraftItem{ID: string(c.ChunkID), Candidate: c, Surface: s.Text, Class: foundation.EvidenceClass(s.EvidenceClass)})
	}
	policies := make([]ids.PolicyID, len(req.PolicyRefs))
	for i, id := range req.PolicyRefs {
		policies[i] = ids.PolicyID(id)
	}
	built, err := (pack.Builder{}).Build(ctx, pack.BuildRequest{
		PackID: ids.PackID("pack_" + hash), PlanID: ids.PlanID("plan_" + hash),
		ProjectID: ids.ProjectID(req.ProjectID), TaskID: ids.TaskID(req.TaskID), Purpose: req.Focus.Objective,
		Focus: focus, Instructions: req.Instructions, PolicyRefs: policies,
		VerificationRequirements: req.VerificationRequirements, Items: drafts,
	})
	if err != nil {
		return PackResult{}, fmt.Errorf("context runtime pack: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return PackResult{}, err
	}
	raw, err = json.Marshal(built)
	if err != nil {
		return PackResult{}, err
	}
	return PackResult{PackResult: contextkit.PackResult{ContextPack: raw, FocusID: req.Focus.ID}, Snapshot: r.Snapshot()}, nil
}
