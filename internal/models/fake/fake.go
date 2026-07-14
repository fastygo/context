// Package fake provides a deterministic Completer for tests (ADR-0005).
package fake

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/models"
)

const (
	ProviderID          = "fake"
	ModelVersion        = "fake-v1"
	EmbeddingVersion    = "fake-hash-v1"
	DefaultEmbedDim     = 8
)

// Completer returns deterministic text and optional tool hints from the pack.
type Completer struct {
	// ToolHint forces a tool name hint when non-empty.
	ToolHint string
	// ToolInput is JSON passed through as the suggested tool input when ToolHint is set.
	ToolInput string
}

func (c Completer) Complete(ctx context.Context, req models.CompletionRequest) (models.CompletionResult, error) {
	if err := ctx.Err(); err != nil {
		return models.CompletionResult{}, err
	}
	var b strings.Builder
	b.WriteString("fake-answer:")
	b.WriteString(string(req.Pack.ID))
	b.WriteByte(':')
	for _, item := range req.Pack.EvidenceItems {
		b.WriteString(item.Surface)
		b.WriteByte('|')
	}
	text := b.String()
	sum := sha256.Sum256([]byte(text))
	call := models.ModelCall{
		ID:            ids.ModelCallID("mc_" + hex.EncodeToString(sum[:8])),
		ProjectID:     req.ProjectID,
		PackID:        req.Pack.ID,
		ProviderID:    ProviderID,
		ModelVersion:  ModelVersion,
		InputChecksum: hex.EncodeToString(sum[:]),
		Status:        "completed",
	}
	var hints []string
	if c.ToolHint != "" {
		hints = append(hints, c.ToolHint)
		if c.ToolInput != "" {
			hints = append(hints, c.ToolInput)
		}
	}
	return models.CompletionResult{Text: text, ModelCall: call, ToolHints: hints}, nil
}

// Embedder is a deterministic HashEmbed-based embedding adapter.
type Embedder struct {
	Dim     int
	Version string // defaults to EmbeddingVersion (fake-hash-v1)
}

func (e Embedder) Embed(ctx context.Context, texts []string) ([][]float32, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	dim := e.Dim
	if dim <= 0 {
		dim = DefaultEmbedDim
	}
	ver := e.Version
	if ver == "" {
		ver = EmbeddingVersion
	}
	out := make([][]float32, len(texts))
	for i, text := range texts {
		out[i] = hashEmbed(text, dim)
	}
	return out, ver, nil
}

func hashEmbed(text string, dim int) []float32 {
	out := make([]float32, dim)
	for i, r := range text {
		out[i%dim] += float32(int(r)%31) / 31
	}
	return out
}

// Reranker returns deterministic passage scores (length + query overlap).
type Reranker struct{}

func (Reranker) Rerank(ctx context.Context, query string, passages []string) ([]float64, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]float64, len(passages))
	q := strings.ToLower(query)
	for i, p := range passages {
		score := float64(len(p))
		if q != "" && strings.Contains(strings.ToLower(p), q) {
			score += 100
		}
		out[i] = score
	}
	return out, nil
}

var (
	_ models.Completer = Completer{}
	_ models.Embedder  = Embedder{}
	_ models.Reranker  = Reranker{}
)