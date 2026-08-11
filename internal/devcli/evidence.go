package devcli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/artifacts"
	"github.com/fastygo/context/internal/artifacts/localfs"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/policy/isolation"
	"github.com/fastygo/context/internal/storage"
)

// ArtifactPutInput is the application command behind the additive public
// structured-artifact route. Body is transported as base64 by JSON clients.
type ArtifactPutInput struct {
	ProjectID        string
	ArtifactID       string
	MediaType        string
	ArtifactType     string
	SchemaID         string
	SourceID         string
	Body             []byte
	ExpectedChecksum string
	TrustLevel       foundation.TrustLevel
	EvidenceClass    foundation.EvidenceClass
	Lineage          *artifacts.ArtifactLineage
}

// ArtifactView omits adapter storage URIs from public responses.
type ArtifactView struct {
	ID            ids.ArtifactID           `json:"artifact_id"`
	ProjectID     ids.ProjectID            `json:"project_id"`
	SourceID      ids.SourceID             `json:"source_id,omitempty"`
	MediaType     string                   `json:"media_type"`
	ByteSize      int64                    `json:"byte_size"`
	Checksum      foundation.ChecksumHex   `json:"checksum"`
	ArtifactType  string                   `json:"artifact_type"`
	SchemaID      string                   `json:"schema_id,omitempty"`
	TrustLevel    foundation.TrustLevel    `json:"trust_level"`
	EvidenceClass foundation.EvidenceClass `json:"evidence_class"`
}

// ArtifactResult is returned by put/get. Body is omitted by list.
type ArtifactResult struct {
	Artifact ArtifactView               `json:"artifact"`
	Body     []byte                     `json:"body_base64,omitempty"`
	Lineage  *artifacts.ArtifactLineage `json:"lineage,omitempty"`
}

type ArtifactListResult struct {
	Artifacts []ArtifactView `json:"artifacts"`
}

// PutArtifact stores immutable bytes and optional immutable lineage, then
// registers the artifact as searchable project memory.
func PutArtifact(ctx context.Context, dataDir string, in ArtifactPutInput) (ArtifactResult, error) {
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return ArtifactResult{}, err
	}
	if err := isolation.RequireProjectMatch(st.Project.ID, ids.ProjectID(in.ProjectID)); err != nil {
		return ArtifactResult{}, err
	}
	if strings.TrimSpace(in.ArtifactID) == "" || strings.TrimSpace(in.MediaType) == "" {
		return ArtifactResult{}, apperr.New(apperr.Validation, "artifact_id and media_type required")
	}
	if in.TrustLevel == "" {
		in.TrustLevel = foundation.TrustProject
	}
	if err := in.TrustLevel.Validate(); err != nil {
		return ArtifactResult{}, apperr.Wrap(apperr.Validation, "trust_level", err)
	}
	if in.EvidenceClass == "" {
		in.EvidenceClass = foundation.EvidenceSourceText
	}
	if err := in.EvidenceClass.Validate(); err != nil {
		return ArtifactResult{}, apperr.Wrap(apperr.Validation, "evidence_class", err)
	}

	sum := sha256.Sum256(in.Body)
	checksum := foundation.ChecksumHex(hex.EncodeToString(sum[:]))
	if in.ExpectedChecksum != "" && in.ExpectedChecksum != string(checksum) {
		return ArtifactResult{}, apperr.New(apperr.Conflict, "artifact checksum does not match body")
	}

	var lineage *artifacts.ArtifactLineage
	if in.Lineage != nil {
		copy := *in.Lineage
		copy.ProjectID = st.Project.ID
		copy.OutputArtifactID = ids.ArtifactID(in.ArtifactID)
		if copy.CreatedAt.IsZero() {
			if existing := findLineage(st, copy.OutputArtifactID); existing != nil {
				copy.CreatedAt = existing.CreatedAt
			} else {
				copy.CreatedAt = time.Now().UTC()
			}
		}
		if err := copy.Validate(); err != nil {
			return ArtifactResult{}, apperr.Wrap(apperr.Validation, "artifact lineage", err)
		}
		lineage = &copy
	}

	store, err := localfs.New(ws.ArtifactsDir())
	if err != nil {
		return ArtifactResult{}, err
	}
	art, err := store.Put(ctx, st.Project.ID, ids.ArtifactID(in.ArtifactID), in.MediaType, in.Body, &artifacts.PutOptions{
		ArtifactType: in.ArtifactType,
		SchemaID:     in.SchemaID,
		SourceID:     ids.SourceID(in.SourceID),
	})
	if err != nil {
		return ArtifactResult{}, err
	}
	record := ArtifactRecord{Artifact: art, TrustLevel: in.TrustLevel, EvidenceClass: in.EvidenceClass}
	if err := putArtifactRecord(&st, record); err != nil {
		return ArtifactResult{}, err
	}
	if lineage != nil {
		if err := putLineageRecord(&st, *lineage); err != nil {
			return ArtifactResult{}, err
		}
	}
	if err := ws.Save(st); err != nil {
		return ArtifactResult{}, err
	}
	if err := persistPublicArtifact(ctx, st, record, lineage); err != nil {
		return ArtifactResult{}, err
	}
	return ArtifactResult{Artifact: artifactView(record), Body: append([]byte(nil), in.Body...), Lineage: lineage}, nil
}

func GetArtifact(ctx context.Context, dataDir, projectID, artifactID string) (ArtifactResult, error) {
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return ArtifactResult{}, err
	}
	if err := isolation.RequireProjectMatch(st.Project.ID, ids.ProjectID(projectID)); err != nil {
		return ArtifactResult{}, err
	}
	record, ok := findArtifactRecord(st, ids.ArtifactID(artifactID))
	if !ok {
		return ArtifactResult{}, apperr.New(apperr.NotFound, "artifact not found")
	}
	store, err := localfs.New(ws.ArtifactsDir())
	if err != nil {
		return ArtifactResult{}, err
	}
	_, body, err := store.Get(ctx, st.Project.ID, record.Artifact.ID)
	if err != nil {
		return ArtifactResult{}, err
	}
	return ArtifactResult{Artifact: artifactView(record), Body: body, Lineage: findLineage(st, record.Artifact.ID)}, nil
}

func ListArtifacts(dataDir, projectID string) (ArtifactListResult, error) {
	st, err := (Workspace{DataDir: dataDir}).Load()
	if err != nil {
		return ArtifactListResult{}, err
	}
	if err := isolation.RequireProjectMatch(st.Project.ID, ids.ProjectID(projectID)); err != nil {
		return ArtifactListResult{}, err
	}
	out := make([]ArtifactView, 0, len(st.Artifacts))
	for _, record := range st.Artifacts {
		out = append(out, artifactView(record))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return ArtifactListResult{Artifacts: out}, nil
}

func putArtifactRecord(st *State, record ArtifactRecord) error {
	for _, existing := range st.Artifacts {
		if existing.Artifact.ID != record.Artifact.ID {
			continue
		}
		if !reflect.DeepEqual(existing, record) {
			return apperr.New(apperr.Conflict, "artifact policy metadata is immutable")
		}
		return nil
	}
	st.Artifacts = append(st.Artifacts, record)
	return nil
}

func putLineageRecord(st *State, lineage artifacts.ArtifactLineage) error {
	for _, existing := range st.Lineages {
		if existing.OutputArtifactID != lineage.OutputArtifactID {
			continue
		}
		if !reflect.DeepEqual(existing, lineage) {
			return apperr.New(apperr.Conflict, "artifact lineage is immutable")
		}
		return nil
	}
	st.Lineages = append(st.Lineages, lineage)
	return nil
}

func persistPublicArtifact(ctx context.Context, st State, record ArtifactRecord, lineage *artifacts.ArtifactLineage) error {
	handle, err := OpenMetadata(ctx)
	if err != nil {
		return err
	}
	defer handle.Close()
	if !handle.UsesPostgres() {
		return nil
	}
	if err := handle.Store.PutProject(ctx, st.Project); err != nil {
		return err
	}
	meta, ok := handle.Store.(storage.ArtifactMetaStore)
	if !ok {
		return apperr.New(apperr.Unavailable, "artifact metadata store unavailable")
	}
	if err := meta.PutArtifactMeta(ctx, record.Artifact); err != nil {
		return err
	}
	if lineage != nil {
		existing, getErr := handle.Store.GetArtifactLineage(ctx, lineage.ProjectID, lineage.OutputArtifactID)
		if getErr == nil {
			if reflect.DeepEqual(existing, *lineage) {
				return nil
			}
			return apperr.New(apperr.Conflict, "artifact lineage is immutable")
		}
		if !apperr.Is(getErr, apperr.NotFound) {
			return getErr
		}
		return handle.Store.PutArtifactLineage(ctx, *lineage)
	}
	return nil
}

func artifactView(record ArtifactRecord) ArtifactView {
	a := record.Artifact
	return ArtifactView{
		ID: a.ID, ProjectID: a.ProjectID, SourceID: a.SourceID, MediaType: a.MediaType,
		ByteSize: a.ByteSize, Checksum: a.Checksum, ArtifactType: a.ArtifactType,
		SchemaID: a.SchemaID, TrustLevel: record.TrustLevel, EvidenceClass: record.EvidenceClass,
	}
}

func findArtifactRecord(st State, id ids.ArtifactID) (ArtifactRecord, bool) {
	for _, record := range st.Artifacts {
		if record.Artifact.ID == id {
			return record, true
		}
	}
	return ArtifactRecord{}, false
}

func findLineage(st State, id ids.ArtifactID) *artifacts.ArtifactLineage {
	for _, lineage := range st.Lineages {
		if lineage.OutputArtifactID == id {
			copy := lineage
			return &copy
		}
	}
	return nil
}
