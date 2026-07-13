package devcli

import (
	"context"
	"os"
	"path/filepath"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/evals/golden"
)

// EvalResult is CLI JSON wrapping the Lab-facing golden report.
type EvalResult struct {
	Report golden.Report `json:"report"`
	Out    string        `json:"out,omitempty"`
}

// RunEval executes the offline golden suite and optionally writes a JSON report.
func RunEval(outPath string) (EvalResult, error) {
	rep, err := golden.Run(context.Background())
	if err != nil {
		return EvalResult{}, err
	}
	res := EvalResult{Report: rep}
	if outPath == "" {
		return res, nil
	}
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
	return res, nil
}
