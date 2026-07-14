package fake

import (
	"context"
	"encoding/json"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/policy"
	"github.com/fastygo/context/internal/tools"
)

const WriteNoteName = "write_note"

// WriteNoteSchema is a write side-effect tool for approval baseline tests (C6).
func WriteNoteSchema() tools.ToolSchema {
	return tools.ToolSchema{
		Name:             WriteNoteName,
		Description:      "Write a note (mutates external state in real deployments)",
		InputSchemaJSON:  `{"type":"object","properties":{"text":{"type":"string"}}}`,
		OutputSchemaJSON: `{"type":"object","properties":{"ok":{"type":"boolean"}}}`,
		InputSchemaVer:   "v1",
		OutputSchemaVer:  "v1",
		PermissionPolicy: "write",
		RiskLevel:        policy.RiskHigh,
		SideEffectClass:  tools.SideEffectWrite,
		TimeoutMillis:    1000,
	}
}

// WriteNoteExecutor is a deterministic write tool for tests.
type WriteNoteExecutor struct{}

func (WriteNoteExecutor) Execute(ctx context.Context, call tools.ToolCall, input []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if call.ToolName != WriteNoteName {
		return nil, apperr.New(apperr.Validation, "unexpected tool: "+call.ToolName)
	}
	_ = input
	return json.Marshal(map[string]bool{"ok": true})
}

var _ tools.Executor = WriteNoteExecutor{}
