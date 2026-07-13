package localfs_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/artifacts"
	"github.com/fastygo/context/internal/artifacts/localfs"
	"github.com/fastygo/context/internal/ids"
)

func TestPutGetRoundTrip(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store, err := localfs.New(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	art, err := store.Put(ctx, "proj1", "art1", "text/plain", []byte("hello"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if art.ByteSize != 5 || art.Checksum == "" || art.StorageURI == "" {
		t.Fatalf("unexpected artifact: %#v", art)
	}
	got, body, err := store.Get(ctx, "proj1", "art1")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "hello" {
		t.Fatalf("body=%q", body)
	}
	if got.Checksum != art.Checksum {
		t.Fatalf("checksum mismatch meta=%s put=%s", got.Checksum, art.Checksum)
	}
}

func TestRejectsPathTraversalIDs(t *testing.T) {
	t.Parallel()
	store, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	cases := []struct {
		project  ids.ProjectID
		artifact ids.ArtifactID
	}{
		{project: "../escape", artifact: "a1"},
		{project: "p1", artifact: "../escape"},
		{project: "p1/../x", artifact: "a1"},
		{project: "p1", artifact: "a/b"},
		{project: `p1\x`, artifact: "a1"},
	}
	for _, tc := range cases {
		_, err := store.Put(ctx, tc.project, tc.artifact, "text/plain", []byte("x"), nil)
		if !apperr.Is(err, apperr.Permission) {
			t.Fatalf("project=%q artifact=%q err=%v", tc.project, tc.artifact, err)
		}
	}
}

func TestChecksumMismatchDetected(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store, err := localfs.New(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := store.Put(ctx, "proj1", "art1", "text/plain", []byte("hello"), nil); err != nil {
		t.Fatal(err)
	}
	dataPath := filepath.Join(root, "proj1", "art1", "data.bin")
	if err := os.WriteFile(dataPath, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err = store.Get(ctx, "proj1", "art1")
	if !apperr.Is(err, apperr.Conflict) {
		t.Fatalf("expected checksum conflict, got %v", err)
	}
}

func TestPutStructuredRoundTrip(t *testing.T) {
	t.Parallel()
	store, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	body := []byte(`{"nodes":[]}`)
	art, err := store.Put(ctx, "proj1", "spec1", "application/json", body, &artifacts.PutOptions{
		ArtifactType: artifacts.TypeStructured,
		SchemaID:     "uxspec.screen.v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if art.ArtifactType != artifacts.TypeStructured || art.SchemaID != "uxspec.screen.v1" {
		t.Fatalf("put meta: %#v", art)
	}
	got, raw, err := store.Get(ctx, "proj1", "spec1")
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != string(body) {
		t.Fatalf("body=%s", raw)
	}
	if got.SchemaID != "uxspec.screen.v1" || got.ArtifactType != artifacts.TypeStructured {
		t.Fatalf("get meta: %#v", got)
	}
}

func TestPutStructuredWithoutSchemaRejected(t *testing.T) {
	t.Parallel()
	store, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Put(context.Background(), "proj1", "bad", "application/json", []byte(`{}`), &artifacts.PutOptions{
		ArtifactType: artifacts.TypeStructured,
	})
	if !apperr.Is(err, apperr.Validation) {
		t.Fatalf("expected validation, got %v", err)
	}
}

func TestGetMissingNotFound(t *testing.T) {
	t.Parallel()
	store, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = store.Get(context.Background(), "proj1", "missing")
	if !apperr.Is(err, apperr.NotFound) {
		t.Fatalf("expected not_found, got %v", err)
	}
}
