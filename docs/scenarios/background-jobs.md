# Scenario: background jobs

In-process background **AgentRun** only (no cron/queue). Jobs die if the
process exits. `owner` is required.

## CLI

```bash
go run ./cmd/context-dev job-start --data "$DATA" --project demo \
  --query 'ZEBRA42' --owner lab
go run ./cmd/context-dev job-status --data "$DATA" --project demo --job job_...
go run ./cmd/context-dev job-list --data "$DATA" --project demo
go run ./cmd/context-dev job-cancel --data "$DATA" --project demo --job job_...
```

Records: `<data>/ops/jobs/*.json` (no host paths in API JSON).

## HTTP

```bash
# 202 Accepted
curl -s -X POST http://127.0.0.1:8080/v1/jobs \
  -H 'Content-Type: application/json' \
  -d '{"project_id":"demo","query":"ZEBRA42","owner":"lab"}'

curl -s "http://127.0.0.1:8080/v1/jobs?project_id=demo"
curl -s "http://127.0.0.1:8080/v1/jobs/job_...?project_id=demo"
curl -s -X POST "http://127.0.0.1:8080/v1/jobs/job_.../cancel?project_id=demo"
```

## Status values

`pending` → `running` → `completed` | `failed` | `cancelled`

Prefer `context-serve` for jobs that must outlive a single CLI invocation.
