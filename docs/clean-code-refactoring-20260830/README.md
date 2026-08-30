# Clean-Code Refactoring Task

This package tracks the production-Go clean-code refactoring requested on 2026-08-30.

## Scope

- Inspect every non-test `.go` file in the repository.
- Process files largest-first.
- Preserve behavior unless a behavior change is clearly safer or required for a cleaner design.
- Do not refactor test files.
- Keep one focused commit per completed production file where code changes are made.

## Loop

For each file:

1. Confirm the file-level bar.
2. Obtain a bounded candidate review.
3. Implement the smallest cohesive refactor.
4. Review the diff and fix findings.
5. Commit the file-scoped change.
6. Record the bar result and evidence.

Tests are intentionally deferred until all implementation and review loops are complete.

## Documents

- [Quality bar](00-quality-bar.md)
- [Tracking](01-tracking.md)
- [Candidate findings](02-candidate-findings.md)
- [Improvement ideas](03-improvement-ideas.md)
- [Per-file evidence](04-file-evidence.md)
- [Resulting production-file inventory](05-resulting-file-inventory.md)
- [GitHub Actions cost optimization](06-github-actions-optimization.md)
