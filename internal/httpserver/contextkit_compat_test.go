package httpserver_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/fastygo/context/internal/httpserver"
	"github.com/fastygo/context/pkg/contextkit"
)

func TestContextKitCompatSmoke(t *testing.T) {
	dataDir := setupWorkspace(t)
	srv, err := httpserver.New(httpserver.Config{DataDir: dataDir})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	cli := &contextkit.Client{BaseURL: ts.URL}
	ctx := context.Background()

	st, err := cli.Status(ctx, "proj_http")
	if err != nil || !st.OK || st.Chunks < 1 {
		t.Fatalf("status: %+v %v", st, err)
	}
	search, err := cli.Search(ctx, contextkit.SearchRequest{
		ProjectID: "proj_http", Query: "ZEBRA42", Mode: "exact",
	})
	if err != nil || len(search.Candidates) < 1 {
		t.Fatalf("search: %+v %v", search, err)
	}
	pack, err := cli.ContextPack(ctx, contextkit.PackRequest{
		ProjectID: "proj_http", Query: "ZEBRA42",
	})
	if err != nil || len(pack.ContextPack) == 0 {
		t.Fatalf("pack: %+v %v", pack, err)
	}
	agent, err := cli.AgentRun(ctx, contextkit.PackRequest{
		ProjectID: "proj_http", Query: "ZEBRA42",
	})
	if err != nil {
		t.Fatalf("agent: %v", err)
	}
	runID, err := agent.RunID()
	if err != nil || runID == "" {
		t.Fatalf("run id: %v %#v", err, agent)
	}
	trace, err := cli.Trace(ctx, "proj_http", runID)
	if err != nil || len(trace.Events) < 1 {
		t.Fatalf("trace: %+v %v", trace, err)
	}
}
