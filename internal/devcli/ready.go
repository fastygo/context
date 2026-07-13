package devcli

import (
	"context"
	"os"
	"strings"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/config"
	"github.com/fastygo/context/internal/ops/readiness"
)

// ReadyResult is CLI/HTTP JSON for readiness.
type ReadyResult = readiness.Report

// Ready probes configured backends (Chunk 29).
func Ready(ctx context.Context) (ReadyResult, error) {
	cfg, err := config.LoadStorageConfigFromEnv()
	if err != nil {
		return ReadyResult{}, err
	}
	opts := readiness.Options{
		DenseEnabled: denseEnabledByEnv(),
		ProbeVector:  denseEnabledByEnv() || forceProbeVector(),
	}
	return readiness.Probe(ctx, cfg, opts), nil
}

func forceProbeVector() bool {
	v := strings.TrimSpace(os.Getenv("CONTEXT_PROBE_VECTOR"))
	return v == "1" || strings.EqualFold(v, "true")
}

// ReadyOrErr returns the report and an Unavailable error when not ready.
func ReadyOrErr(ctx context.Context) (ReadyResult, error) {
	rep, err := Ready(ctx)
	if err != nil {
		return ReadyResult{}, err
	}
	if !rep.Ready {
		return rep, apperr.New(apperr.Unavailable, "one or more backends unavailable")
	}
	return rep, nil
}
