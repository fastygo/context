// Package policy defines permission and approval decisions outside the model.
package policy

import (
	"fmt"

	"github.com/fastygo/context/internal/ids"
)

// Decision is the allow/ask/deny outcome for a tool or side effect.
type Decision string

const (
	DecisionAllow Decision = "allow"
	DecisionAsk   Decision = "ask"
	DecisionDeny  Decision = "deny"
)

func (d Decision) Validate() error {
	switch d {
	case DecisionAllow, DecisionAsk, DecisionDeny:
		return nil
	case "":
		return fmt.Errorf("policy decision: empty")
	default:
		return fmt.Errorf("policy decision: unknown %q", d)
	}
}

// RiskLevel classifies tool or action risk.
type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

// PolicySnapshot freezes permission and approval policy for one run.
type PolicySnapshot struct {
	ID        ids.PolicyID
	ProjectID ids.ProjectID
	Version   string
	Rules     []Rule
}

func (p PolicySnapshot) Validate() error {
	if err := p.ID.Validate(); err != nil {
		return err
	}
	if err := p.ProjectID.Validate(); err != nil {
		return err
	}
	if p.Version == "" {
		return fmt.Errorf("policy_snapshot version: empty")
	}
	return nil
}

// Rule is a named policy clause.
type Rule struct {
	Name     string
	ToolName string
	Decision Decision
	Risk     RiskLevel
}
