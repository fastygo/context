package devcli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fastygo/context/internal/devcli"
)

func TestAgentRunLocalEchoCompleter(t *testing.T) {
	t.Setenv("CONTEXT_ENABLE_DENSE", "")
	t.Setenv("CONTEXT_SPARSE_KIND", "")
	t.Setenv("CONTEXT_COMPLETER_KIND", "localecho")
	t.Setenv("CONTEXT_METADATA_KIND", "")
	root := t.TempDir()
	corpus := filepath.Join(root, "corpus")
	data := filepath.Join(root, "data")
	if err := os.MkdirAll(corpus, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corpus, "a.md"), []byte("# Hi\n\nZEBRA42 cite me\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.InitProject(data, corpus, "proj_echo", "Echo"); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.Ingest(data, "proj_echo", ""); err != nil {
		t.Fatal(err)
	}
	res, err := devcli.AgentRun(data, "proj_echo", "ZEBRA42", "")
	if err != nil {
		t.Fatal(err)
	}
	if res.CompleterKind != "localecho" || res.ModelProvider != "localecho" {
		t.Fatalf("%#v", res)
	}
	if !strings.Contains(res.ModelText, "cite[") {
		t.Fatalf("text=%q", res.ModelText)
	}
	if !res.VerifyOK {
		t.Fatalf("verify failed: %#v", res)
	}
}
