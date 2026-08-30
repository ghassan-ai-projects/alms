# Per-File Evidence

Each completed file gets a subsection with the candidate review, chosen change or inspection disposition, self-review findings, commit, and bar result.

## Evidence template

### `<path>`

- Candidate review:
- Change or disposition:
- Review/fix:
- Commit:
- Bar result:
- Behavior note:

For split files, also record the resulting file names and line counts, and verify that each resulting production Go file is at or below 250 lines.

### `internal/service/storemock/mock.go`

- Candidate review: Three mock stores and duplicated predicates were identified; the bounded review required a cohesive split and named shared matching helpers.
- Change or disposition: Replaced the monolithic mock file with `agent_store.go`, `learning_store.go`, `protocol_store.go`, and `helpers.go`. Reused `tagsOverlap` in learning filtering to remove duplicated nested loops without changing matching semantics.
- Review/fix: Reviewed the split for interface method preservation, ordering, limits, IDs, metadata normalization, and lock/error behavior. Ran `gofmt` and `git diff --check`; tests deferred by request.
- Commit: `49ad345` (`refactor: split storemock implementations by responsibility`).
- Bar result: Met for this file loop. Resulting files: `agent_store.go` 163, `helpers.go` 44, `learning_records.go` 147, `learning_search.go` 89, `learning_sync.go` 97, `protocol_store.go` 148 lines.
- Behavior note: This is test support used by service tests, but it is a non-test Go file and is included in the 250-line inventory.
