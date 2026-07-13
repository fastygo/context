package postgres

import (
	"context"

	"github.com/fastygo/context/internal/config"
	"github.com/fastygo/context/internal/storage"
	"github.com/fastygo/context/internal/storage/memory"
)

// OpenFromConfig opens memory or postgres metadata based on StorageConfig.
func OpenFromConfig(ctx context.Context, cfg config.StorageConfig) (storage.MetadataStore, func(), error) {
	if cfg.Metadata.Kind == config.StoreKindPostgres {
		store, err := Open(ctx, cfg.Metadata.DSN)
		if err != nil {
			return nil, nil, err
		}
		return store, store.Close, nil
	}
	return memory.New(), func() {}, nil
}
