// Package config holds replaceable storage endpoint settings for local and
// live PoC stacks. Domain packages must not import vendor SDKs from here.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// StoreKind selects a storage adapter without naming vendor types in domain
// code. New backends add kind values; callers keep reading these structs.
type StoreKind string

const (
	StoreKindMemory          StoreKind = "memory"
	StoreKindLocalFS         StoreKind = "localfs"
	StoreKindPostgres        StoreKind = "postgres"
	StoreKindPostgresVector  StoreKind = "postgres_pgvector"
	StoreKindPostgresFTS     StoreKind = "postgres_fts"
	StoreKindObjectStore     StoreKind = "object_store"
	StoreKindQdrant          StoreKind = "qdrant"
	StoreKindContextSparse   StoreKind = "context_sparse"
)

// PoC defaults match docker-compose.yml / .env.example and fake HashEmbed(dim=8).
const (
	DefaultPGHost              = "127.0.0.1"
	DefaultPGPort              = 5432
	DefaultPGUser              = "context"
	DefaultPGPassword          = "context"
	DefaultPGDatabase          = "context"
	DefaultPGSSLMode           = "disable"
	DefaultEmbeddingVersion    = "fake-hash-v1"
	DefaultEmbeddingDimension  = 8
	DefaultVectorMetric        = "cosine"
	DefaultVectorCollection    = "context_dense_v1"
	DefaultArtifactRoot        = ".context/artifacts"
)

// StorageConfig groups the four replaceable store roles (ADR-0014, ADR-0017).
type StorageConfig struct {
	Metadata MetadataStoreConfig `json:"metadata"`
	Vector   VectorStoreConfig   `json:"vector"`
	Sparse   SparseStoreConfig   `json:"sparse"`
	Artifact ArtifactStoreConfig `json:"artifact"`
}

// MetadataStoreConfig configures relational/project metadata.
type MetadataStoreConfig struct {
	Kind StoreKind `json:"kind"`
	DSN  string    `json:"dsn,omitempty"`
}

// VectorStoreConfig configures dense embedding storage behind VectorStore.
// Kind stays backend-neutral; DSN/Endpoint/Collection are adapter inputs.
type VectorStoreConfig struct {
	Kind             StoreKind `json:"kind"`
	DSN              string    `json:"dsn,omitempty"`
	Endpoint         string    `json:"endpoint,omitempty"`
	Collection       string    `json:"collection,omitempty"`
	Dimension        int       `json:"dimension"`
	Metric           string    `json:"metric,omitempty"`
	EmbeddingVersion string    `json:"embedding_version,omitempty"`
}

// SparseStoreConfig configures sparse/FTS search behind SparseSearchClient.
type SparseStoreConfig struct {
	Kind     StoreKind `json:"kind"`
	DSN      string    `json:"dsn,omitempty"`
	Endpoint string    `json:"endpoint,omitempty"`
}

// ArtifactStoreConfig configures blob storage behind ArtifactStore.
type ArtifactStoreConfig struct {
	Kind StoreKind `json:"kind"`
	Root string    `json:"root,omitempty"`
	DSN  string    `json:"dsn,omitempty"` // object-store style endpoints later
}

// DefaultStorageConfig returns the Chunk 09 local PoC wiring:
// memory metadata/sparse, localfs artifacts, postgres_pgvector for dense.
// Live metadata/sparse Postgres adapters arrive in later chunks; DSN is still
// populated so one container can serve both roles without hardcoding vendors.
func DefaultStorageConfig() StorageConfig {
	dsn := DefaultPostgresDSN()
	return StorageConfig{
		Metadata: MetadataStoreConfig{
			Kind: StoreKindMemory,
			DSN:  dsn,
		},
		Vector: VectorStoreConfig{
			Kind:             StoreKindPostgresVector,
			DSN:              dsn,
			Collection:       DefaultVectorCollection,
			Dimension:        DefaultEmbeddingDimension,
			Metric:           DefaultVectorMetric,
			EmbeddingVersion: DefaultEmbeddingVersion,
		},
		Sparse: SparseStoreConfig{
			Kind: StoreKindMemory,
			DSN:  dsn,
		},
		Artifact: ArtifactStoreConfig{
			Kind: StoreKindLocalFS,
			Root: DefaultArtifactRoot,
		},
	}
}

// DefaultPostgresDSN builds the local compose DSN.
func DefaultPostgresDSN() string {
	return BuildPostgresDSN(DefaultPGHost, DefaultPGPort, DefaultPGUser, DefaultPGPassword, DefaultPGDatabase, DefaultPGSSLMode)
}

// BuildPostgresDSN formats a libpq-style URL without importing a driver.
func BuildPostgresDSN(host string, port int, user, password, database, sslmode string) string {
	if port <= 0 {
		port = DefaultPGPort
	}
	if sslmode == "" {
		sslmode = DefaultPGSSLMode
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		user, password, host, port, database, sslmode,
	)
}

// LoadStorageConfigFromEnv starts from DefaultStorageConfig and overlays
// CONTEXT_* variables from the process environment.
func LoadStorageConfigFromEnv() (StorageConfig, error) {
	cfg := DefaultStorageConfig()
	dsn := strings.TrimSpace(os.Getenv("CONTEXT_PG_DSN"))
	if dsn == "" {
		host := envOr("CONTEXT_PG_HOST", DefaultPGHost)
		port, err := envInt("CONTEXT_PG_PORT", DefaultPGPort)
		if err != nil {
			return StorageConfig{}, err
		}
		user := envOr("CONTEXT_PG_USER", DefaultPGUser)
		password := envOr("CONTEXT_PG_PASSWORD", DefaultPGPassword)
		database := envOr("CONTEXT_PG_DATABASE", DefaultPGDatabase)
		sslmode := envOr("CONTEXT_PG_SSLMODE", DefaultPGSSLMode)
		dsn = BuildPostgresDSN(host, port, user, password, database, sslmode)
	}

	cfg.Metadata.DSN = dsn
	cfg.Vector.DSN = dsn
	cfg.Sparse.DSN = dsn

	if v := strings.TrimSpace(os.Getenv("CONTEXT_METADATA_KIND")); v != "" {
		cfg.Metadata.Kind = StoreKind(v)
	}
	if v := strings.TrimSpace(os.Getenv("CONTEXT_VECTOR_KIND")); v != "" {
		cfg.Vector.Kind = StoreKind(v)
	}
	if v := strings.TrimSpace(os.Getenv("CONTEXT_SPARSE_KIND")); v != "" {
		cfg.Sparse.Kind = StoreKind(v)
	}
	if v := strings.TrimSpace(os.Getenv("CONTEXT_ARTIFACT_KIND")); v != "" {
		cfg.Artifact.Kind = StoreKind(v)
	}
	if v := strings.TrimSpace(os.Getenv("CONTEXT_ARTIFACT_ROOT")); v != "" {
		cfg.Artifact.Root = v
	}
	if v := strings.TrimSpace(os.Getenv("CONTEXT_VECTOR_COLLECTION")); v != "" {
		cfg.Vector.Collection = v
	}
	if v := strings.TrimSpace(os.Getenv("CONTEXT_VECTOR_METRIC")); v != "" {
		cfg.Vector.Metric = v
	}
	if v := strings.TrimSpace(os.Getenv("CONTEXT_EMBEDDING_VERSION")); v != "" {
		cfg.Vector.EmbeddingVersion = v
	}
	if raw := strings.TrimSpace(os.Getenv("CONTEXT_EMBEDDING_DIMENSION")); raw != "" {
		dim, err := strconv.Atoi(raw)
		if err != nil {
			return StorageConfig{}, fmt.Errorf("CONTEXT_EMBEDDING_DIMENSION: %w", err)
		}
		cfg.Vector.Dimension = dim
	}

	if err := cfg.Validate(); err != nil {
		return StorageConfig{}, err
	}
	return cfg, nil
}

// Validate checks adapter-agnostic invariants needed before later chunks wire
// live stores. It does not open network connections.
func (c StorageConfig) Validate() error {
	if c.Metadata.Kind == "" {
		return fmt.Errorf("metadata.kind required")
	}
	if c.Vector.Kind == "" {
		return fmt.Errorf("vector.kind required")
	}
	if c.Sparse.Kind == "" {
		return fmt.Errorf("sparse.kind required")
	}
	if c.Artifact.Kind == "" {
		return fmt.Errorf("artifact.kind required")
	}
	if c.Vector.Dimension <= 0 {
		return fmt.Errorf("vector.dimension must be > 0")
	}
	if needsDSN(c.Metadata.Kind) && strings.TrimSpace(c.Metadata.DSN) == "" {
		return fmt.Errorf("metadata.dsn required for kind %q", c.Metadata.Kind)
	}
	if needsDSN(c.Vector.Kind) && strings.TrimSpace(c.Vector.DSN) == "" && strings.TrimSpace(c.Vector.Endpoint) == "" {
		return fmt.Errorf("vector.dsn or vector.endpoint required for kind %q", c.Vector.Kind)
	}
	if needsDSN(c.Sparse.Kind) && strings.TrimSpace(c.Sparse.DSN) == "" && strings.TrimSpace(c.Sparse.Endpoint) == "" {
		return fmt.Errorf("sparse.dsn or sparse.endpoint required for kind %q", c.Sparse.Kind)
	}
	if c.Artifact.Kind == StoreKindLocalFS && strings.TrimSpace(c.Artifact.Root) == "" {
		return fmt.Errorf("artifact.root required for kind %q", c.Artifact.Kind)
	}
	return nil
}

func needsDSN(kind StoreKind) bool {
	switch kind {
	case StoreKindPostgres, StoreKindPostgresVector, StoreKindPostgresFTS:
		return true
	default:
		return false
	}
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return n, nil
}
