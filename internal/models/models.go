// Package models defines LLM, embedding, and reranker adapter ports (ADR-0005).
package models

import (
	"context"
	"fmt"

	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/retrieval"
)

// ModelCall records one model invocation for replay.
type ModelCall struct {
	ID              ids.ModelCallID
	ProjectID       ids.ProjectID
	RunID           ids.RunID
	PackID          ids.PackID
	ProviderID      string
	ModelVersion    string
	InputChecksum   string
	OutputArtifact  ids.ArtifactID // optional; long outputs stored as artifacts
	Status          string
}

func (m ModelCall) Validate() error {
	if err := m.ID.Validate(); err != nil {
		return err
	}
	if err := m.ProjectID.Validate(); err != nil {
		return err
	}
	if m.ProviderID == "" || m.ModelVersion == "" {
		return fmt.Errorf("model_call: provider and model_version required")
	}
	return nil
}

// ChatMessage is a provider-neutral message unit.
type ChatMessage struct {
	Role    string
	Content string
}

// CompletionRequest is a deterministic-friendly model request.
type CompletionRequest struct {
	ProjectID ids.ProjectID
	Pack      retrieval.ContextPack
	Messages  []ChatMessage
}

// CompletionResult is structured model output.
type CompletionResult struct {
	Text       string
	ModelCall  ModelCall
	ToolHints  []string
}

// Completer is the LLM port; fake-first for tests.
type Completer interface {
	Complete(ctx context.Context, req CompletionRequest) (CompletionResult, error)
}

// Embedder produces dense vectors for chunk text.
type Embedder interface {
	Embed(ctx context.Context, texts []string) (vectors [][]float32, modelVersion string, err error)
}

// Reranker adapts cross-encoder style passage scoring.
// Wire via retrieval/rerank.ModelAdapter onto hybrid.Engine.Reranker (C11).
type Reranker interface {
	Rerank(ctx context.Context, query string, passages []string) ([]float64, error)
}
