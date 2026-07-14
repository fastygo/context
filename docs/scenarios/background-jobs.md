# Scenario: background jobs and schedules

## Jobs (in-process AgentRun)

Mid-flight jobs die if the process exits. `owner` is required. Prefer
`context-serve` for jobs that must outlive a single CLI invocation.

```bash
go run ./cmd/context-dev job-start --data "$DATA" --project demo \
  --query 'ZEBRA42' --owner lab
go run ./cmd/context-dev job-status --data "$DATA" --project demo --job job_...
go run ./cmd/context-dev job-list --data "$DATA" --project demo
go run ./cmd/context-dev job-cancel --data "$DATA" --project demo --job job_...
```

Records: `<data>/ops/jobs/*.json` (no host paths in API JSON).

## Schedules (ADR-0031 / stabilization C8)

Schedule **definitions** survive process restart (`ops/schedules/*.json`).
Execution still uses the in-process job registry. After restart, tick to
enqueue overdue work as new jobs (`context-serve` ticks once on start).

```bash
# Fire once as soon as tick runs
go run ./cmd/context-dev schedule-put --data "$DATA" --project demo \
  --owner lab --query 'ZEBRA42' --kind once_at --at 2020-01-01T00:00:00Z

go run ./cmd/context-dev schedule-tick --data "$DATA"
go run ./cmd/context-dev job-list --data "$DATA" --project demo

# Interval (seconds) or event
go run ./cmd/context-dev schedule-put --data "$DATA" --project demo \
  --owner lab --query 'ZEBRA42' --kind interval --interval 3600
go run ./cmd/context-dev schedule-put --data "$DATA" --project demo \
  --owner lab --query 'ZEBRA42' --kind event --event ingest.completed
go run ./cmd/context-dev schedule-fire --data "$DATA" --project demo \
  --event ingest.completed
```

## HTTP

```bash
curl -s -X POST http://127.0.0.1:8080/v1/jobs \
  -H 'Content-Type: application/json' \
  -d '{"project_id":"demo","query":"ZEBRA42","owner":"lab"}'

curl -s -X PUT http://127.0.0.1:8080/v1/schedules \
  -H 'Content-Type: application/json' \
  -d '{"project_id":"demo","owner":"lab","query":"ZEBRA42","kind":"once_at","enabled":true,"next_run_at":"2020-01-01T00:00:00Z"}'

curl -s -X POST http://127.0.0.1:8080/v1/schedules/tick
```

## Job status values

`pending` → `running` → `completed` | `failed` | `cancelled`
