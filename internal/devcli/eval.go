package devcli

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/evals/golden"
	"github.com/fastygo/context/internal/ops"
)

// EvalResult is CLI JSON wrapping the Lab-facing golden report.
type EvalResult struct {
	Report  golden.Report `json:"report"`
	Out     string        `json:"out,omitempty"`
	History string        `json:"history,omitempty"`
}

// RunEval executes the offline golden suite, optionally writes a report file,
// and optionally appends a summary to an append-only JSONL history.
func RunEval(outPath, historyPath string) (EvalResult, error) {
	start := time.Now()
	rep, err := golden.Run(context.Background())
	if err != nil {
		return EvalResult{}, err
	}
	dur := time.Since(start)
	res := EvalResult{Report: rep}
	if outPath != "" {
		dir := filepath.Dir(outPath)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return EvalResult{}, apperr.Wrap(apperr.Validation, "eval out dir", err)
			}
		}
		raw, err := golden.MarshalReport(rep)
		if err != nil {
			return EvalResult{}, err
		}
		if err := os.WriteFile(outPath, raw, 0o644); err != nil {
			return EvalResult{}, apperr.Wrap(apperr.Internal, "write eval report", err)
		}
		res.Out = outPath
	}
	if historyPath != "" {
		if err := ops.AppendEval(historyPath, summaryFromReport(rep, dur)); err != nil {
			return EvalResult{}, err
		}
		res.History = historyPath
	}
	return res, nil
}
