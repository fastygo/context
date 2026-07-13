#!/usr/bin/env bash
# Local PostgreSQL/pgvector helpers (Chunk 09). Same targets as Makefile.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

COMPOSE=(docker compose)
if [[ -f .env ]]; then
  COMPOSE+=(--env-file .env)
else
  COMPOSE+=(--env-file .env.example)
fi

PG_USER="${PG_USER:-context}"
PG_DATABASE="${PG_DATABASE:-context}"
PG_SERVICE="${PG_SERVICE:-postgres}"
EMBED_DIM="${EMBED_DIM:-8}"

usage() {
  cat <<'EOF'
Usage: scripts/dev.sh <command>

  up       Start PostgreSQL/pgvector
  down     Stop containers (keep volume)
  reset    Stop and remove containers + volume
  logs     Follow postgres logs
  ps       Show compose status
  wait     Wait until postgres accepts connections
  health   Connection + pgvector + dimension smoke checks
EOF
}

cmd_wait() {
  echo "Waiting for ${PG_SERVICE}..."
  local i=0
  while (( i < 60 )); do
    if "${COMPOSE[@]}" exec -T "$PG_SERVICE" pg_isready -U "$PG_USER" -d "$PG_DATABASE" >/dev/null 2>&1; then
      echo "ready"
      return 0
    fi
    i=$((i + 1))
    sleep 1
  done
  echo "postgres did not become ready in time" >&2
  return 1
}

cmd_health() {
  cmd_wait
  "${COMPOSE[@]}" exec -T "$PG_SERVICE" psql -U "$PG_USER" -d "$PG_DATABASE" -v ON_ERROR_STOP=1 -c "SELECT 1 AS ok;"
  "${COMPOSE[@]}" exec -T "$PG_SERVICE" psql -U "$PG_USER" -d "$PG_DATABASE" -v ON_ERROR_STOP=1 -c "SELECT extname, extversion FROM pg_extension WHERE extname = 'vector';"
  "${COMPOSE[@]}" exec -T "$PG_SERVICE" psql -U "$PG_USER" -d "$PG_DATABASE" -v ON_ERROR_STOP=1 -c "CREATE EXTENSION IF NOT EXISTS vector;"
  "${COMPOSE[@]}" exec -T "$PG_SERVICE" psql -U "$PG_USER" -d "$PG_DATABASE" -v ON_ERROR_STOP=1 -c "DROP TABLE IF EXISTS context_vector_dim_smoke; CREATE TEMP TABLE context_vector_dim_smoke (embedding vector(${EMBED_DIM})); INSERT INTO context_vector_dim_smoke VALUES (array_fill(0::float4, ARRAY[${EMBED_DIM}])::vector); SELECT vector_dims(embedding) AS dims FROM context_vector_dim_smoke;"
  echo "dev-health passed (pgvector + dimension ${EMBED_DIM})"
}

case "${1:-}" in
  up) "${COMPOSE[@]}" up -d ;;
  down) "${COMPOSE[@]}" down ;;
  reset) "${COMPOSE[@]}" down -v ;;
  logs) "${COMPOSE[@]}" logs -f "$PG_SERVICE" ;;
  ps) "${COMPOSE[@]}" ps ;;
  wait) cmd_wait ;;
  health) cmd_health ;;
  *) usage; exit 1 ;;
esac
