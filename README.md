# Dagr

Privacy-centric, self-hostable team chat. A Slack alternative you run yourself.

**Documentation:** [docs.page/kilobyteno/dagr](https://docs.page/kilobyteno/dagr)

## Try it

You need Docker, Node.js 20+, and pnpm. Put a [shadcnblocks](https://www.shadcnblocks.com) Pro key in `web/.env` (see `web/.env.example`).

```bash
git clone https://github.com/kilobyteno/dagr.git
cd dagr
make compose-up
```

The API listens on `http://localhost:8080`. Check it with `curl -s http://localhost:8080/api/v1/health`.

```bash
make web-install
make web-dev
```

On login, choose **Self-hosted** and enter `http://localhost:8080`. Create an account and a workspace.

Stop the stack with `make compose-down`.

The project is early. Treat it as something to try, not as production-ready chat yet.

## More

- [Compare](https://docs.page/kilobyteno/dagr/compare)
- [Quick start](https://docs.page/kilobyteno/dagr/quickstart)
- [Self-hosting](https://docs.page/kilobyteno/dagr/hosting/self-hosting)
- [Coolify](https://docs.page/kilobyteno/dagr/hosting/coolify)
- [Northflank](https://docs.page/kilobyteno/dagr/hosting/northflank)
- [Email](https://docs.page/kilobyteno/dagr/hosting/email)
- [Worker](https://docs.page/kilobyteno/dagr/hosting/worker)
- [Desktop app](https://docs.page/kilobyteno/dagr/desktop)
- [Contributing](https://docs.page/kilobyteno/dagr/contributing)

## Licence

Apache License 2.0. See [LICENSE](LICENSE).
