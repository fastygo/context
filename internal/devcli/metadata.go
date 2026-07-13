package devcli

import (
	"context"

	"github.com/fastygo/context/internal/config"
	"github.com/fastygo/context/internal/ops/failinject"
	"github.com/fastygo/context/internal/storage"
	"github.com/fastygo/context/internal/storage/postgres"
)

// MetadataHandle is an opened MetadataStore plus its configured kind.
type MetadataHandle struct {
	Store  storage.MetadataStore
	Kind   config.StoreKind
	Close  func()
	Config config.StorageConfig
}

// OpenMetadata opens memory or postgres metadata from process env config.
// Callers must invoke Close when finished.
func OpenMetadata(ctx context.Context) (MetadataHandle, error) {
	if err := failinject.Check(failinject.Metadata); err != nil {
		return MetadataHandle{}, err
	}
	cfg, err := config.LoadStorageConfigFromEnv()
	if err != nil {
		return MetadataHandle{}, err
	}
	store, closer, err := postgres.OpenFromConfig(ctx, cfg)
	if err != nil {
		return MetadataHandle{}, err
	}
	if closer == nil {
		closer = func() {}
	}
	return MetadataHandle{
		Store:  store,
		Kind:   cfg.Metadata.Kind,
		Close:  closer,
		Config: cfg,
	}, nil
}

// MetadataKindFromEnv reports the configured metadata kind without opening a store.
func MetadataKindFromEnv() (config.StoreKind, error) {
	cfg, err := config.LoadStorageConfigFromEnv()
	if err != nil {
		return "", err
	}
	return cfg.Metadata.Kind, nil
}

// UsesPostgres reports whether durable postgres metadata is enabled.
func (h MetadataHandle) UsesPostgres() bool {
	return h.Kind == config.StoreKindPostgres
}
