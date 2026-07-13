# Local development helpers for the PostgreSQL/pgvector PoC stack (Chunk 09).
# Unit tests do not require these targets: `go test ./...` stays offline.

COMPOSE ?= docker compose
ENV_FILE ?= .env
ifneq (,$(wildcard $(ENV_FILE)))
  COMPOSE_ENV := --env-file $(ENV_FILE)
else
  COMPOSE_ENV := --env-file .env.example
endif

PG_USER ?= context
PG_DATABASE ?= context
PG_SERVICE ?= postgres
EMBED_DIM ?= 8

.PHONY: help dev-up dev-down dev-reset dev-logs dev-ps dev-wait dev-health go-test

help:
	@echo "Targets:"
	@echo "  make dev-up      Start PostgreSQL/pgvector"
	@echo "  make dev-down    Stop containers (keep volume)"
	@echo "  make dev-reset   Stop and remove containers + volume"
	@echo "  make dev-logs    Follow postgres logs"
	@echo "  make dev-ps      Show compose status"
	@echo "  make dev-wait    Wait until postgres is healthy"
	@echo "  make dev-health  Run connection + pgvector smoke checks"
	@echo "  make go-test     Run unit tests (no Docker required)"

dev-up:
	$(COMPOSE) $(COMPOSE_ENV) up -d

dev-down:
	$(COMPOSE) $(COMPOSE_ENV) down

dev-reset:
	$(COMPOSE) $(COMPOSE_ENV) down -v

dev-logs:
	$(COMPOSE) $(COMPOSE_ENV) logs -f $(PG_SERVICE)

dev-ps:
	$(COMPOSE) $(COMPOSE_ENV) ps

dev-wait:
	@echo "Waiting for $(PG_SERVICE) health..."
	@i=0; \
	while [ $$i -lt 60 ]; do \
	  status=$$($(COMPOSE) $(COMPOSE_ENV) ps --format json $(PG_SERVICE) 2>/dev/null | sed -n 's/.*"Health":"\([^"]*\)".*/\1/p' | head -n1); \
	  if [ "$$status" = "healthy" ]; then echo "healthy"; exit 0; fi; \
	  ready=$$($(COMPOSE) $(COMPOSE_ENV) exec -T $(PG_SERVICE) pg_isready -U $(PG_USER) -d $(PG_DATABASE) >/dev/null 2>&1 && echo ok || echo no); \
	  if [ "$$ready" = "ok" ]; then echo "ready"; exit 0; fi; \
	  i=$$((i+1)); sleep 1; \
	done; \
	echo "postgres did not become ready in time" >&2; exit 1

dev-health: dev-wait
	@$(COMPOSE) $(COMPOSE_ENV) exec -T $(PG_SERVICE) psql -U $(PG_USER) -d $(PG_DATABASE) -v ON_ERROR_STOP=1 -c "SELECT 1 AS ok;"
	@$(COMPOSE) $(COMPOSE_ENV) exec -T $(PG_SERVICE) psql -U $(PG_USER) -d $(PG_DATABASE) -v ON_ERROR_STOP=1 -c "SELECT extname, extversion FROM pg_extension WHERE extname = 'vector';"
	@$(COMPOSE) $(COMPOSE_ENV) exec -T $(PG_SERVICE) psql -U $(PG_USER) -d $(PG_DATABASE) -v ON_ERROR_STOP=1 -c "CREATE EXTENSION IF NOT EXISTS vector;"
	@$(COMPOSE) $(COMPOSE_ENV) exec -T $(PG_SERVICE) psql -U $(PG_USER) -d $(PG_DATABASE) -v ON_ERROR_STOP=1 -c "DROP TABLE IF EXISTS context_vector_dim_smoke; CREATE TEMP TABLE context_vector_dim_smoke (embedding vector($(EMBED_DIM))); INSERT INTO context_vector_dim_smoke VALUES (array_fill(0::float4, ARRAY[$(EMBED_DIM)])::vector); SELECT vector_dims(embedding) AS dims FROM context_vector_dim_smoke;"
	@echo "dev-health passed (pgvector + dimension $(EMBED_DIM))"

go-test:
	go test ./...
