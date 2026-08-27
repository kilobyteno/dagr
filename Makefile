.PHONY: build run run-watch worker-run test lint migrate-up migrate-down compose-up compose-infra compose-down docker-build tidy docs-preview docs-check client-install client-dev client-dev-2 client-dev-web client-build client-build-web client-typecheck client-package client-docker website-install website-dev website-build website-docker

GO ?= go
COMPOSE ?= docker compose -f deploy/docker-compose.yml
BIN_DIR ?= bin
DOCS_CLI ?= npx --yes @docs.page/cli
PNPM ?= pnpm
CLIENT_DIR ?= client
WEBSITE_DIR ?= website
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

client-install:
	$(PNPM) --dir $(CLIENT_DIR) install

client-dev:
	$(PNPM) --dir $(CLIENT_DIR) dev

# Second Electron window against the Vite server from client-dev (DAGR_INSTANCE=2).
client-dev-2:
	$(PNPM) --dir $(CLIENT_DIR) dev:2

# Browser-only Vite (no Electron window).
client-dev-web:
	$(PNPM) --dir $(CLIENT_DIR) dev:web

client-build:
	$(PNPM) --dir $(CLIENT_DIR) build

client-build-web:
	$(PNPM) --dir $(CLIENT_DIR) build:web

client-typecheck:
	$(PNPM) --dir $(CLIENT_DIR) typecheck

client-package:
	$(PNPM) --dir $(CLIENT_DIR) package

client-docker:
	docker build -f client/Dockerfile -t dagr-client:local .

website-install:
	$(PNPM) --dir $(WEBSITE_DIR) install

website-dev:
	$(PNPM) --dir $(WEBSITE_DIR) dev

website-build:
	$(PNPM) --dir $(WEBSITE_DIR) build

website-docker:
	docker build -f website/Dockerfile -t dagr-website:local .
