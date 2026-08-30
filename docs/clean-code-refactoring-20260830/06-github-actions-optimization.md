# GitHub Actions Cost Optimization

Date: 2026-08-31

## Baseline findings

The ALMS workflow already used one CI job, but each run rebuilt `deadcode`,
`govulncheck`, and `golangci-lint` from `@latest`. Pull requests also built and
uploaded the Linux artifact, and superseded runs had no cancellation or timeout.

## Changes

- Pin the developer-tool versions to stable repository-compatible versions.
- Cache the compiled developer tools by runner, Go, and tool versions.
- Cache `golangci-lint` analysis between runs.
- Cancel an older run when a newer run exists for the same ref.
- Cap a CI job at 20 minutes.
- Keep the full correctness and security checks on pull requests.
- Cross-compile and upload the Linux artifact only on pushes to `main`.
- Restrict the workflow token to `contents: read`.

## Tradeoff

Pull requests no longer produce a downloadable Linux artifact. The artifact is
a post-merge deployment verification, while PR correctness remains covered by
build, vet, lint, race-enabled short tests, deadcode, and vulnerability checks.
No application behavior changed.

## Verification

- `git diff --check`: passed.
- The workflow was compared with the optimized Agentic Stream pattern and kept
  as one job to avoid introducing matrix or job duplication.
- The previous refactor validation, including `make test`, remains recorded in
  `01-tracking.md`.
