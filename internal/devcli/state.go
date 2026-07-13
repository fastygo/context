// Package devcli implements the context-dev developer workflow helpers.
package devcli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/fastygo/context/internal/agentruntime"
	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/tools"
	"github.com/fastygo/context/internal/tracing"
)

const stateFile = "state.json"

// IndexedChunk is a persisted searchable chunk for local CLI workflows.
type IndexedChunk struct {
	ChunkID            ids.ChunkID              `json:"chunk_id"`
	SourceID           ids.SourceID             `json:"source_id"`
	SnapshotID         ids.SnapshotID           `json:"snapshot_id"`
	PathKey            string                   `json:"path_key"`
	RelativePath       string                   `json:"relative_path"`
	SpanStart          uint64                   `json:"span_start"`
	SpanEnd            uint64                   `json:"span_end"`
	Text               string                   `json:"text"`
	TextChecksum       foundation.ChecksumHex   `json:"text_checksum"`
	ChunkHash          foundation.ChecksumHex   `json:"chunk_hash"`
	TrustLevel         foundation.TrustLevel    `json:"trust_level"`
	ChunkerVersion     string                   `json:"chunker_version,omitempty"`
	EmbeddingVersion   string                   `json:"embedding_version,omitempty"`
	MorphVersion       string                   `json:"morph_version,omitempty"`
	DictionaryVersion  string                   `json:"dictionary_version,omitempty"`
	SparseVersion      string                   `json:"sparse_version,omitempty"`
	Language           string                   `json:"language,omitempty"`
}

// State is the durable local PoC workspace persisted under --data.
type State struct {
	Project   corpus.Project           `json:"project"`
	CorpusRoot string                  `json:"corpus_root"`
	Snapshot  indexing.IndexSnapshot   `json:"snapshot"`
	Chunks    []IndexedChunk           `json:"chunks"`
	Packs     []retrieval.ContextPack  `json:"packs"`
	Runs      []agentruntime.AgentRun  `json:"runs"`
	ToolCalls []tools.ToolCall         `json:"tool_calls"`
	Traces    []tracing.Event          `json:"traces"`
	Focuses   []retrieval.FocusProfile `json:"focuses,omitempty"`
	UpdatedAt time.Time                `json:"updated_at"`
}

// Workspace roots local CLI storage.
type Workspace struct {
	DataDir string
}

func (w Workspace) StatePath() string {
	return filepath.Join(w.DataDir, stateFile)
}

func (w Workspace) ArtifactsDir() string {
	return filepath.Join(w.DataDir, "artifacts")
}

func (w Workspace) Load() (State, error) {
	raw, err := os.ReadFile(w.StatePath())
	if err != nil {
		if os.IsNotExist(err) {
			return State{}, apperr.New(apperr.NotFound, "workspace state not found; run init-project")
		}
		return State{}, apperr.Wrap(apperr.Validation, "read state", err)
	}
	var st State
	if err := json.Unmarshal(raw, &st); err != nil {
		return State{}, apperr.Wrap(apperr.Validation, "decode state", err)
	}
	return st, nil
}

func (w Workspace) Save(st State) error {
	if err := os.MkdirAll(w.DataDir, 0o755); err != nil {
		return apperr.Wrap(apperr.Validation, "create data dir", err)
	}
	st.UpdatedAt = time.Now().UTC()
	raw, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.Validation, "encode state", err)
	}
	tmp := w.StatePath() + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return apperr.Wrap(apperr.Validation, "write state tmp", err)
	}
	if err := os.Rename(tmp, w.StatePath()); err != nil {
		return apperr.Wrap(apperr.Validation, "replace state", err)
	}
	return nil
}

// PrintJSON writes stable machine-readable JSON to stdout.
func PrintJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
