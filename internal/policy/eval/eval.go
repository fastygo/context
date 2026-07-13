// Package eval decides allow/ask/deny outside the model.
package eval

import (
	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/policy"
	"github.com/fastygo/context/internal/tools"
)

// Engine evaluates tool calls against a frozen PolicySnapshot.
type Engine struct {
	Snapshot policy.PolicySnapshot
	Default  policy.Decision // used when no rule matches; default deny
}

func (e Engine) Decide(toolName string, schema tools.ToolSchema) (policy.Decision, error) {
	if err := e.Snapshot.Validate(); err != nil {
		return "", apperr.Wrap(apperr.Validation, "policy_snapshot", err)
	}
	for _, rule := range e.Snapshot.Rules {
		if rule.ToolName == toolName || rule.ToolName == "*" {
			if err := rule.Decision.Validate(); err != nil {
				return "", err
			}
			return rule.Decision, nil
		}
	}
	if e.Default != "" {
		return e.Default, nil
	}
	_ = schema
	return policy.DecisionDeny, nil
}
