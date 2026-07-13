package devcli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fastygo/context/internal/devcli"
)

func TestCLIWorkflowSmoke(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	data := t.TempDir()
	doc := filepath.Join(root, "note.md")
	if err := os.WriteFile(doc, []byte("# Hello\n\nContextPack evidence lives here.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	st, err := devcli.InitProject(data, root, "demo", "Demo")
	if err != nil {
		t.Fatal(err)
	}
	if st.Project.ID != "demo" {
		t.Fatalf("project=%s", st.Project.ID)
	}

	st, err = devcli.Ingest(data, "demo", root)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Chunks) == 0 {
		t.Fatal("expected chunks")
	}

	search, err := devcli.Search(data, "demo", "ContextPack", "hybrid", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(search.Candidates) == 0 {
		t.Fatal("expected search hits")
	}

	pack, err := devcli.BuildPack(data, "demo", "ContextPack", "")
	if err != nil {
		t.Fatal(err)
	}
	if pack.Pack.Checksum == "" {
		t.Fatal("expected pack checksum")
	}

	agent, err := devcli.AgentRun(data, "demo", "ContextPack", "")
	if err != nil {
		t.Fatal(err)
	}
	if agent.Run.ID == "" || agent.ModelText == "" {
		t.Fatalf("agent=%#v", agent)
	}

	tr, err := devcli.Trace(data, "demo", string(agent.Run.ID))
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.Events) == 0 {
		t.Fatal("expected trace events")
	}
}
