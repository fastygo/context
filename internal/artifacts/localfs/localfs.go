// Package localfs implements a project-scoped filesystem ArtifactStore.
package localfs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/artifacts"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
)

const (
	dataFile = "data.bin"
	metaFile = "meta.txt"
)

// Store persists artifact bytes under a configured root directory.
type Store struct {
	root string
}

// New returns an ArtifactStore rooted at root. The directory is created if needed.
func New(root string) (*Store, error) {
	if root == "" {
		return nil, apperr.New(apperr.Validation, "artifact store root: empty")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, apperr.Wrap(apperr.Validation, "artifact store root", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, apperr.Wrap(apperr.Validation, "create artifact root", err)
	}
	return &Store{root: abs}, nil
}

func (s *Store) Put(ctx context.Context, projectID ids.ProjectID, artifactID ids.ArtifactID, mediaType string, body []byte, opts *artifacts.PutOptions) (artifacts.Artifact, error) {
	if err := ctx.Err(); err != nil {
		return artifacts.Artifact{}, err
	}
	if err := projectID.Validate(); err != nil {
		return artifacts.Artifact{}, apperr.Wrap(apperr.Validation, "project_id", err)
	}
	if err := artifactID.Validate(); err != nil {
		return artifacts.Artifact{}, apperr.Wrap(apperr.Validation, "artifact_id", err)
	}
	if mediaType == "" {
		return artifacts.Artifact{}, apperr.New(apperr.Validation, "media_type: empty")
	}
	dir, err := s.artifactDir(projectID, artifactID)
	if err != nil {
		return artifacts.Artifact{}, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return artifacts.Artifact{}, apperr.Wrap(apperr.Validation, "create artifact dir", err)
	}
	sum := sha256.Sum256(body)
	checksum := foundation.ChecksumHex(hex.EncodeToString(sum[:]))
	dataPath := filepath.Join(dir, dataFile)
	if err := os.WriteFile(dataPath, body, 0o644); err != nil {
		return artifacts.Artifact{}, apperr.Wrap(apperr.Validation, "write artifact data", err)
	}
	art := artifacts.ApplyPutOptions(artifacts.Artifact{
		ID:         artifactID,
		ProjectID:  projectID,
		MediaType:  mediaType,
		ByteSize:   int64(len(body)),
		Checksum:   checksum,
		StorageURI: storageURI(projectID, artifactID),
	}, opts)
	if err := art.Validate(); err != nil {
		return artifacts.Artifact{}, apperr.Wrap(apperr.Validation, "artifact", err)
	}
	if err := writeMeta(filepath.Join(dir, metaFile), art); err != nil {
		return artifacts.Artifact{}, err
	}
	return art, nil
}

func (s *Store) Get(ctx context.Context, projectID ids.ProjectID, artifactID ids.ArtifactID) (artifacts.Artifact, []byte, error) {
	if err := ctx.Err(); err != nil {
		return artifacts.Artifact{}, nil, err
	}
	dir, err := s.artifactDir(projectID, artifactID)
	if err != nil {
		return artifacts.Artifact{}, nil, err
	}
	art, err := readMeta(filepath.Join(dir, metaFile))
	if err != nil {
		return artifacts.Artifact{}, nil, err
	}
	body, err := os.ReadFile(filepath.Join(dir, dataFile))
	if err != nil {
		if os.IsNotExist(err) {
			return artifacts.Artifact{}, nil, apperr.New(apperr.NotFound, "artifact data missing")
		}
		return artifacts.Artifact{}, nil, apperr.Wrap(apperr.Validation, "read artifact data", err)
	}
	sum := sha256.Sum256(body)
	got := hex.EncodeToString(sum[:])
	if got != string(art.Checksum) {
		return artifacts.Artifact{}, nil, apperr.New(apperr.Conflict, "artifact checksum mismatch")
	}
	if int64(len(body)) != art.ByteSize {
		return artifacts.Artifact{}, nil, apperr.New(apperr.Conflict, "artifact size mismatch")
	}
	return art, body, nil
}

func (s *Store) Delete(ctx context.Context, projectID ids.ProjectID, artifactID ids.ArtifactID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	dir, err := s.artifactDir(projectID, artifactID)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return apperr.Wrap(apperr.Validation, "delete artifact", err)
	}
	return nil
}

func (s *Store) artifactDir(projectID ids.ProjectID, artifactID ids.ArtifactID) (string, error) {
	proj, err := safeSegment("project_id", string(projectID))
	if err != nil {
		return "", err
	}
	art, err := safeSegment("artifact_id", string(artifactID))
	if err != nil {
		return "", err
	}
	dir := filepath.Join(s.root, proj, art)
	rel, err := filepath.Rel(s.root, dir)
	if err != nil {
		return "", apperr.Wrap(apperr.Permission, "resolve artifact path", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", apperr.New(apperr.Permission, "artifact path escapes store root")
	}
	return dir, nil
}

func safeSegment(name, value string) (string, error) {
	if value == "" {
		return "", apperr.New(apperr.Validation, name+": empty")
	}
	if strings.Contains(value, "/") || strings.Contains(value, "\\") || strings.Contains(value, "..") {
		return "", apperr.New(apperr.Permission, name+": path traversal rejected")
	}
	if value == "." || value == ".." {
		return "", apperr.New(apperr.Permission, name+": path traversal rejected")
	}
	cleaned := filepath.Base(value)
	if cleaned != value {
		return "", apperr.New(apperr.Permission, name+": path traversal rejected")
	}
	return value, nil
}

func storageURI(projectID ids.ProjectID, artifactID ids.ArtifactID) string {
	return fmt.Sprintf("localfs://%s/%s", projectID, artifactID)
}

func writeMeta(path string, art artifacts.Artifact) error {
	var b strings.Builder
	fmt.Fprintf(&b, "id=%s\nproject_id=%s\nmedia_type=%s\nbyte_size=%d\nchecksum=%s\nstorage_uri=%s\nartifact_type=%s\n",
		art.ID, art.ProjectID, art.MediaType, art.ByteSize, art.Checksum, art.StorageURI, artifacts.NormalizeType(art.ArtifactType))
	if art.SchemaID != "" {
		fmt.Fprintf(&b, "schema_id=%s\n", art.SchemaID)
	}
	if art.SourceID != "" {
		fmt.Fprintf(&b, "source_id=%s\n", art.SourceID)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return apperr.Wrap(apperr.Validation, "write artifact meta", err)
	}
	return nil
}

func readMeta(path string) (artifacts.Artifact, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return artifacts.Artifact{}, apperr.New(apperr.NotFound, "artifact not found")
		}
		return artifacts.Artifact{}, apperr.Wrap(apperr.Validation, "read artifact meta", err)
	}
	var art artifacts.Artifact
	for _, line := range strings.Split(string(raw), "\n") {
		if line == "" {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch key {
		case "id":
			art.ID = ids.ArtifactID(val)
		case "project_id":
			art.ProjectID = ids.ProjectID(val)
		case "media_type":
			art.MediaType = val
		case "byte_size":
			var n int64
			if _, err := fmt.Sscanf(val, "%d", &n); err != nil {
				return artifacts.Artifact{}, apperr.New(apperr.Validation, "artifact meta byte_size invalid")
			}
			art.ByteSize = n
		case "checksum":
			art.Checksum = foundation.ChecksumHex(val)
		case "storage_uri":
			art.StorageURI = val
		case "artifact_type":
			art.ArtifactType = val
		case "schema_id":
			art.SchemaID = val
		case "source_id":
			art.SourceID = ids.SourceID(val)
		}
	}
	art.ArtifactType = artifacts.NormalizeType(art.ArtifactType)
	if err := art.Validate(); err != nil {
		return artifacts.Artifact{}, apperr.Wrap(apperr.Validation, "artifact meta", err)
	}
	return art, nil
}

// Ensure Store implements artifacts.ArtifactStore.
var _ artifacts.ArtifactStore = (*Store)(nil)
