// Package harness provides offline contract tests for lexicon resource adapters
// (ADR-0016 / Chunk 18). External TEI/SKOS mappers should pass RunContract
// without changing vector or metadata adapters.
package harness

import (
	"context"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/lexicon"
	"github.com/fastygo/context/internal/linguistic"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/exact"
	"github.com/fastygo/context/internal/retrieval/hybrid"
	"github.com/fastygo/context/internal/retrieval/index"
)

const (
	ProjectID       = "p_harness"
	SenseID         = "sense-run-sport"
	ConceptID       = "concept-running"
	AttestationID   = "att-runners-1"
	LexiconSourceID = "lex-proof-1"
	SourceAuthority = "proof-fixture"
)

// Seed holds typed lexicon fixtures aligned with proof 04-lexicon.json IDs.
type Seed struct {
	Sense       lexicon.Sense
	Concept     lexicon.Concept
	Attestation lexicon.Attestation
	Source      lexicon.LexiconSource
	Chunk       index.ChunkRecord
}

// DefaultSeed returns fixtures aligned with .project/proof/04-lexicon.json IDs.
func DefaultSeed() Seed {
	return Seed{
		Sense: lexicon.Sense{
			ID: SenseID, ProjectID: ProjectID, LexemeID: "lex-run", Language: "en",
			Definition: "to jog / compete on foot", ConceptID: ConceptID,
			Register: "sport", Region: "us", TimePeriod: "2020s",
			LexiconSourceID: LexiconSourceID, SourceAuthority: SourceAuthority,
		},
		Concept: lexicon.Concept{
			ID: ConceptID, ProjectID: ProjectID, PreferredLabel: "running",
			Labels: []string{"running", "jogging"}, LexiconSourceID: LexiconSourceID,
		},
		Attestation: lexicon.Attestation{
			ID: AttestationID, ProjectID: ProjectID, SourceID: "s_lex", ChunkID: "c_lex",
			Span: foundation.ByteSpan{Start: 0, End: 7}, Quote: "runners", Language: "en",
			LexemeID: "lex-run", SenseID: SenseID, ConceptID: ConceptID,
			Region: "us", Register: "sport", SourceAuthority: SourceAuthority,
			ImportVersion: "fixture-v1",
		},
		Source: lexicon.LexiconSource{
			ID: LexiconSourceID, Kind: "fixture", Title: "Harness Lexicon",
			Authority: SourceAuthority, License: "CC0", Version: "fixture-v1",
			LanguageScope: []linguistic.LanguageCode{"en"},
		},
		Chunk: index.ChunkRecord{
			ProjectID: ProjectID, SnapshotID: "snap1", ChunkID: "c_lex", SourceID: "s_lex",
			Span: foundation.ByteSpan{Start: 0, End: 7}, Text: "runners",
			TextChecksum: "h-lex", TrustLevel: foundation.TrustProject,
			Language: "en", SenseIDs: []ids.SenseID{SenseID},
			ConceptIDs: []ids.ConceptID{ConceptID},
			AttestationIDs: []ids.AttestationID{AttestationID},
			Register: "sport", Region: "us", TimePeriod: "2020s",
			LexiconSourceID: LexiconSourceID, SourceAuthority: SourceAuthority,
		},
	}
}

// TB is the testing surface used by RunContract (*testing.T satisfies it).
type TB interface {
	Helper()
	Fatal(args ...any)
	Fatalf(format string, args ...any)
}

// RunContract asserts typed lookups, attestation provenance, and explainable
// sense/concept filters that preserve original chunk text/spans.
func RunContract(t TB, adapter lexicon.ResourceAdapter, seed Seed) {
	t.Helper()
	if adapter == nil {
		t.Fatal("harness: ResourceAdapter required")
	}
	ctx := context.Background()

	if err := seed.Sense.Validate(); err != nil {
		t.Fatalf("sense fixture: %v", err)
	}
	if err := seed.Concept.Validate(); err != nil {
		t.Fatalf("concept fixture: %v", err)
	}
	if err := seed.Attestation.Validate(); err != nil {
		t.Fatalf("attestation fixture: %v", err)
	}
	if err := seed.Source.Validate(); err != nil {
		t.Fatalf("lexicon source fixture: %v", err)
	}

	sense, err := adapter.LookupSense(ctx, ProjectID, seed.Sense.ID)
	if err != nil {
		t.Fatalf("LookupSense: %v", err)
	}
	if sense.Definition == "" || sense.LexemeID == "" {
		t.Fatalf("sense incomplete: %#v", sense)
	}
	if sense.ConceptID != "" && string(sense.ConceptID) == string(sense.LexemeID) {
		t.Fatal("sense must not collapse into lemma/lexeme identity as concept")
	}

	concept, err := adapter.LookupConcept(ctx, ProjectID, seed.Concept.ID)
	if err != nil {
		t.Fatalf("LookupConcept: %v", err)
	}
	if concept.PreferredLabel == "" {
		t.Fatal("concept preferred_label required")
	}

	att, err := adapter.LookupAttestation(ctx, ProjectID, seed.Attestation.ID)
	if err != nil {
		t.Fatalf("LookupAttestation: %v", err)
	}
	if att.Quote == "" || att.SourceAuthority == "" {
		t.Fatalf("attestation must carry quote+authority: %#v", att)
	}
	if err := att.Span.Validate(); err != nil {
		t.Fatalf("attestation span: %v", err)
	}
	if att.Quote != seed.Chunk.Text {
		t.Fatalf("attestation quote=%q must match witnessed chunk text %q", att.Quote, seed.Chunk.Text)
	}

	src, err := adapter.LicenseMetadata(ctx, seed.Source.ID)
	if err != nil {
		t.Fatalf("LicenseMetadata: %v", err)
	}
	if src.Authority == "" || src.Version == "" {
		t.Fatalf("lexicon source incomplete: %#v", src)
	}

	idx := index.NewMemory(seed.Chunk)
	eng := hybrid.Engine{Exact: exact.Retriever{Index: idx}}
	plan := retrieval.RetrievalPlan{
		ID: "lex-harness", ProjectID: ProjectID, SnapshotID: seed.Chunk.SnapshotID,
		Strategies: []retrieval.RetrieverStrategy{{RetrieverID: "exact"}},
		Filters: retrieval.RetrievalFilters{
			SenseID: seed.Sense.ID, ConceptID: seed.Concept.ID,
			AttestationID: seed.Attestation.ID, Register: seed.Sense.Register,
			DialectRegion: seed.Sense.Region, TimePeriod: seed.Sense.TimePeriod,
			LexiconSourceID: seed.Source.ID, SourceAuthority: seed.Source.Authority,
		},
	}
	res, err := eng.Search(ctx, plan, "q_lex_harness", seed.Chunk.Text)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(res.Candidates) != 1 {
		t.Fatalf("expected 1 filtered candidate, got %#v", res.Candidates)
	}
	c := res.Candidates[0]
	if c.TextChecksum != seed.Chunk.TextChecksum {
		t.Fatalf("checksum changed: %q vs %q", c.TextChecksum, seed.Chunk.TextChecksum)
	}
	if c.SourceRef.Span != seed.Chunk.Span {
		t.Fatalf("span not preserved: %#v", c.SourceRef.Span)
	}
	reasons := map[foundation.ScoreReason]bool{}
	for _, contrib := range c.Contributions {
		for _, r := range contrib.Reasons {
			reasons[r] = true
		}
	}
	if !reasons[foundation.ReasonSenseFilter] || !reasons[foundation.ReasonConceptFilter] {
		t.Fatalf("expected sense_filter and concept_filter reasons, got %#v", c.Contributions)
	}
}
