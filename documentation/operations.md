# Operations

## Deployment Model

ALMS is designed for single-binary deployment with PostgreSQL as an external dependency. A common production shape is:

- systemd-managed ALMS process
- local or nearby PostgreSQL instance
- reverse proxy or private network boundary in front of ALMS

The repository includes deployment assets under `deploy/`, but those files are examples, not a required topology.

## Build For Deployment

```bash
make build
make deploy-linux
```

## Run Under systemd

The repository includes `deploy/alms.service` as a starting point. Adapt paths and environment handling to your environment before using it directly.

## Database Operations

Apply migrations:

```bash
migrate -path internal/store/migrations -database "$ALMS_PG_DSN" up
```

Rollback one step:

```bash
migrate -path internal/store/migrations -database "$ALMS_PG_DSN" down 1
```

## Verification Checks

After deployment, verify:

- the process starts successfully
- `health.check` returns `ok` or a deliberately explained degraded state
- MCP tool listing works through your real ingress path
- PostgreSQL connectivity is healthy

## Backups

Back up PostgreSQL using your standard platform approach. The repository includes backup-oriented helper scripts in `deploy/`, but production backup policy should live with your database operations, not only inside application scripts.

Minimum backup policy:

- daily backups
- tested restore process
- retention policy defined by your environment

## Logging

ALMS uses `log/slog`. Route stdout/stderr into your platform logging stack and keep startup, shutdown, and database-connect logs accessible.

## Release Operations

Before publishing a release:

1. run `make ci-check`
2. run `make test`
3. verify README and `documentation/` links
4. update `CHANGELOG.md`
5. add or update release notes in `documentation/releases/`
