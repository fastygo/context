package storage_test

import (
	"testing"

	"github.com/fastygo/context/internal/storage"
)

func TestMetadataStoreIsInterfaceOnly(t *testing.T) {
	t.Parallel()
	// Chunk 02 ships ports only; durable adapters arrive in later chunks.
	var store storage.MetadataStore
	if store != nil {
		t.Fatal("expected nil interface value without an adapter")
	}
}
