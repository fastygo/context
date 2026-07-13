package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fastygo/context/internal/devcli"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/retrieval"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]
	var err error
	switch cmd {
	case "init-project":
		err = cmdInit(args)
	case "ingest":
		err = cmdIngest(args)
	case "search":
		err = cmdSearch(args)
	case "context-pack":
		err = cmdPack(args)
	case "agent-run":
		err = cmdAgent(args)
	case "trace":
		err = cmdTrace(args)
	case "focus-put":
		err = cmdFocusPut(args)
	case "focus-get":
		err = cmdFocusGet(args)
	case "focus-list":
		err = cmdFocusList(args)
	case "eval":
		err = cmdEval(args)
	case "metrics":
		err = cmdMetrics(args)
	case "eval-history":
		err = cmdEvalHistory(args)
	case "repair":
		err = cmdRepair(args)
	case "inspect":
		err = cmdInspect(args)
	case "meta-check":
		err = cmdMetaCheck(args)
	case "proof-run":
		err = cmdProofRun(args)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", cmd)
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `context-dev — local Context Runtime developer CLI

Usage:
  context-dev init-project --root <dir> --data <dir> [--project <id>]
  context-dev ingest --data <dir> --project <id> [--path <dir-or-file>]
  context-dev search --data <dir> --project <id> --query <text> [--mode exact|sparse|hybrid|dense|hybrid-dense] [--focus <id>]
  context-dev context-pack --data <dir> --project <id> --query <text> [--focus <id>]
  context-dev agent-run --data <dir> --project <id> --query <text> [--focus <id>]
  context-dev focus-put --data <dir> --project <id> --json <file-or-inline> [--id <focus_id>]
  context-dev focus-get --data <dir> --project <id> --focus <id>
  context-dev focus-list --data <dir> --project <id>
  context-dev eval [--out <.project/proof/eval/report.json>] [--history <jsonl>] [--data <dir>]
  context-dev metrics --data <dir>
  context-dev eval-history [--data <dir>] [--history <jsonl>] [--limit N]
  context-dev repair --data <dir> --project <id> [--mode rebuild|retry-failed] [--target all|dense|sparse]
  context-dev inspect --data <dir> --project <id> (--query <text> | --pack <id>) [--focus <id>]
  context-dev trace --data <dir> --project <id> --run <id>
  context-dev meta-check [--backend postgres]
  context-dev proof-run [--root <repo>] [--out <.project/proof>]

Ingest skips paths via defaults + optional .contextignore at corpus root.
eval runs offline golden retrieval suite (exact/sparse/dense/hybrid + multilingual/lexicon/pack).
With --data, eval also appends a summary to <data>/ops/eval_history.jsonl (or --history).
metrics / eval-history expose workspace counters and append-only eval regression history.
repair rebuilds index payloads for the active ready snapshot, or retries last_failed under a new snapshot_id (ADR-0021).
inspect explains search/pack decisions for Lab (budget, selected/rejected, scores) without host paths.
Modes dense and hybrid-dense require PostgreSQL/pgvector (see .project/local-server.md).
Set CONTEXT_ENABLE_DENSE=1 to upsert dense vectors on ingest and include dense in hybrid search.
Set CONTEXT_DENSE_REBUILD=1 to force search-time vector rebuild (default: prefer ingest commit).
Set CONTEXT_EMBEDDER_KIND=local_hash for offline L2/SHA embedder (dim 32, local-hash-v1).
Set CONTEXT_SPARSE_KIND=postgres_fts for live Postgres FTS sparse/hybrid search.
Focus profiles persist to state.json and MetadataStore (postgres when configured).
meta-check verifies durable metadata (schema_id, lineage, temporal, documents).
proof-run executes Chunk 12 end-to-end proof and writes JSON under --out.
Outputs stable JSON on stdout for Lab/fixture consumption.
`)
}

func cmdMetaCheck(args []string) error {
	f := flagMap(args)
	res, err := devcli.MetaCheck(f["backend"])
	if err != nil {
		return err
	}
	return devcli.PrintJSON(res)
}

func cmdProofRun(args []string) error {
	f := flagMap(args)
	root := f["root"]
	if root == "" {
		root = "."
	}
	out := f["out"]
	res, err := devcli.RunProof(root, out)
	if err != nil {
		_ = devcli.PrintJSON(res)
		return err
	}
	return devcli.PrintJSON(res)
}

func flagMap(args []string) map[string]string {
	out := map[string]string{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if len(a) < 3 || a[:2] != "--" {
			continue
		}
		key := a[2:]
		if i+1 >= len(args) {
			out[key] = "true"
			continue
		}
		out[key] = args[i+1]
		i++
	}
	return out
}

func require(flags map[string]string, keys ...string) error {
	for _, k := range keys {
		if flags[k] == "" {
			return fmt.Errorf("missing --%s", k)
		}
	}
	return nil
}

func cmdInit(args []string) error {
	f := flagMap(args)
	if err := require(f, "root", "data"); err != nil {
		return err
	}
	st, err := devcli.InitProject(f["data"], f["root"], f["project"], f["name"])
	if err != nil {
		return err
	}
	return devcli.PrintJSON(map[string]any{
		"ok":         true,
		"project_id": st.Project.ID,
		"data":       f["data"],
		"root":       st.CorpusRoot,
	})
}

func cmdIngest(args []string) error {
	f := flagMap(args)
	if err := require(f, "data", "project"); err != nil {
		return err
	}
	st, err := devcli.Ingest(f["data"], f["project"], f["path"])
	if err != nil {
		return err
	}
	return devcli.PrintJSON(map[string]any{
		"ok":          true,
		"project_id":  st.Project.ID,
		"snapshot_id": st.Snapshot.ID,
		"chunks":      len(st.Chunks),
		"status":      st.Snapshot.Status,
		"source_root": st.Snapshot.SourceMerkleRoot,
		"chunk_set":   st.Snapshot.ChunkSetHash,
	})
}

func cmdSearch(args []string) error {
	f := flagMap(args)
	if err := require(f, "data", "project", "query"); err != nil {
		return err
	}
	res, err := devcli.Search(f["data"], f["project"], f["query"], f["mode"], f["focus"])
	if err != nil {
		return err
	}
	return devcli.PrintJSON(res)
}

func cmdPack(args []string) error {
	f := flagMap(args)
	if err := require(f, "data", "project", "query"); err != nil {
		return err
	}
	res, err := devcli.BuildPack(f["data"], f["project"], f["query"], f["focus"])
	if err != nil {
		return err
	}
	return devcli.PrintJSON(res)
}

func cmdAgent(args []string) error {
	f := flagMap(args)
	if err := require(f, "data", "project", "query"); err != nil {
		return err
	}
	res, err := devcli.AgentRun(f["data"], f["project"], f["query"], f["focus"])
	if err != nil {
		return err
	}
	return devcli.PrintJSON(res)
}

func cmdFocusPut(args []string) error {
	f := flagMap(args)
	if err := require(f, "data", "project", "json"); err != nil {
		return err
	}
	focus, err := devcli.ParseFocusJSON(f["json"])
	if err != nil {
		return err
	}
	if f["id"] != "" {
		focus.ID = ids.FocusID(f["id"])
	}
	if focus.Objective == "" && f["objective"] != "" {
		focus.Objective = f["objective"]
	}
	if focus.RequiredTrustLevel == "" {
		focus.RequiredTrustLevel = foundation.TrustProject
	}
	if focus.ContextBudget.MaxItems == 0 {
		focus.ContextBudget = retrieval.Budget{MaxItems: 8, MaxChars: 4000}
	}
	res, err := devcli.PutFocus(f["data"], f["project"], focus)
	if err != nil {
		return err
	}
	return devcli.PrintJSON(res)
}

func cmdFocusGet(args []string) error {
	f := flagMap(args)
	if err := require(f, "data", "project", "focus"); err != nil {
		return err
	}
	focus, kind, err := devcli.GetFocus(f["data"], f["project"], f["focus"])
	if err != nil {
		return err
	}
	return devcli.PrintJSON(map[string]any{"focus": focus, "meta_kind": kind})
}

func cmdFocusList(args []string) error {
	f := flagMap(args)
	if err := require(f, "data", "project"); err != nil {
		return err
	}
	res, err := devcli.ListFocus(f["data"], f["project"])
	if err != nil {
		return err
	}
	return devcli.PrintJSON(res)
}

func cmdEval(args []string) error {
	f := flagMap(args)
	out := f["out"]
	if out == "" {
		out = ".project/proof/eval/report.json"
	}
	history := f["history"]
	if history == "" && f["data"] != "" {
		history = filepath.Join(f["data"], "ops", "eval_history.jsonl")
	}
	res, err := devcli.RunEval(out, history)
	if err != nil {
		return err
	}
	if err := devcli.PrintJSON(res); err != nil {
		return err
	}
	if !res.Report.OK {
		return fmt.Errorf("eval golden suite failed")
	}
	return nil
}

func cmdMetrics(args []string) error {
	f := flagMap(args)
	if err := require(f, "data"); err != nil {
		return err
	}
	res, err := devcli.Metrics(f["data"])
	if err != nil {
		return err
	}
	return devcli.PrintJSON(res)
}

func cmdEvalHistory(args []string) error {
	f := flagMap(args)
	history := f["history"]
	if history == "" && f["data"] != "" {
		history = filepath.Join(f["data"], "ops", "eval_history.jsonl")
	}
	if history == "" {
		return fmt.Errorf("missing --history or --data")
	}
	limit := devcli.ParseLimit(f["limit"], 20)
	res, err := devcli.EvalHistory(history, limit)
	if err != nil {
		return err
	}
	return devcli.PrintJSON(res)
}

func cmdRepair(args []string) error {
	f := flagMap(args)
	if err := require(f, "data", "project"); err != nil {
		return err
	}
	res, err := devcli.Repair(f["data"], f["project"], f["mode"], f["target"])
	if err != nil {
		return err
	}
	return devcli.PrintJSON(res)
}

func cmdInspect(args []string) error {
	f := flagMap(args)
	if err := require(f, "data", "project"); err != nil {
		return err
	}
	if f["query"] == "" && f["pack"] == "" {
		return fmt.Errorf("missing --query or --pack")
	}
	res, err := devcli.Inspect(f["data"], f["project"], f["query"], f["focus"], f["pack"])
	if err != nil {
		return err
	}
	return devcli.PrintJSON(res)
}

func cmdTrace(args []string) error {
	f := flagMap(args)
	if err := require(f, "data", "project", "run"); err != nil {
		return err
	}
	res, err := devcli.Trace(f["data"], f["project"], f["run"])
	if err != nil {
		return err
	}
	return devcli.PrintJSON(res)
}
