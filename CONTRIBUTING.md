# Contributing

## Scope

Contributions are welcome for code, tests, documentation, prompt assets, and integration guidance.

## Before You Start

- read [README.md](README.md)
- read [documentation/README.md](documentation/README.md)
- read [AGENTS.md](AGENTS.md) if you are contributing with a coding agent

## Development Setup

```bash
docker compose up -d db
export ALMS_PG_DSN="postgres://alms:alms@localhost:5432/alms_db?sslmode=disable"
migrate -path internal/store/migrations -database "$ALMS_PG_DSN" up
make build
make test
```

## Contribution Rules

- preserve the layered architecture
- keep changes scoped and coherent
- add or update tests with code changes
- update documentation when behavior or public contracts change
- do not rewrite old migrations; add new ones

## Quality Gate

Run before opening a PR:

```bash
make ci-check
make test
```

For larger feature work, also run:

```bash
golangci-lint run ./...
```

## Pull Request Expectations

Each pull request should include:

- a clear problem statement
- the chosen approach
- any contract or migration impact
- test coverage for behavior changes
- documentation updates if the change affects users or contributors

## Documentation Contributions

The tracked public docs live in `documentation/`. The legacy `docs/` directory is ignored by git and should not be used for new public documentation.

## Conduct

By participating, you agree to follow [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
