# Scenario: agent-run + trace

Foreground AgentRun: build pack → Completer → optional tool → verify → persist
run + trace events.

## CLI

```bash
# Offline citation Completer (recommended for demos)
CONTEXT_COMPLETER_KIND=localecho \
  go run ./cmd/context-dev agent-run --data "$DATA" --project demo --query 'ZEBRA42'

go run ./cmd/context-dev trace --data "$DATA" --project demo --run run_...
```

Completer kinds: `fake` (default) | `localecho` | `http`  
(`CONTEXT_COMPLETER_HTTP_URL` for http).

Response includes `model_text`, `redacted`, `verify_ok`, `completer_kind`.

## HTTP

```bash
curl -s -X POST http://127.0.0.1:8080/v1/agent-run \
  -H 'Content-Type: application/json' \
  -d '{"project_id":"demo","query":"ZEBRA42"}'

curl -s "http://127.0.0.1:8080/v1/trace?project_id=demo&run_id=run_..."
```

## Policy / quotas

- Soft quotas may deny pack/run when at hard limit (`CONTEXT_QUOTA_MAX_*`).
- Tool permissions stay outside the model (`policy` package).

## Related

- Async variant: [background-jobs.md](background-jobs.md)
- API catalog: [api/v1.md](../api/v1.md)
