// Package factory selects Embedder and Completer adapters from config (ADR-0005).
package factory

import (
	"fmt"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/config"
	"github.com/fastygo/context/internal/models"
	"github.com/fastygo/context/internal/models/fake"
	"github.com/fastygo/context/internal/models/httpjson"
	"github.com/fastygo/context/internal/models/localecho"
	"github.com/fastygo/context/internal/models/localhash"
)

// CompleterOptions tunes fake Completer tool hints used by agent-run PoC.
type CompleterOptions struct {
	ToolHint  string
	ToolInput string
}

// OpenEmbedder selects an Embedder from storage/embedder config.
// Returns the adapter and the embedding_version pin it will emit.
func OpenEmbedder(cfg config.StorageConfig) (models.Embedder, string, error) {
	dim := cfg.Vector.Dimension
	if dim <= 0 {
		dim = config.DefaultEmbeddingDimension
	}
	ver := cfg.Vector.EmbeddingVersion
	if ver == "" {
		ver = config.DefaultEmbeddingVersion
	}
	kind := cfg.Embedder.Kind
	if kind == "" {
		kind = config.EmbedderKindFake
	}

	switch kind {
	case config.EmbedderKindFake, "hash", "fake_hash":
		if err := config.ValidateEmbeddingPin(ver, dim); err != nil {
			return nil, "", err
		}
		return fake.Embedder{Dim: dim, Version: ver}, firstNonEmpty(ver, fake.EmbeddingVersion), nil
	case config.EmbedderKindLocalHash:
		if ver == config.DefaultEmbeddingVersion || ver == fake.EmbeddingVersion {
			ver = localhash.Version
		}
		if err := config.ValidateEmbeddingPin(ver, dim); err != nil {
			return nil, "", err
		}
		if dim <= 0 {
			dim = localhash.DefaultDim
		}
		return localhash.Embedder{Dim: dim, Version: ver}, ver, nil
	case config.EmbedderKindHTTP:
		if cfg.Embedder.Endpoint == "" {
			return nil, "", apperr.New(apperr.Validation, "CONTEXT_EMBEDDER_HTTP_URL required for embedder kind http")
		}
		if err := config.ValidateEmbeddingPin(ver, dim); err != nil {
			return nil, "", err
		}
		return httpjson.Embedder{
			BaseURL:   cfg.Embedder.Endpoint,
			Version:   ver,
			Dimension: dim,
		}, ver, nil
	default:
		return nil, "", apperr.New(apperr.Validation, fmt.Sprintf("unknown embedder kind %q", kind))
	}
}

// OpenCompleter selects a Completer from config. Fake may receive tool hints.
func OpenCompleter(cfg config.StorageConfig, opts CompleterOptions) (models.Completer, string, error) {
	kind := cfg.Completer.Kind
	if kind == "" {
		kind = config.CompleterKindFake
	}
	switch kind {
	case config.CompleterKindFake:
		return fake.Completer{ToolHint: opts.ToolHint, ToolInput: opts.ToolInput}, string(kind), nil
	case config.CompleterKindLocalEcho:
		return localecho.Completer{}, string(kind), nil
	case config.CompleterKindHTTP:
		if cfg.Completer.Endpoint == "" {
			return nil, "", apperr.New(apperr.Validation, "CONTEXT_COMPLETER_HTTP_URL required for completer kind http")
		}
		return httpjson.Completer{BaseURL: cfg.Completer.Endpoint}, string(kind), nil
	default:
		return nil, "", apperr.New(apperr.Validation, fmt.Sprintf("unknown completer kind %q", kind))
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
