package factory_test

import (
	"context"
	"testing"

	"github.com/fastygo/context/internal/config"
	"github.com/fastygo/context/internal/models/factory"
	"github.com/fastygo/context/internal/models/fake"
	"github.com/fastygo/context/internal/models/localhash"
)

func TestOpenEmbedderFakeDefault(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultStorageConfig()
	emb, ver, err := factory.OpenEmbedder(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if ver != fake.EmbeddingVersion {
		t.Fatalf("ver=%q", ver)
	}
	vecs, gotVer, err := emb.Embed(context.Background(), []string{"x"})
	if err != nil || gotVer != fake.EmbeddingVersion || len(vecs[0]) != 8 {
		t.Fatalf("embed=%v ver=%q err=%v", len(vecs[0]), gotVer, err)
	}
}

func TestOpenEmbedderLocalHash(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultStorageConfig()
	cfg.Embedder.Kind = config.EmbedderKindLocalHash
	cfg.Vector.EmbeddingVersion = localhash.Version
	cfg.Vector.Dimension = localhash.DefaultDim
	emb, ver, err := factory.OpenEmbedder(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if ver != localhash.Version {
		t.Fatalf("ver=%q", ver)
	}
	vecs, gotVer, err := emb.Embed(context.Background(), []string{"ContextPack"})
	if err != nil || gotVer != localhash.Version || len(vecs[0]) != 32 {
		t.Fatalf("dim=%d ver=%q err=%v", len(vecs[0]), gotVer, err)
	}
}

func TestOpenCompleterLocalEcho(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultStorageConfig()
	cfg.Completer.Kind = config.CompleterKindLocalEcho
	comp, kind, err := factory.OpenCompleter(cfg, factory.CompleterOptions{})
	if err != nil || kind != "localecho" {
		t.Fatalf("kind=%q err=%v", kind, err)
	}
	_ = comp
}

func TestOpenCompleterHTTPRequiresURL(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultStorageConfig()
	cfg.Completer.Kind = config.CompleterKindHTTP
	_, _, err := factory.OpenCompleter(cfg, factory.CompleterOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOpenEmbedderRejectsUnknownKind(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultStorageConfig()
	cfg.Embedder.Kind = "openai"
	_, _, err := factory.OpenEmbedder(cfg)
	if err == nil {
		t.Fatal("expected error")
	}
}
