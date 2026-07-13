// Package readiness probes configured storage/model backends (Chunk 29).
package readiness

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/config"
	"github.com/fastygo/context/internal/models/factory"
	"github.com/fastygo/context/internal/ops/failinject"
	"github.com/fastygo/context/internal/retrieval/dense/postgresvector"
	"github.com/fastygo/context/internal/retrieval/sparse/postgresfts"
	"github.com/fastygo/context/internal/storage/postgres"
)

// Status is one backend's readiness.
type Status string

const (
	StatusReady       Status = "ready"
	StatusUnavailable Status = "unavailable"
	StatusSkipped     Status = "skipped"
)

// Backend is Lab-facing readiness for one role (no host secrets).
type Backend struct {
	Role   string `json:"role"` // metadata|vector|sparse|embedder|artifact|completer
	Kind   string `json:"kind"`
	Status Status `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// Report aggregates backend readiness for health/metrics.
type Report struct {
	OK       bool      `json:"ok"`       // process can answer
	Ready    bool      `json:"ready"`    // all non-skipped backends ready
	Degraded bool      `json:"degraded"` // at least one unavailable among probed
	Backends []Backend `json:"backends"`
}

// Options tunes which live probes run.
type Options struct {
	// ProbeVector forces a vector ping even when dense is not enabled.
	ProbeVector bool
	// DenseEnabled marks vector as required (not skipped) when true.
	DenseEnabled bool
}

// Probe checks configured backends. Offline kinds stay ready unless fail-injected.
func Probe(ctx context.Context, cfg config.StorageConfig, opts Options) Report {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rep := Report{OK: true, Ready: true}
	rep.Backends = append(rep.Backends, probeMetadata(ctx, cfg))
	rep.Backends = append(rep.Backends, probeSparse(ctx, cfg))
	rep.Backends = append(rep.Backends, probeVector(ctx, cfg, opts))
	rep.Backends = append(rep.Backends, probeEmbedder(cfg))
	rep.Backends = append(rep.Backends, probeArtifact(cfg))
	rep.Backends = append(rep.Backends, probeCompleter(cfg))

	for _, b := range rep.Backends {
		if b.Status == StatusUnavailable {
			rep.Degraded = true
			rep.Ready = false
		}
	}
	return rep
}

func probeMetadata(ctx context.Context, cfg config.StorageConfig) Backend {
	b := Backend{Role: failinject.Metadata, Kind: string(cfg.Metadata.Kind)}
	if err := failinject.Check(failinject.Metadata); err != nil {
		b.Status = StatusUnavailable
		b.Reason = err.Error()
		return b
	}
	store, closer, err := postgres.OpenFromConfig(ctx, cfg)
	if closer != nil {
		defer closer()
	}
	if err != nil {
		b.Status = StatusUnavailable
		b.Reason = err.Error()
		return b
	}
	_ = store
	b.Status = StatusReady
	return b
}

func probeSparse(ctx context.Context, cfg config.StorageConfig) Backend {
	b := Backend{Role: failinject.Sparse, Kind: string(cfg.Sparse.Kind)}
	if err := failinject.Check(failinject.Sparse); err != nil {
		b.Status = StatusUnavailable
		b.Reason = err.Error()
		return b
	}
	if cfg.Sparse.Kind == config.StoreKindPostgresFTS {
		client, err := postgresfts.Open(ctx, cfg.Sparse.DSN)
		if err != nil {
			b.Status = StatusUnavailable
			b.Reason = err.Error()
			return b
		}
		client.Close()
		b.Status = StatusReady
		return b
	}
	b.Status = StatusReady
	return b
}

func probeVector(ctx context.Context, cfg config.StorageConfig, opts Options) Backend {
	b := Backend{Role: failinject.Vector, Kind: string(cfg.Vector.Kind)}
	required := opts.DenseEnabled || opts.ProbeVector
	if !required {
		b.Status = StatusSkipped
		b.Reason = "dense not enabled"
		return b
	}
	if err := failinject.Check(failinject.Vector); err != nil {
		b.Status = StatusUnavailable
		b.Reason = err.Error()
		return b
	}
	store, err := postgresvector.Open(ctx, cfg.Vector.DSN, postgresvector.Config{
		Collection: cfg.Vector.Collection,
		Dimension:  cfg.Vector.Dimension,
		Metric:     cfg.Vector.Metric,
	})
	if err != nil {
		b.Status = StatusUnavailable
		b.Reason = err.Error()
		return b
	}
	store.Close()
	b.Status = StatusReady
	return b
}

func probeEmbedder(cfg config.StorageConfig) Backend {
	kind := string(cfg.Embedder.Kind)
	if kind == "" {
		kind = string(config.DefaultEmbedderKind)
	}
	b := Backend{Role: failinject.Embedder, Kind: kind}
	if err := failinject.Check(failinject.Embedder); err != nil {
		b.Status = StatusUnavailable
		b.Reason = err.Error()
		return b
	}
	if _, _, err := factory.OpenEmbedder(cfg); err != nil {
		b.Status = StatusUnavailable
		b.Reason = err.Error()
		return b
	}
	b.Status = StatusReady
	return b
}

func probeArtifact(cfg config.StorageConfig) Backend {
	b := Backend{Role: failinject.Artifact, Kind: string(cfg.Artifact.Kind)}
	if err := failinject.Check(failinject.Artifact); err != nil {
		b.Status = StatusUnavailable
		b.Reason = err.Error()
		return b
	}
	if cfg.Artifact.Kind == config.StoreKindLocalFS {
		root := strings.TrimSpace(cfg.Artifact.Root)
		if root == "" {
			b.Status = StatusUnavailable
			b.Reason = "artifact.root empty"
			return b
		}
		if _, err := os.Stat(root); err != nil {
			// Root may not exist yet; still mark ready for local PoC (created on write).
			if !os.IsNotExist(err) {
				b.Status = StatusUnavailable
				b.Reason = err.Error()
				return b
			}
		}
	}
	b.Status = StatusReady
	return b
}

func probeCompleter(cfg config.StorageConfig) Backend {
	kind := string(cfg.Completer.Kind)
	if kind == "" {
		kind = string(config.DefaultCompleterKind)
	}
	b := Backend{Role: failinject.Completer, Kind: kind}
	if err := failinject.Check(failinject.Completer); err != nil {
		b.Status = StatusUnavailable
		b.Reason = err.Error()
		return b
	}
	if _, _, err := factory.OpenCompleter(cfg, factory.CompleterOptions{}); err != nil {
		b.Status = StatusUnavailable
		b.Reason = err.Error()
		return b
	}
	b.Status = StatusReady
	return b
}

// RequireReady returns Unavailable when the report is not ready.
func RequireReady(rep Report) error {
	if rep.Ready {
		return nil
	}
	for _, b := range rep.Backends {
		if b.Status == StatusUnavailable {
			return apperr.New(apperr.Unavailable, b.Role+" backend "+string(b.Status)+": "+b.Reason)
		}
	}
	return apperr.New(apperr.Unavailable, "backends not ready")
}
