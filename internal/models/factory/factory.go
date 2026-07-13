// Package factory selects Embedder adapters from config (ADR-0005).
package factory

import (
	"fmt"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/config"
	"github.com/fastygo/context/internal/models"
	"github.com/fastygo/context/internal/models/fake"
	"github.com/fastygo/context/internal/models/localhash"
)

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
	default:
		return nil, "", apperr.New(apperr.Validation, fmt.Sprintf("unknown embedder kind %q", kind))
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
