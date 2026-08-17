.PHONY: build run run-watch worker-run test lint migrate-up migrate-down compose-up compose-infra compose-down docker-build tidy docs-preview docs-check web-install web-dev web-dev-2 web-build web-typecheck

GO ?= go
COMPOSE ?= docker compose -f deploy/docker-compose.yml
BIN_DIR ?= bin
DOCS_CLI ?= npx --yes @docs.page/cli
PNPM ?= pnpm
WEB_DIR ?= web
AIR ?= $(GO) run github.com/air-verse/air@v1.62.0
ENV_FILE ?= deploy/.env

# Load deploy/.env for local API/worker/migrate targets (HTTP_ADDR, DATABASE_URL, …).
ifneq (,$(wildcard $(ENV_FILE)))
include $(ENV_FILE)
export
endif

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/dagr ./cmd/server
	$(GO) build -o $(BIN_DIR)/dagr-worker ./cmd/worker
	$(GO) build -o $(BIN_DIR)/dagr-migrate ./cmd/migrate

run:
	$(GO) run ./cmd/server

# Rebuild and restart the API on Go file changes (uses Air + .air.toml).
# Prefer compose-infra so Postgres/Redis/MinIO stay up without the API container.
run-watch:
	$(AIR) -c .air.toml

# asynq worker (scheduled message publish + email stubs). Needs Redis.
worker-run:
	$(GO) run ./cmd/worker

test:
	$(GO) test ./...

lint:
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || $(GO) vet ./...

tidy:
	$(GO) mod tidy

migrate-up:
	$(GO) run ./cmd/migrate up

migrate-down:
	$(GO) run ./cmd/migrate down

compose-up:
	$(COMPOSE) up --build -d

# Dependencies only (no API image). Use with make run-watch.
compose-infra:
	$(COMPOSE) up -d postgres redis minio

compose-down:
	$(COMPOSE) down

docker-build:
	docker build -f deploy/Dockerfile -t dagr:local .

docs-preview:
	$(DOCS_CLI) preview

docs-check:
	$(DOCS_CLI) check .

web-install:
	$(PNPM) --dir $(WEB_DIR) install

web-dev:
	$(PNPM) --dir $(WEB_DIR) dev

# Second Electron window against the Vite server from web-dev (DAGR_INSTANCE=2).
web-dev-2:
	$(PNPM) --dir $(WEB_DIR) dev:2

web-build:
	$(PNPM) --dir $(WEB_DIR) build

web-typecheck:
	$(PNPM) --dir $(WEB_DIR) typecheck
