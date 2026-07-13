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
	ProviderID   = "fake"
	ModelVersion = "fake-v1"
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

var _ models.Completer = Completer{}
