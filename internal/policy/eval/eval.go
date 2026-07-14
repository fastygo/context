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

// Decide returns the policy decision for a tool.
// Explicit rules win. When no rule matches, write/external side effects require
// approval (ask) before Default is applied — ADR-0034 / C6 baseline.
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
	switch schema.SideEffectClass {
	case tools.SideEffectWrite, tools.SideEffectExternal:
		return policy.DecisionAsk, nil
	}
	if e.Default != "" {
		return e.Default, nil
	}
	return policy.DecisionDeny, nil
}
