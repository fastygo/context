package devcli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fastygo/context/internal/devcli"
)

func TestAgentAndInspectRedactSecrets(t *testing.T) {
	t.Setenv("CONTEXT_REDACT", "1")
	t.Setenv("CONTEXT_COMPLETER_KIND", "localecho")
	t.Setenv("CONTEXT_ENABLE_DENSE", "")
	t.Setenv("CONTEXT_FAIL_VECTOR", "")
	t.Setenv("CONTEXT_FAIL_METADATA", "")

	root := t.TempDir()
	corpus := filepath.Join(root, "corpus")
	data := filepath.Join(root, "data")
	if err := os.MkdirAll(corpus, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "# Note\n\napi_key=supersecretvalue99 and token ZEBRA42\n"
	if err := os.WriteFile(filepath.Join(corpus, "secret.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.InitProject(data, corpus, "proj_redact", "Redact"); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.Ingest(data, "proj_redact", ""); err != nil {
		t.Fatal(err)
	}

	agent, err := devcli.AgentRun(data, "proj_redact", "ZEBRA42", "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(agent.ModelText, "supersecretvalue99") {
		t.Fatalf("model_text leaked secret: %q", agent.ModelText)
	}
	if !agent.Redacted {
		t.Fatalf("want agent redacted flag: %#v text=%q", agent, agent.ModelText)
	}

	ins, err := devcli.Inspect(data, "proj_redact", "ZEBRA42", "", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ins.Selected {
		if strings.Contains(e.SurfacePreview, "supersecretvalue99") {
			t.Fatalf("inspect preview leaked: %#v", e)
		}
	}
	if !ins.Redacted {
		t.Fatalf("want inspect redacted: %#v", ins.Selected)
	}
}
