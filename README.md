# ALMS

ALMS is a self-hosted MCP server for shared agent memory. It provides an agent registry, a cross-agent learning store, and protocol distribution without becoming an agent runtime or orchestration framework.

## Why ALMS

Teams running multiple autonomous agents usually hit the same failure mode: one agent learns something useful, but the rest of the fleet never sees it. ALMS solves that with a small control plane that agents can use to:

- register themselves and send heartbeats
- publish reusable learnings
- sync new learnings with gap-safe acknowledgement
- distribute operational protocols by tag

ALMS is designed to stay out of the hot path. Agents should continue working when ALMS is unavailable and resync later.

## What Is Included In `0.1.0`

- Go single-binary MCP server
- PostgreSQL-backed agent registry
- learning storage, search, sync, soft delete, and enrichment update flow
- protocol publishing and retrieval
- Streamable HTTP MCP transport
- deployment assets for systemd-based environments
- prompts and a skill definition for agent integration
- production-verified helper scripts for agent sync and publish workflows

## Quick Start

Prerequisites:

- Go 1.22+
- Docker and Docker Compose
- `golang-migrate`
- `golangci-lint`

Run locally:

```bash
docker compose up -d db
export ALMS_PG_DSN="postgres://alms:alms@localhost:5432/alms_db?sslmode=disable"
export ALMS_AUTH_TOKEN="change-me"
migrate -path internal/store/migrations -database "$ALMS_PG_DSN" up
make build
./bin/alms
```

Verify:

```bash
curl -s -X POST http://127.0.0.1:8001/mcp \
  -H "Content-Type: application/json" \
  -H "X-ALMS-TOKEN: change-me" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

## Documentation

The main documentation set lives in [documentation/README.md](documentation/README.md).

- Product overview: [documentation/product-overview.md](documentation/product-overview.md)
- How it works: [documentation/how-it-works.md](documentation/how-it-works.md)
- Repository structure: [documentation/repository-structure.md](documentation/repository-structure.md)
- Getting started: [documentation/getting-started.md](documentation/getting-started.md)
- Configuration: [documentation/configuration.md](documentation/configuration.md)
- MCP API reference: [documentation/mcp-api.md](documentation/mcp-api.md)
- Operations: [documentation/operations.md](documentation/operations.md)
- Security model: [documentation/security-model.md](documentation/security-model.md)
- Skills and prompts: [documentation/skills-and-prompts.md](documentation/skills-and-prompts.md)
- Agent learning integration: [documentation/agent-learning-integration.md](documentation/agent-learning-integration.md)
- Release notes for `0.1.0`: [documentation/releases/0.1.0.md](documentation/releases/0.1.0.md)

## Open Source Package

- License: [LICENSE](LICENSE)
- Changelog: [CHANGELOG.md](CHANGELOG.md)
- Contributing guide: [CONTRIBUTING.md](CONTRIBUTING.md)
- Security policy: [SECURITY.md](SECURITY.md)
- Code of conduct: [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- Support guide: [SUPPORT.md](SUPPORT.md)

ALMS is released under the MIT license. For this project, MIT is the right starting point for `0.1.0`: it is simple, widely understood, commercially permissive, and a good fit for infrastructure intended to be integrated into different agent stacks.

## Status

Current release: `0.1.0`

This is an initial public release. The project is usable, but the maintainers should expect some API and workflow hardening as real adopters push on sync semantics, security boundaries, and enrichment workflows.
