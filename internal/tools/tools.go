// Package tools defines typed tool registry schemas and tool call records.
package tools

import (
	"context"
	"fmt"

	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/policy"
)

// SideEffectClass describes whether a tool mutates external state.
type SideEffectClass string

const (
	SideEffectNone    SideEffectClass = "none"
	SideEffectRead    SideEffectClass = "read"
	SideEffectWrite   SideEffectClass = "write"
	SideEffectExternal SideEffectClass = "external"
)

// ToolSchema is typed metadata for a registered tool.
type ToolSchema struct {
	Name              string
	Description       string
	InputSchemaJSON   string
	OutputSchemaJSON  string
	InputSchemaVer    string
	OutputSchemaVer   string
	PermissionPolicy  string
	RiskLevel         policy.RiskLevel
	SideEffectClass   SideEffectClass
	TimeoutMillis     int64
	BackgroundSupport bool
}

func (t ToolSchema) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("tool name: empty")
	}
	if t.InputSchemaVer == "" || t.OutputSchemaVer == "" {
		return fmt.Errorf("tool schema versions required")
	}
	return nil
}

// ToolCall is one typed invocation with policy outcome.
type ToolCall struct {
	ID              ids.ToolCallID
	ProjectID       ids.ProjectID
	RunID           ids.RunID
	ToolName        string
	InputArtifactID ids.ArtifactID
	OutputArtifactID ids.ArtifactID
	Status          string
	Decision        policy.Decision
	RiskLevel       policy.RiskLevel
	Error           string
}

func (c ToolCall) Validate() error {
	if err := c.ID.Validate(); err != nil {
		return err
	}
	if err := c.ProjectID.Validate(); err != nil {
		return err
	}
	if c.ToolName == "" {
		return fmt.Errorf("tool_call: tool_name empty")
	}
	return c.Decision.Validate()
}

// Registry looks up tool schemas by name.
type Registry interface {
	Register(schema ToolSchema) error
	Get(name string) (ToolSchema, bool)
	List() []ToolSchema
}

// Executor runs a tool after policy has decided allow.
type Executor interface {
	Execute(ctx context.Context, call ToolCall, input []byte) (output []byte, err error)
}
