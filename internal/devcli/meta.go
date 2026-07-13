package devcli

import (
	"context"
	"time"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/artifacts"
	"github.com/fastygo/context/internal/config"
	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/storage"
	"github.com/fastygo/context/internal/storage/postgres"
)

// MetaCheckResult is CLI JSON for postgres metadata smoke.
type MetaCheckResult struct {
	OK           bool   `json:"ok"`
	Backend      string `json:"backend"`
	ProjectID    string `json:"project_id"`
	SchemaID     string `json:"schema_id"`
	LineageOK    bool   `json:"lineage_ok"`
	TemporalOK   bool   `json:"temporal_ok"`
	DocumentKind string `json:"document_kind"`
}

// MetaCheck writes and re-reads project/artifact/lineage/temporal/document rows
// through the configured metadata store (requires CONTEXT_METADATA_KIND=postgres
// or defaults to postgres DSN when --backend postgres).
func MetaCheck(backend string) (MetaCheckResult, error) {
	cfg, err := config.LoadStorageConfigFromEnv()
	if err != nil {
		return MetaCheckResult{}, err
	}
	if backend == "" {
		backend = string(cfg.Metadata.Kind)
	}
	if backend == "postgres" || backend == string(config.StoreKindPostgres) {
		cfg.Metadata.Kind = config.StoreKindPostgres
	}
	if cfg.Metadata.Kind != config.StoreKindPostgres {
		return MetaCheckResult{}, apperr.New(apperr.Validation, "meta-check requires --backend postgres or CONTEXT_METADATA_KIND=postgres")
	}

	ctx := context.Background()
	store, err := postgres.Open(ctx, cfg.Metadata.DSN)
	if err != nil {
		return MetaCheckResult{}, err
	}
	defer store.Close()

	projectID := ids.ProjectID("cli-meta-" + time.Now().UTC().Format("150405000"))
	now := time.Now().UTC().Truncate(time.Millisecond)
	temporal := &corpus.TemporalMetadata{
		Range:      corpus.TemporalRange{Start: now.Add(-time.Minute), End: now, Basis: corpus.TimeBasisOccurred},
		IngestedAt: now,
	}
	if err := store.PutProject(ctx, corpus.Project{ID: projectID, Name: "cli-meta"}); err != nil {
		return MetaCheckResult{}, err
	}
	if err := store.PutSource(ctx, corpus.Source{
		ID: "s1", ProjectID: projectID, Type: corpus.SourceTypeFile, PathKey: "cli",
		TrustLevel: foundation.TrustProject, TemporalMetadata: temporal,
	}); err != nil {
		return MetaCheckResult{}, err
	}
	if err := store.PutArtifactMeta(ctx, artifacts.Artifact{
		ID: "a1", ProjectID: projectID, MediaType: "application/json", ByteSize: 2,
		Checksum: "ab", StorageURI: "local://a1", ArtifactType: artifacts.TypeBlob,
	}); err != nil {
		return MetaCheckResult{}, err
	}
	if err := store.PutArtifactMeta(ctx, artifacts.Artifact{
		ID: "a2", ProjectID: projectID, MediaType: "application/json", ByteSize: 4,
		Checksum: "cd", StorageURI: "local://a2",
		ArtifactType: artifacts.TypeStructured, SchemaID: "uxspec.screen.v1",
	}); err != nil {
		return MetaCheckResult{}, err
	}
	if err := store.PutArtifactLineage(ctx, artifacts.ArtifactLineage{
		ProjectID: projectID, OutputArtifactID: "a2", InputArtifactIDs: []ids.ArtifactID{"a1"},
		GeneratorID: "cli", GeneratorVersion: "v1", TransformationKind: "derive", CreatedAt: now,
	}); err != nil {
		return MetaCheckResult{}, err
	}
	if err := store.PutDocument(ctx, storage.MetaDocument{
		ProjectID: projectID, Kind: storage.DocumentSense, ID: "sense-1",
		Language: "en", SenseID: "sense-1", Payload: []byte(`{"definition":"fixture"}`),
	}); err != nil {
		return MetaCheckResult{}, err
	}

	art, err := store.GetArtifactMeta(ctx, projectID, "a2")
	if err != nil {
		return MetaCheckResult{}, err
	}
	lineage, err := store.GetArtifactLineage(ctx, projectID, "a2")
	if err != nil {
		return MetaCheckResult{}, err
	}
	src, err := store.GetSource(ctx, projectID, "s1")
	if err != nil {
		return MetaCheckResult{}, err
	}
	doc, err := store.GetDocument(ctx, projectID, storage.DocumentSense, "sense-1")
	if err != nil {
		return MetaCheckResult{}, err
	}

	return MetaCheckResult{
		OK:           true,
		Backend:      "postgres",
		ProjectID:    string(projectID),
		SchemaID:     art.SchemaID,
		LineageOK:    len(lineage.InputArtifactIDs) == 1,
		TemporalOK:   src.TemporalMetadata != nil,
		DocumentKind: string(doc.Kind),
	}, nil
}
