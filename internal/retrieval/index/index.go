// Package index provides an in-memory chunk corpus for deterministic retrieval tests.
package index

import (
	"strings"

	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/lexicon"
	"github.com/fastygo/context/internal/retrieval"
)

// ChunkRecord is one searchable chunk with optional lexical/lexicographic metadata.
type ChunkRecord struct {
	ProjectID        ids.ProjectID
	SnapshotID       ids.SnapshotID
	ChunkID          ids.ChunkID
	SourceID         ids.SourceID
	Span             foundation.ByteSpan
	Text             string
	TextChecksum     foundation.ChecksumHex
	TrustLevel       foundation.TrustLevel
	Lemmas           []string
	Wordforms        []string
	SenseIDs         []ids.SenseID
	ConceptIDs       []ids.ConceptID
	AttestationIDs   []ids.AttestationID
	Register         lexicon.Register
	Region           lexicon.DialectRegion
	TimePeriod       lexicon.TimePeriod
	LexiconSourceID  ids.LexiconSourceID
	SourceAuthority  string
	TemporalMetadata *corpus.TemporalMetadata
	Language         string // BCP 47; empty means unknown
	AnalyzerVersion  string // language adapter / analyzer pin for explainability
}

// Memory is a project/snapshot scoped chunk index.
type Memory struct {
	chunks []ChunkRecord
}

// NewMemory returns an empty in-memory index.
func NewMemory(records ...ChunkRecord) *Memory {
	m := &Memory{}
	m.chunks = append(m.chunks, records...)
	return m
}

func (m *Memory) Add(records ...ChunkRecord) {
	m.chunks = append(m.chunks, records...)
}

func (m *Memory) List(projectID ids.ProjectID, snapshotID ids.SnapshotID) []ChunkRecord {
	out := make([]ChunkRecord, 0)
	for _, c := range m.chunks {
		if c.ProjectID == projectID && c.SnapshotID == snapshotID {
			out = append(out, c)
		}
	}
	return out
}

func (m *Memory) Get(projectID ids.ProjectID, snapshotID ids.SnapshotID, chunkID ids.ChunkID) (ChunkRecord, bool) {
	for _, c := range m.chunks {
		if c.ProjectID == projectID && c.SnapshotID == snapshotID && c.ChunkID == chunkID {
			return c, true
		}
	}
	return ChunkRecord{}, false
}

// MatchesFilters reports whether a chunk satisfies RetrievalFilters (empty fields ignored).
func MatchesFilters(c ChunkRecord, f retrieval.RetrievalFilters) bool {
	if f.SenseID != "" && !containsSense(c.SenseIDs, f.SenseID) {
		return false
	}
	if f.ConceptID != "" && !containsConcept(c.ConceptIDs, f.ConceptID) {
		return false
	}
	if f.AttestationID != "" && !containsAttestation(c.AttestationIDs, f.AttestationID) {
		return false
	}
	if f.Register != "" && c.Register != f.Register {
		return false
	}
	if f.DialectRegion != "" && c.Region != f.DialectRegion {
		return false
	}
	if f.TimePeriod != "" && c.TimePeriod != f.TimePeriod {
		return false
	}
	if f.LexiconSourceID != "" && c.LexiconSourceID != f.LexiconSourceID {
		return false
	}
	if f.SourceAuthority != "" && c.SourceAuthority != f.SourceAuthority {
		return false
	}
	if f.Language != "" && c.Language != f.Language {
		return false
	}
	if !f.MatchesTemporal(c.TemporalMetadata) {
		return false
	}
	return true
}

func containsSense(idsList []ids.SenseID, want ids.SenseID) bool {
	for _, id := range idsList {
		if id == want {
			return true
		}
	}
	return false
}

func containsConcept(idsList []ids.ConceptID, want ids.ConceptID) bool {
	for _, id := range idsList {
		if id == want {
			return true
		}
	}
	return false
}

func containsAttestation(idsList []ids.AttestationID, want ids.AttestationID) bool {
	for _, id := range idsList {
		if id == want {
			return true
		}
	}
	return false
}

// ContainsPhrase reports a case-sensitive substring match.
func ContainsPhrase(text, query string) bool {
	if query == "" {
		return false
	}
	return strings.Contains(text, query)
}

// ContainsFolded reports NFC-ish simple lower-case match for keyword tests.
func ContainsFolded(text, query string) bool {
	if query == "" {
		return false
	}
	return strings.Contains(strings.ToLower(text), strings.ToLower(query))
}
