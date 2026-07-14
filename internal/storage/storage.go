// Package storage defines metadata store ports without durable adapters.
package storage

import (
	"context"
	"time"

	"github.com/fastygo/context/internal/agentruntime"
	"github.com/fastygo/context/internal/artifacts"
	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/tools"
	"github.com/fastygo/context/internal/tracing"
)

// MetadataStore is the replaceable relational/metadata port (memory first, PostgreSQL later).
type MetadataStore interface {
	ProjectStore
	SourceStore
	ChunkStore
	SnapshotStore
	PackStore
	FocusStore
	RunStore
	ToolCallStore
	TraceStore
	ArtifactLineageStore
}

type ProjectStore interface {
	PutProject(ctx context.Context, project corpus.Project) error
	GetProject(ctx context.Context, id ids.ProjectID) (corpus.Project, error)
	ListProjects(ctx context.Context) ([]corpus.Project, error)
	// DeleteProject removes the project and cascaded metadata rows (stabilization C7).
	DeleteProject(ctx context.Context, id ids.ProjectID) error
}

type SourceStore interface {
	PutSource(ctx context.Context, source corpus.Source) error
	GetSource(ctx context.Context, projectID ids.ProjectID, sourceID ids.SourceID) (corpus.Source, error)
	ListSources(ctx context.Context, projectID ids.ProjectID) ([]corpus.Source, error)
	// TombstoneSource soft-deletes a source (stabilization C1). Idempotent when
	// already tombstoned. Re-ingest via PutSource with nil TombstonedAt revives.
	TombstoneSource(ctx context.Context, projectID ids.ProjectID, sourceID ids.SourceID, at time.Time) error
}

type ChunkStore interface {
	PutChunk(ctx context.Context, chunk corpus.Chunk) error
	GetChunk(ctx context.Context, projectID ids.ProjectID, chunkID ids.ChunkID) (corpus.Chunk, error)
	ListChunks(ctx context.Context, projectID ids.ProjectID, snapshotID ids.SnapshotID) ([]corpus.Chunk, error)
}

type SnapshotStore interface {
	PutSnapshot(ctx context.Context, snapshot indexing.IndexSnapshot) error
	GetSnapshot(ctx context.Context, projectID ids.ProjectID, snapshotID ids.SnapshotID) (indexing.IndexSnapshot, error)
	SetActiveSnapshot(ctx context.Context, projectID ids.ProjectID, snapshotID ids.SnapshotID) error
}

type PackStore interface {
	PutPack(ctx context.Context, pack retrieval.ContextPack) error
	GetPack(ctx context.Context, projectID ids.ProjectID, packID ids.PackID) (retrieval.ContextPack, error)
}

// FocusStore persists FocusProfile lenses (Chunk 17).
type FocusStore interface {
	PutFocus(ctx context.Context, focus retrieval.FocusProfile) error
	GetFocus(ctx context.Context, projectID ids.ProjectID, focusID ids.FocusID) (retrieval.FocusProfile, error)
	ListFocus(ctx context.Context, projectID ids.ProjectID) ([]retrieval.FocusProfile, error)
}

type RunStore interface {
	PutRun(ctx context.Context, run agentruntime.AgentRun) error
	GetRun(ctx context.Context, projectID ids.ProjectID, runID ids.RunID) (agentruntime.AgentRun, error)
}

type ToolCallStore interface {
	PutToolCall(ctx context.Context, call tools.ToolCall) error
	GetToolCall(ctx context.Context, projectID ids.ProjectID, callID ids.ToolCallID) (tools.ToolCall, error)
}

type TraceStore interface {
	AppendTrace(ctx context.Context, event tracing.Event) error
	ListTrace(ctx context.Context, projectID ids.ProjectID, runID ids.RunID) ([]tracing.Event, error)
}

// ArtifactLineageStore persists derivation provenance independently from
// runtime traces.
type ArtifactLineageStore interface {
	PutArtifactLineage(ctx context.Context, lineage artifacts.ArtifactLineage) error
	GetArtifactLineage(ctx context.Context, projectID ids.ProjectID, outputArtifactID ids.ArtifactID) (artifacts.ArtifactLineage, error)
	ListArtifactLineage(ctx context.Context, projectID ids.ProjectID) ([]artifacts.ArtifactLineage, error)
}

// ArtifactMetaStore optionally indexes artifact metadata separately from blob bytes.
type ArtifactMetaStore interface {
	PutArtifactMeta(ctx context.Context, artifact artifacts.Artifact) error
	GetArtifactMeta(ctx context.Context, projectID ids.ProjectID, artifactID ids.ArtifactID) (artifacts.Artifact, error)
	ListArtifacts(ctx context.Context, projectID ids.ProjectID) ([]artifacts.Artifact, error)
}

// DocumentKind classifies adapter-neutral metadata documents persisted without
// importing language or lexicon adapter packages into the storage layer.
type DocumentKind string

const (
	DocumentTokenOccurrence DocumentKind = "token_occurrence"
	DocumentMorphAnalysis   DocumentKind = "morph_analysis"
	DocumentQueryExpansion  DocumentKind = "query_expansion"
	DocumentSense           DocumentKind = "sense"
	DocumentConcept         DocumentKind = "concept"
	DocumentAttestation     DocumentKind = "attestation"
	DocumentVariant         DocumentKind = "variant"
	DocumentMWE             DocumentKind = "multiword_expression"
	DocumentLexiconSource   DocumentKind = "lexicon_source"
	DocumentRegister        DocumentKind = "register"
	DocumentDialectRegion   DocumentKind = "dialect_region"
	DocumentTimePeriod      DocumentKind = "time_period"
	DocumentEvaluation      DocumentKind = "evaluation"
	DocumentManifestNode    DocumentKind = "manifest_node"
	DocumentChunkAlias      DocumentKind = "chunk_alias"
)

// MetaDocument is a filterable JSON document for linguistic/lexicographic rows.
type MetaDocument struct {
	ProjectID          ids.ProjectID
	Kind               DocumentKind
	ID                 string
	Language           string
	LexemeID           string
	SenseID            string
	ConceptID          string
	Region             string
	Register           string
	TimePeriod         string
	LexiconSourceID    string
	SourceAuthority    string
	AnalyzerVersion    string
	DictionaryVersion  string
	SnapshotID         ids.SnapshotID
	ChunkID            ids.ChunkID
	Payload            []byte
}

// DocumentStore persists adapter-neutral JSON documents (Chunk 11).
type DocumentStore interface {
	PutDocument(ctx context.Context, doc MetaDocument) error
	GetDocument(ctx context.Context, projectID ids.ProjectID, kind DocumentKind, id string) (MetaDocument, error)
	ListDocuments(ctx context.Context, projectID ids.ProjectID, kind DocumentKind) ([]MetaDocument, error)
}

// TxRunner exposes transactional boundaries for indexing and run updates.
type TxRunner interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
