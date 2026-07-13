package config

import (
	"testing"
)

func TestDefaultStorageConfig(t *testing.T) {
	cfg := DefaultStorageConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if cfg.Metadata.Kind != StoreKindMemory {
		t.Fatalf("metadata kind = %q", cfg.Metadata.Kind)
	}
	if cfg.Vector.Kind != StoreKindPostgresVector {
		t.Fatalf("vector kind = %q", cfg.Vector.Kind)
	}
	if cfg.Sparse.Kind != StoreKindMemory {
		t.Fatalf("sparse kind = %q", cfg.Sparse.Kind)
	}
	if cfg.Artifact.Kind != StoreKindLocalFS {
		t.Fatalf("artifact kind = %q", cfg.Artifact.Kind)
	}
	if cfg.Vector.Dimension != DefaultEmbeddingDimension {
		t.Fatalf("dimension = %d", cfg.Vector.Dimension)
	}
	if cfg.Vector.EmbeddingVersion != DefaultEmbeddingVersion {
		t.Fatalf("embedding version = %q", cfg.Vector.EmbeddingVersion)
	}
	wantDSN := DefaultPostgresDSN()
	if cfg.Vector.DSN != wantDSN {
		t.Fatalf("vector dsn = %q, want %q", cfg.Vector.DSN, wantDSN)
	}
}

func TestLoadStorageConfigFromEnv(t *testing.T) {
	t.Setenv("CONTEXT_PG_HOST", "db.example")
	t.Setenv("CONTEXT_PG_PORT", "6543")
	t.Setenv("CONTEXT_PG_USER", "u")
	t.Setenv("CONTEXT_PG_PASSWORD", "p")
	t.Setenv("CONTEXT_PG_DATABASE", "ctx")
	t.Setenv("CONTEXT_PG_SSLMODE", "require")
	t.Setenv("CONTEXT_EMBEDDING_DIMENSION", "384")
	t.Setenv("CONTEXT_EMBEDDING_VERSION", "model-x")
	t.Setenv("CONTEXT_VECTOR_KIND", "qdrant")
	t.Setenv("CONTEXT_METADATA_KIND", "postgres")
	t.Setenv("CONTEXT_SPARSE_KIND", "postgres_fts")
	t.Setenv("CONTEXT_ARTIFACT_ROOT", "/tmp/arts")

	t.Setenv("CONTEXT_PG_DSN", "")

	cfg, err := LoadStorageConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadStorageConfigFromEnv: %v", err)
	}
	want := "postgres://u:p@db.example:6543/ctx?sslmode=require"
	if cfg.Metadata.DSN != want {
		t.Fatalf("dsn = %q", cfg.Metadata.DSN)
	}
	if cfg.Vector.Dimension != 384 {
		t.Fatalf("dimension = %d", cfg.Vector.Dimension)
	}
	if cfg.Vector.Kind != StoreKindQdrant {
		t.Fatalf("vector kind = %q", cfg.Vector.Kind)
	}
	if cfg.Metadata.Kind != StoreKindPostgres {
		t.Fatalf("metadata kind = %q", cfg.Metadata.Kind)
	}
	if cfg.Sparse.Kind != StoreKindPostgresFTS {
		t.Fatalf("sparse kind = %q", cfg.Sparse.Kind)
	}
	if cfg.Artifact.Root != "/tmp/arts" {
		t.Fatalf("artifact root = %q", cfg.Artifact.Root)
	}
}

func TestLoadStorageConfigFromEnv_DSNOverride(t *testing.T) {
	t.Setenv("CONTEXT_PG_DSN", "postgres://a:b@h:1/d?sslmode=disable")
	t.Setenv("CONTEXT_PG_HOST", "ignored")

	cfg, err := LoadStorageConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadStorageConfigFromEnv: %v", err)
	}
	if cfg.Vector.DSN != "postgres://a:b@h:1/d?sslmode=disable" {
		t.Fatalf("dsn = %q", cfg.Vector.DSN)
	}
}

func TestValidateRejectsZeroDimension(t *testing.T) {
	cfg := DefaultStorageConfig()
	cfg.Vector.Dimension = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateRequiresPostgresDSN(t *testing.T) {
	cfg := DefaultStorageConfig()
	cfg.Metadata.Kind = StoreKindPostgres
	cfg.Metadata.DSN = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error")
	}
}
