# Scenario: Lab / BFF consumer

Lab may call Core only through HTTP or `pkg/contextkit`. See [lab-gate.md](../lab-gate.md).

## Rules

1. Always send `project_id` (Lab maps UI identity → allowed projects).
2. Never import `github.com/fastygo/context/internal/...`.
3. Never expect absolute host paths in JSON (`path_key` only).
4. Treat packs/traces/metrics as source of truth; Completer text is secondary.
5. Brand/product names stay in Lab config.

## Minimal Go client

```go
package main

import (
	"context"
	"fmt"
	"github.com/fastygo/context/pkg/contextkit"
)

func main() {
	cli := &contextkit.Client{BaseURL: "http://127.0.0.1:8080"}
	h, err := cli.Health(context.Background())
	if err != nil {
		panic(err)
	}
	fmt.Println(h.APIVersion, cli.LastAPIVersion)

	res, err := cli.Search(context.Background(), contextkit.SearchRequest{
		ProjectID: "demo", Query: "ZEBRA42", Mode: "hybrid",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(len(res.Candidates))
}
```

## Smoke path

`go test ./internal/httpserver/ -run TestLabGateSmoke -count=1` covers the
gate path offline (health → … → job).
