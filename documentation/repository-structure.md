# Repository Structure

## Top-Level Layout

```text
cmd/alms/                 entry point and process wiring
internal/config/          YAML and environment configuration
internal/models/          shared data structures and validation
internal/store/           PostgreSQL access via pgx
internal/service/         business logic
internal/server/          MCP handlers, resources, middleware, dashboard
internal/integration/     integration tests
deploy/                   deployment assets and examples
prompts/                  prompt library for ALMS-aware agents
skill/                    ALMS skill definition
documentation/            tracked public documentation
test/                     acceptance and load-test scripts
```

## Layering Rules

- `store` should not depend on `service`
- `service` should not depend on `server`
- `models` should remain the lowest-level package
- `cmd/alms` is the only place that should assemble the full application

## Why This Structure Works

- It makes testing straightforward.
- It reduces the chance of circular dependencies.
- It gives contributors a predictable place to add new behavior.
- It is easy for coding agents to navigate and modify safely.

## Files Worth Knowing First

- `cmd/alms/main.go`: process startup, config loading, service wiring, shutdown
- `internal/server/tools.go`: MCP tool definitions
- `internal/server/resources.go`: MCP resources
- `internal/service/*.go`: business rules
- `internal/store/migrations/`: schema history
- `deploy/alms.yaml`: example config

## Documentation-Adjacent Assets

- `prompts/prompts.md`: prompt contract for agents using ALMS
- `prompts/README.md`: rationale for the prompt library
- `skill/SKILL.md`: skill definition for ALMS agent workflows

These are product assets, not incidental notes, and they should be treated as part of the public integration surface.
