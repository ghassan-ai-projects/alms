# Quality Bar

The refactoring is complete only when all of the following are evidenced:

## Scope

- Every non-test `.go` file is inspected and has an exact count and recorded disposition in [the resulting-file inventory](05-resulting-file-inventory.md).
- No test file is changed.
- No unrelated file is modified.

## Clean-code structure

- No production `.go` file exceeds 250 lines after the refactoring.
- Any split creates cohesive files with clear ownership; it does not merely move arbitrary blocks.
- Public top-level functions read as a small domain-specific language.
- Functions state intent in their names.
- Functions are short enough to expose one responsibility.
- Each function stays at one abstraction level and delegates to the next level down.
- Extraction does not create pass-through helpers, speculative abstractions, or duplicated logic.

## Behavior and safety

- Existing behavior is preserved unless a deliberate behavior change is recorded in this task package.
- Error handling, authorization, persistence, ordering, and lifecycle semantics remain explicit.
- No new dependency, migration, public API, or configuration behavior is introduced by this refactor.

## Verification

- Every production-code change has same-package test coverage already present or a documented coverage gap.
- `gofmt`, `go vet`, lint, build, and the repository test suite are run once after the refactoring loop is complete.
- The final worktree, commit list, and changed-file scope are inspected.
- Any skipped check or unresolved risk is recorded in [per-file evidence](04-file-evidence.md) or the final handoff.
