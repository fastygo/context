package querylang

import (
	"context"
	"sort"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/linguistic"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/index"
	"github.com/fastygo/context/internal/retrieval/merge"
	"github.com/fastygo/context/internal/retrieval/snippet"
	"github.com/fastygo/context/internal/tracing"
)

// Retriever IDs used by operator leaves.
const (
	TermRetrieverID        = "term"
	PhraseRetrieverID      = "exact"
	MorphPhraseRetrieverID = "morphphrase"
)

// DefaultMaxExpansions bounds per-term morphology expansion.
const DefaultMaxExpansions = 24

// Runner executes a parsed operator query over the chunk index.
// Linguistic ports are optional: without them, term matching degrades to
// case/NFC-folded token equality and ~ markers add no expansions.
type Runner struct {
	Index      *index.Memory
	Sparse     retrieval.Retriever // optional extra scoring signal for terms
	Normalizer linguistic.LexicalNormalizer
	Analyzer   linguistic.MorphAnalyzer
	Expander   linguistic.QueryExpander
	Language   linguistic.LanguageCode // default when the query has no lang:
	AdapterID  string                  // explain/trace pin for the ports above
	Recorder   tracing.Recorder
	// RejectExp lists expanded terms that must be ignored (false-positive control).
	RejectExp     map[string]bool
	MaxExpansions int
}

// Result carries deterministic candidates plus the full interpretation.
type Result struct {
	Candidates []retrieval.Candidate
	Explain    Explain
	Events     []tracing.Event
}

// LeafExplain shows how one term/phrase leaf was interpreted.
type LeafExplain struct {
	Kind               string   `json:"kind"` // term|morph_term|phrase|morph_phrase
	Text               string   `json:"text"`
	Expansions         []string `json:"expansions,omitempty"`
	RejectedExpansions []string `json:"rejected_expansions,omitempty"`
	Retrievers         []string `json:"retrievers"`
	Matches            int      `json:"matches"`
}

// Explain is the Lab-facing interpretation of an operator query.
type Explain struct {
	Query     string        `json:"query"`
	Canonical string        `json:"canonical"`
	Language  string        `json:"language,omitempty"`
	AdapterID string        `json:"adapter_id,omitempty"`
	Leaves    []LeafExplain `json:"leaves"`
}

type chunkSet map[ids.ChunkID]retrieval.Candidate

type execState struct {
	plan    retrieval.RetrievalPlan
	queryID ids.QueryID
	lang    linguistic.LanguageCode
	// caches scoped to one Run call
	tokensByChunk map[ids.ChunkID][]tokenSpan
	normCache     map[string]string
	lemmaCache    map[string][]string
	leaves        []LeafExplain
}

// Run parses and executes an operator query against the plan scope.
func (r Runner) Run(ctx context.Context, plan retrieval.RetrievalPlan, queryID ids.QueryID, rawQuery string) (Result, error) {
	if r.Index == nil {
		return Result{}, apperr.New(apperr.Validation, "querylang: chunk index required")
	}
	if err := plan.Validate(); err != nil {
		return Result{}, err
	}
	parsed, err := Parse(rawQuery)
	if err != nil {
		return Result{}, apperr.Wrap(apperr.Validation, "querylang parse", err)
	}
	lang := r.Language
	if parsed.Language != "" {
		lang = linguistic.LanguageCode(parsed.Language)
	}
	st := &execState{
		plan:          plan,
		queryID:       queryID,
		lang:          lang,
		tokensByChunk: map[ids.ChunkID][]tokenSpan{},
		normCache:     map[string]string{},
		lemmaCache:    map[string][]string{},
	}
	set, err := r.eval(ctx, st, parsed.Root)
	if err != nil {
		return Result{}, err
	}
	cands := make([]retrieval.Candidate, 0, len(set))
	for _, c := range set {
		cands = append(cands, c)
	}
	// Deterministic pre-merge order so DedupAndMerge tie-breaks are stable.
	sort.Slice(cands, func(i, j int) bool { return cands[i].ChunkID < cands[j].ChunkID })
	out := merge.DedupAndMerge(cands)

	res := Result{
		Candidates: out,
		Explain: Explain{
			Query:     rawQuery,
			Canonical: parsed.Root.Canonical(),
			Language:  string(lang),
			AdapterID: r.AdapterID,
			Leaves:    st.leaves,
		},
	}
	now := time.Now().UTC()
	res.Events = append(res.Events,
		tracing.Event{
			ID:         ids.TraceEventID(string(queryID) + ":query"),
			ProjectID:  plan.ProjectID,
			RunID:      ids.RunID(queryID),
			Type:       tracing.EventRetrievalQuery,
			Timestamp:  now,
			SnapshotID: plan.SnapshotID,
			Payload: map[string]string{
				"query":     rawQuery,
				"canonical": res.Explain.Canonical,
				"language":  string(lang),
				"layer":     "querylang",
			},
		},
		tracing.Event{
			ID:         ids.TraceEventID(string(queryID) + ":candidates"),
			ProjectID:  plan.ProjectID,
			RunID:      ids.RunID(queryID),
			Type:       tracing.EventRetrievalCandidates,
			Timestamp:  now,
			SnapshotID: plan.SnapshotID,
			Payload: map[string]string{
				"count": itoa(len(out)),
				"layer": "querylang",
			},
		},
	)
	if r.Recorder != nil {
		for _, ev := range res.Events {
			if err := r.Recorder.Append(ctx, ev); err != nil {
				return Result{}, err
			}
		}
	}
	return res, nil
}

func (r Runner) eval(ctx context.Context, st *execState, node *Node) (chunkSet, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch node.Kind {
	case KindTerm:
		return r.termLeaf(ctx, st, node)
	case KindPhrase:
		if node.Morph {
			return r.morphPhraseLeaf(ctx, st, node)
		}
		return r.phraseLeaf(ctx, st, node)
	case KindAnd:
		return r.evalAnd(ctx, st, node)
	case KindOr:
		return r.evalOr(ctx, st, node)
	case KindNot:
		return nil, apperr.New(apperr.Validation, "querylang: negation outside AND scope")
	default:
		return nil, apperr.New(apperr.Validation, "querylang: unknown node kind")
	}
}

func (r Runner) evalAnd(ctx context.Context, st *execState, node *Node) (chunkSet, error) {
	var acc chunkSet
	first := true
	// Positives first, then negations subtract.
	for _, child := range node.Children {
		if child.Kind == KindNot {
			continue
		}
		set, err := r.eval(ctx, st, child)
		if err != nil {
			return nil, err
		}
		if first {
			acc = set
			first = false
			continue
		}
		acc = intersect(acc, set)
	}
	if first {
		return nil, apperr.New(apperr.Validation, "querylang: AND requires a positive operand")
	}
	for _, child := range node.Children {
		if child.Kind != KindNot {
			continue
		}
		neg, err := r.eval(ctx, st, child.Children[0])
		if err != nil {
			return nil, err
		}
		for id := range neg {
			delete(acc, id)
		}
	}
	return acc, nil
}

func (r Runner) evalOr(ctx context.Context, st *execState, node *Node) (chunkSet, error) {
	acc := chunkSet{}
	for _, child := range node.Children {
		set, err := r.eval(ctx, st, child)
		if err != nil {
			return nil, err
		}
		for id, c := range set {
			acc[id] = mergeCandidates(acc[id], c)
		}
	}
	return acc, nil
}

// termLeaf matches a single term (plus explainable expansions) on token
// boundaries, merging optional sparse retriever signal.
func (r Runner) termLeaf(ctx context.Context, st *execState, node *Node) (chunkSet, error) {
	leaf := LeafExplain{Kind: "term", Text: node.Text}
	if node.Morph {
		leaf.Kind = "morph_term"
	}

	terms := map[string]bool{} // normalized term -> via expansion
	origNorm, err := r.normalizeWord(ctx, st, node.Text)
	if err != nil {
		return nil, err
	}
	terms[origNorm] = false

	if r.Expander != nil && st.lang != "" {
		exps, err := r.Expander.Expand(ctx, st.queryID, node.Text, st.lang)
		if err != nil {
			return nil, err
		}
		maxExp := r.MaxExpansions
		if maxExp <= 0 {
			maxExp = DefaultMaxExpansions
		}
		for _, exp := range exps {
			if len(leaf.Expansions) >= maxExp {
				break
			}
			if r.RejectExp[exp.ExpandedTerm] {
				leaf.RejectedExpansions = append(leaf.RejectedExpansions, exp.ExpandedTerm)
				continue
			}
			en, err := r.normalizeWord(ctx, st, exp.ExpandedTerm)
			if err != nil {
				return nil, err
			}
			if _, seen := terms[en]; !seen {
				terms[en] = true
				leaf.Expansions = append(leaf.Expansions, exp.ExpandedTerm)
			}
		}
	}

	set := chunkSet{}
	for _, rec := range r.Index.List(st.plan.ProjectID, st.plan.SnapshotID) {
		if !index.MatchesFilters(rec, st.plan.Filters) {
			continue
		}
		toks, err := r.chunkTokens(ctx, st, rec)
		if err != nil {
			return nil, err
		}
		var firstMatch *tokenSpan
		matched := 0
		viaExpansion := false
		for i := range toks {
			via, ok := terms[toks[i].normalized]
			if !ok {
				continue
			}
			matched++
			if firstMatch == nil {
				firstMatch = &toks[i]
				viaExpansion = via
			}
		}
		if matched == 0 {
			continue
		}
		reasons := []foundation.ScoreReason{foundation.ReasonTokenTerm}
		explanation := "token-boundary term match"
		if viaExpansion {
			reasons = append(reasons, foundation.ReasonWordformExpand)
			explanation = "token-boundary match via morph expansion"
		}
		cand := r.newCandidate(st, rec, TermRetrieverID, reasons, explanation, float64(matched))
		if firstMatch != nil {
			if sn, err := snippet.Extract(rec.Text, rec.TextChecksum, foundation.ByteSpan{
				Start: uint64(firstMatch.start), End: uint64(firstMatch.end),
			}, node.Text, snippet.Options{}); err == nil {
				cand.Snippet = &sn
			}
		}
		set[rec.ChunkID] = mergeCandidates(set[rec.ChunkID], cand)
	}
	leaf.Retrievers = append(leaf.Retrievers, TermRetrieverID)

	if r.Sparse != nil {
		sparseCands, err := r.Sparse.Retrieve(ctx, st.plan, node.Text)
		if err == nil {
			leaf.Retrievers = append(leaf.Retrievers, "sparse")
			for _, c := range sparseCands {
				// Sparse only reinforces token hits; it must not widen the
				// deterministic operator result set on its own.
				if existing, ok := set[c.ChunkID]; ok {
					set[c.ChunkID] = mergeCandidates(existing, c)
				}
			}
		} else if !apperr.Is(err, apperr.Unavailable) {
			return nil, err
		}
	}

	leaf.Matches = len(set)
	st.leaves = append(st.leaves, leaf)
	return set, nil
}

// phraseLeaf keeps exact-retriever semantics: literal substring match.
func (r Runner) phraseLeaf(ctx context.Context, st *execState, node *Node) (chunkSet, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	leaf := LeafExplain{Kind: "phrase", Text: node.Text, Retrievers: []string{PhraseRetrieverID}}
	set := chunkSet{}
	for _, rec := range r.Index.List(st.plan.ProjectID, st.plan.SnapshotID) {
		if !index.MatchesFilters(rec, st.plan.Filters) {
			continue
		}
		if !index.ContainsPhrase(rec.Text, node.Text) {
			continue
		}
		cand := r.newCandidate(st, rec, PhraseRetrieverID,
			[]foundation.ScoreReason{foundation.ReasonExactPhrase},
			"exact phrase match in chunk text", 1)
		if sn, ok := snippet.FromChunk(rec.Text, rec.TextChecksum, node.Text, snippet.Options{}); ok {
			cand.Snippet = &sn
		}
		set[rec.ChunkID] = cand
	}
	leaf.Matches = len(set)
	st.leaves = append(st.leaves, leaf)
	return set, nil
}

// morphPhraseLeaf matches a phrase as a consecutive lemma sequence, so an
// inflected phrase in text matches its citation form in the query.
func (r Runner) morphPhraseLeaf(ctx context.Context, st *execState, node *Node) (chunkSet, error) {
	words := splitWords(node.Text)
	if len(words) == 0 {
		return chunkSet{}, nil
	}
	leaf := LeafExplain{Kind: "morph_phrase", Text: node.Text, Retrievers: []string{MorphPhraseRetrieverID}}

	phraseLemmas := make([][]string, len(words))
	for i, w := range words {
		ls, err := r.lemmaSet(ctx, st, w)
		if err != nil {
			return nil, err
		}
		phraseLemmas[i] = ls
	}

	set := chunkSet{}
	for _, rec := range r.Index.List(st.plan.ProjectID, st.plan.SnapshotID) {
		if !index.MatchesFilters(rec, st.plan.Filters) {
			continue
		}
		toks, err := r.chunkTokens(ctx, st, rec)
		if err != nil {
			return nil, err
		}
		match, ok, err := r.findLemmaSequence(ctx, st, toks, phraseLemmas)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		cand := r.newCandidate(st, rec, MorphPhraseRetrieverID,
			[]foundation.ScoreReason{foundation.ReasonMorphPhrase, foundation.ReasonLemmaMatch},
			"consecutive lemma-sequence phrase match", 1)
		if sn, err := snippet.Extract(rec.Text, rec.TextChecksum, match, node.Text, snippet.Options{}); err == nil {
			cand.Snippet = &sn
		}
		set[rec.ChunkID] = cand
	}
	leaf.Matches = len(set)
	st.leaves = append(st.leaves, leaf)
	return set, nil
}

func (r Runner) findLemmaSequence(ctx context.Context, st *execState, toks []tokenSpan, phrase [][]string) (foundation.ByteSpan, bool, error) {
	if len(toks) < len(phrase) {
		return foundation.ByteSpan{}, false, nil
	}
	for i := 0; i+len(phrase) <= len(toks); i++ {
		all := true
		for j := range phrase {
			tokenLemmas, err := r.lemmaSet(ctx, st, toks[i+j].text)
			if err != nil {
				return foundation.ByteSpan{}, false, err
			}
			if !intersects(tokenLemmas, phrase[j]) {
				all = false
				break
			}
		}
		if all {
			return foundation.ByteSpan{
				Start: uint64(toks[i].start),
				End:   uint64(toks[i+len(phrase)-1].end),
			}, true, nil
		}
	}
	return foundation.ByteSpan{}, false, nil
}

// lemmaSet returns normalized lemma candidates for one word (always includes
// the normalized word itself). Analyzer ambiguity is preserved.
func (r Runner) lemmaSet(ctx context.Context, st *execState, word string) ([]string, error) {
	normWord, err := r.normalizeWord(ctx, st, word)
	if err != nil {
		return nil, err
	}
	if cached, ok := st.lemmaCache[normWord]; ok {
		return cached, nil
	}
	out := []string{normWord}
	if r.Analyzer != nil && st.lang != "" {
		tok := linguistic.TokenOccurrence{
			ID:        ids.TokenID("q:" + normWord),
			ProjectID: st.plan.ProjectID,
			SourceID:  "query",
			ChunkID:   "query",
			Language:  st.lang,
			Script:    "Zyyy",
			Surface:   word,
			Normalized: normWord,
			Span:      foundation.ByteSpan{Start: 0, End: uint64(len(word))},
		}
		analyses, err := r.Analyzer.Analyze(ctx, tok)
		if err != nil {
			return nil, err
		}
		seen := map[string]bool{normWord: true}
		for _, a := range analyses {
			if a.Confidence < 0.4 {
				continue
			}
			lemma, err := r.normalizeWord(ctx, st, string(a.Lemma))
			if err != nil {
				return nil, err
			}
			if !seen[lemma] {
				seen[lemma] = true
				out = append(out, lemma)
			}
			if len(out) >= 5 {
				break
			}
		}
	}
	st.lemmaCache[normWord] = out
	return out, nil
}

func (r Runner) normalizeWord(ctx context.Context, st *execState, word string) (string, error) {
	if cached, ok := st.normCache[word]; ok {
		return cached, nil
	}
	var out string
	if r.Normalizer != nil {
		normed, _, err := r.Normalizer.Normalize(ctx, word, st.lang, "")
		if err != nil {
			return "", err
		}
		out = strings.ToLower(normed)
	} else {
		out = strings.ToLower(norm.NFC.String(word))
	}
	st.normCache[word] = out
	return out, nil
}

func (r Runner) chunkTokens(ctx context.Context, st *execState, rec index.ChunkRecord) ([]tokenSpan, error) {
	if cached, ok := st.tokensByChunk[rec.ChunkID]; ok {
		return cached, nil
	}
	raw := tokenizeText(rec.Text)
	for i := range raw {
		n, err := r.normalizeWord(ctx, st, raw[i].text)
		if err != nil {
			return nil, err
		}
		raw[i].normalized = n
	}
	st.tokensByChunk[rec.ChunkID] = raw
	return raw, nil
}

func (r Runner) newCandidate(st *execState, rec index.ChunkRecord, retrieverID string, reasons []foundation.ScoreReason, explanation string, raw float64) retrieval.Candidate {
	return retrieval.Candidate{
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
			RetrieverID:     retrieverID,
			RawScore:        raw,
			NormalizedScore: 1,
			Weight:          merge.DefaultWeight(retrieverID),
			Reasons:         reasons,
			Explanation:     explanation,
			SnapshotID:      st.plan.SnapshotID,
			ProjectID:       st.plan.ProjectID,
			AnalyzerVersion: rec.AnalyzerVersion,
		}},
	}
}

// tokenSpan is one word token with byte offsets into chunk text.
type tokenSpan struct {
	text       string
	normalized string
	start, end int
}

// tokenizeText splits on non-letter/digit boundaries, keeping hyphens inside
// words, and records byte spans for offset-stable snippets.
func tokenizeText(text string) []tokenSpan {
	var out []tokenSpan
	start := -1
	for i, r := range text {
		if isWordRune(r) {
			if start < 0 {
				start = i
			}
			continue
		}
		if start >= 0 {
			out = append(out, tokenSpan{text: text[start:i], start: start, end: i})
			start = -1
		}
	}
	if start >= 0 {
		out = append(out, tokenSpan{text: text[start:], start: start, end: len(text)})
	}
	return out
}

func splitWords(text string) []string {
	var out []string
	for _, t := range tokenizeText(text) {
		out = append(out, t.text)
	}
	return out
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_'
}

func intersect(a, b chunkSet) chunkSet {
	out := chunkSet{}
	for id, ca := range a {
		if cb, ok := b[id]; ok {
			out[id] = mergeCandidates(ca, cb)
		}
	}
	return out
}

func mergeCandidates(existing retrieval.Candidate, next retrieval.Candidate) retrieval.Candidate {
	if existing.ChunkID == "" {
		return next
	}
	existing.Contributions = append(existing.Contributions, next.Contributions...)
	if existing.Snippet == nil && next.Snippet != nil {
		existing.Snippet = next.Snippet
	}
	return existing
}

func intersects(a, b []string) bool {
	for _, x := range a {
		for _, y := range b {
			if x == y {
				return true
			}
		}
	}
	return false
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
