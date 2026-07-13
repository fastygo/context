package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/fastygo/context/internal/httpserver"
)

func main() {
	addr := envOr("CONTEXT_SERVE_ADDR", ":8080")
	data := ""
	token := os.Getenv("CONTEXT_SERVE_TOKEN")
	evalOut := os.Getenv("CONTEXT_SERVE_EVAL_OUT")

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--addr":
			if i+1 >= len(args) {
				fatal("missing --addr value")
			}
			addr = args[i+1]
			i++
		case "--data":
			if i+1 >= len(args) {
				fatal("missing --data value")
			}
			data = args[i+1]
			i++
		case "--token":
			if i+1 >= len(args) {
				fatal("missing --token value")
			}
			token = args[i+1]
			i++
		case "--eval-out":
			if i+1 >= len(args) {
				fatal("missing --eval-out value")
			}
			evalOut = args[i+1]
			i++
		case "-h", "--help":
			usage()
			os.Exit(0)
		default:
			fatal("unknown flag %s", args[i])
		}
	}
	if data == "" {
		data = os.Getenv("CONTEXT_DATA_DIR")
	}
	if data == "" {
		fatal("--data or CONTEXT_DATA_DIR required")
	}

	srv, err := httpserver.New(httpserver.Config{
		DataDir: data,
		Token:   token,
		EvalOut: evalOut,
	})
	if err != nil {
		fatal("%v", err)
	}

	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	fmt.Fprintf(os.Stderr, "context-serve listening on %s (data=%s)\n", addr, data)
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fatal("%v", err)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func usage() {
	fmt.Fprintf(os.Stderr, `context-serve — thin HTTP+JSON API over Context CLI contracts (ADR-0024)

Usage:
  context-serve --data <dir> [--addr :8080] [--token <secret>] [--eval-out <path>]

Env:
  CONTEXT_DATA_DIR, CONTEXT_SERVE_ADDR, CONTEXT_SERVE_TOKEN, CONTEXT_SERVE_EVAL_OUT
  Same backend env as context-dev (CONTEXT_PG_DSN, CONTEXT_SPARSE_KIND, …).

Endpoints:
  GET  /health
  GET  /v1/status?project_id=
  POST /v1/search          {"project_id","query","mode?","focus_id?"}
  POST /v1/context-pack    {"project_id","query","focus_id?"}
  POST /v1/agent-run       {"project_id","query","focus_id?"}
  GET  /v1/trace?project_id=&run_id=
  PUT  /v1/focus           {"project_id","focus":{...}}
  GET  /v1/focus?project_id=&focus_id=
  GET  /v1/focuses?project_id=
  POST /v1/eval
  GET  /v1/eval/history?limit=
  GET  /v1/metrics?project_id=
  POST /v1/repair          {"project_id","mode?","target?"}
  POST /v1/ingest          {"project_id","path_key?"}  # relative to corpus only

Auth (optional): Authorization: Bearer <token> or X-Context-Token.
`)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
