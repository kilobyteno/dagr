# Migrations

SQL migrations live here as paired up/down files and are embedded via `fs.go` for the migrate CLI (`cmd/migrate`).

```
000001_auth.up.sql
000001_auth.down.sql
```

Apply:

```bash
make migrate-up
make migrate-down
```

Auth queries used by the Postgres store are also documented under `internal/repository/postgres/queries/`.
