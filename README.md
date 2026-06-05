# ALMS — Agent Learning Management System

MCP server for agent registry + cross-agent learning transfer.

## Stack
- **Language:** Go 1.22+
- **Database:** PostgreSQL 16
- **Transport:** MCP Streamable HTTP
- **Deploy:** systemd single binary

## Quick Start

```bash
# Start PostgreSQL
docker compose up -d db

# Run migrations
ALMS_PG_DSN="postgres://alms:alms@localhost:5432/alms_db?sslmode=disable"
migrate -path internal/store/migrations -database "$ALMS_PG_DSN" up

# Build and run
go build -o bin/alms ./cmd/alms/
./bin/alms
```

## Project Structure

```
cmd/alms/      — Entry point, flag parsing, DI
internal/
  config/      — YAML + env config
  models/      — Pure data structs, validation
  store/       — PostgreSQL access (pgx/v5)
  service/     — Business logic
  server/      — MCP handlers + auth
deploy/        — systemd unit, config, deploy script
```

## License

MIT
