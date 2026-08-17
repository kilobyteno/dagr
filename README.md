# Dagr

Privacy-centric, self-hostable team chat. A Slack alternative you run yourself.

Dagr is designed for one-command deployment, zero external SaaS dependencies, and optional end-to-end encryption for direct messages.

**Documentation:** [docs.dagr.no](https://docs.dagr.no) (served from this repo via [docs.page](https://github.com/invertase/docs.page))

## Status

Early scaffold. The HTTP API supports health, public config, and auth (signup/login/me/logout with opaque sessions). The Electron desktop shell in `web/` talks to that auth API. Channels and messaging are next.

## Quick start

### Prerequisites

- Go 1.25+
- Docker and Docker Compose
- Node.js 20+ and pnpm (desktop client)

### One-command self-host

```bash
make compose-up
```

This starts Postgres, Redis, MinIO, and the Dagr API.

Check the API:

```bash
curl -s http://localhost:8080/api/v1/health
```

Stop:

```bash
make compose-down
```

### Local API development

```bash
make compose-infra   # Postgres, Redis, MinIO (no API container)
make migrate-up
make run-watch       # Air: rebuild/restart API on .go changes
```

One-shot without a watcher:

```bash
make run
# or
make build && ./bin/dagr
```

Do not run `make compose-up` and `make run-watch` together; both bind `:8080`. Copy [`deploy/.env.example`](deploy/.env.example) for environment defaults. See the [self-hosting guide](https://docs.dagr.no/self-hosting) for details.

### Desktop client

```bash
make web-install
# add SHADCNBLOCKS_API_KEY to web/.env (see web/.env.example)
make web-dev
```

See the [desktop guide](https://docs.dagr.no/desktop) and [`web/README.md`](web/README.md).

## Documentation

Docs are MDX under [`docs/`](docs/), configured by [`docs.json`](docs.json).

```bash
make docs-preview   # local preview via docs.page CLI
make docs-check     # validate links and MDX
```

The live site is:

```text
https://docs.dagr.no
```

## Stack

| Layer | Choice |
| --- | --- |
| API | Go, chi, `/api/v1/*` |
| Database | Postgres (sqlc + pgx planned) |
| Cache / pub-sub / queues | Redis, asynq |
| Object storage | MinIO (S3-compatible) |
| Desktop | Electron, Vite, React, TypeScript, shadcn/ui, shadcnblocks |
| Search | Postgres FTS initially |
| Docs | [docs.page](https://docs.page) |

## Repository layout

```
cmd/           server, worker, and migrate entrypoints
internal/      domain, services, transport, repository, events, storage
migrations/    SQL up/down migrations
deploy/        Dockerfile and docker-compose
web/           Electron desktop client (shadcn + shadcnblocks)
docs/          documentation (docs.page MDX)
docs.json      docs.page site configuration
.cursor/       Cursor MCP config (shadcn --cwd web)
```

## Make targets

| Target | Description |
| --- | --- |
| `make build` | Build server, worker, and migrate binaries into `bin/` |
| `make run` | Run the API locally |
| `make test` | Run Go tests |
| `make lint` | Run golangci-lint if installed, otherwise `go vet` |
| `make compose-up` | Build and start the full stack |
| `make compose-down` | Stop the stack |
| `make docker-build` | Build the distroless image |
| `make web-install` | Install desktop client dependencies |
| `make web-dev` | Run the Electron desktop client |
| `make web-build` | Build and package the desktop client |
| `make docs-preview` | Preview documentation locally |
| `make docs-check` | Check documentation links and MDX |
| `make migrate-up` / `migrate-down` | Migration CLI stubs |

## Licence

Apache License 2.0. See [LICENSE](LICENSE).

## Contributing

See the [contributing guide](https://docs.dagr.no/contributing).
