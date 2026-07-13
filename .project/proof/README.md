# Chunk 12 Proof Artifacts

Neutral JSON produced by `context-dev proof-run` for Lab replay without live
services. Re-run:

```bash
./scripts/dev.sh up
export CONTEXT_PG_DSN='postgres://context:context@127.0.0.1:5432/context?sslmode=disable'
./scripts/proof-e2e.sh
# or: go run ./cmd/context-dev proof-run --root . --out .project/proof
```

| File | Contents |
| --- | --- |
| `corpus/` | Ingested proof sources (README, roadmap excerpt, multilingual, lexicon, events) |
| `01-ingest.json` | Snapshot + chunk counts |
| `02-search-*.json` | exact / sparse / hybrid / dense / hybrid-dense results |
| `03-multilingual.json` | Language filter, token span, query-expansion trace |
| `04-lexicon.json` | Sense/concept/attestation/register/region/time/authority |
| `05-context-pack.json` | ContextPack for roadmap query |
| `06-agent-run.json` | Fake model/tool run + verifier |
| `07-trace.json` | Replayable AgentRun events |
| `08-events-lineage-temporal.json` | Event-window temporal filter + derived lineage |
| `SUMMARY.json` / `SUMMARY.md` | Hypothesis status, gaps, next decisions |
| `workspace/` | Ephemeral CLI `--data` directory (regenerated each run) |
