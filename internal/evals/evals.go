// Package evals defines evaluation harness records for retrieval and task checks.
package evals

import (
	"fmt"
	"time"

	"github.com/fastygo/context/internal/ids"
)

// Evaluation is a reproducible check for retrieval quality or task correctness.
type Evaluation struct {
	ID         ids.EvalID
	ProjectID  ids.ProjectID
	RunID      ids.RunID
	PackID     ids.PackID
	Kind       string
	Passed     bool
	Score      float64
	Notes      string
	CreatedAt  time.Time
	FixtureID  string
}

func (e Evaluation) Validate() error {
	if err := e.ID.Validate(); err != nil {
		return err
	}
	if err := e.ProjectID.Validate(); err != nil {
		return err
	}
	if e.Kind == "" {
		return fmt.Errorf("evaluation kind: empty")
	}
	return nil
}
