// Package fake provides deterministic tool executors for tests.
package fake

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/policy"
	"github.com/fastygo/context/internal/tools"
)

const ReadSnippetName = "read_snippet"

// ReadSnippetSchema is a read-only tool that returns structured snippet JSON.
func ReadSnippetSchema() tools.ToolSchema {
	return tools.ToolSchema{
		Name:             ReadSnippetName,
		Description:      "Return a structured read-only snippet for a source id",
		InputSchemaJSON:  `{"type":"object","properties":{"source_id":{"type":"string"}}}`,
		OutputSchemaJSON: `{"type":"object","properties":{"source_id":{"type":"string"},"text":{"type":"string"}}}`,
		InputSchemaVer:   "v1",
		OutputSchemaVer:  "v1",
		PermissionPolicy: "read",
		RiskLevel:        policy.RiskLow,
		SideEffectClass:  tools.SideEffectRead,
		TimeoutMillis:    1000,
	}
}

// ReadSnippetExecutor returns deterministic JSON output for tests.
type ReadSnippetExecutor struct {
	// Snippets maps source_id -> text.
	Snippets map[string]string
}

type readInput struct {
	SourceID string `json:"source_id"`
}

type readOutput struct {
	SourceID string `json:"source_id"`
	Text     string `json:"text"`
}

func (e ReadSnippetExecutor) Execute(ctx context.Context, call tools.ToolCall, input []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if call.ToolName != ReadSnippetName {
		return nil, apperr.New(apperr.Validation, "unexpected tool: "+call.ToolName)
	}
	var in readInput
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, apperr.Wrap(apperr.Validation, "tool input", err)
	}
	if in.SourceID == "" {
		return nil, apperr.New(apperr.Validation, "source_id required")
	}
	text, ok := e.Snippets[in.SourceID]
	if !ok {
		return nil, apperr.New(apperr.NotFound, "snippet not found")
	}
	out, err := json.Marshal(readOutput{SourceID: in.SourceID, Text: text})
	if err != nil {
		return nil, fmt.Errorf("marshal tool output: %w", err)
	}
	return out, nil
}

var _ tools.Executor = ReadSnippetExecutor{}
