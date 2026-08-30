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

### `internal/server/tools.go`

- Candidate review: The bounded review found mixed registration, request parsing, service calls, validation, and response shaping across eight workflows in one 554-line file.
- Change or disposition: Kept `tools.go` as orchestration only. Split agent registrations into focused tool files; split learning record registration from sync; separated protocol sync/management, health, enrichment, OKF export, and result marshaling. Renamed private registration functions to state intent.
- Review/fix: Preserved registration order, MCP names/descriptions/schemas, service error-result behavior, sync defaults, search status handling, health timeout, and existing ignored type assertions. Restored unrelated formatting-only drift in `server.go`. Ran `gofmt` on changed server files and `git diff --check`; tests deferred by request.
- Commit: Pending.
- Bar result: Pending final status; all changed production server files are at or below 250 lines.
- Behavior note: No transport validation policy or service behavior was changed.

### `internal/store/learning_store.go`

- Candidate review: The bounded review found CRUD, sync transactions, search query construction, mutations, and row scanning mixed in one 439-line file.
- Change or disposition: Split into `learning_store.go`, `learning_store_sync.go`, `learning_store_search.go`, `learning_store_mutations.go`, and `learning_store_rows.go`. Extracted nullable-field population, row scanning, acknowledgement insertion, and cursor advancement into intent-named helpers.
- Review/fix: Preserved SQL text, parameter numbering, `%w` wrapping, projections, empty-limit defaults, transaction rollback/commit behavior, and caller-supplied last-ID cursor semantics. Ran `gofmt` on changed files and `git diff --check`; tests deferred by request.
- Commit: Pending.
- Bar result: Pending final status; all resulting production store files are at or below 250 lines.
- Behavior note: No migration, schema, service interface, or persistence policy was changed.

### `internal/service/okf.go`

- Candidate review: The bounded review found export orchestration, option normalization, frontmatter/body rendering, index rendering, and format helpers mixed in one 325-line file.
- Change or disposition: Split into `okf.go`, `okf_options.go`, `okf_learning_file.go`, `okf_index.go`, and `okf_helpers.go`. Renamed private helpers to express their intent, including `formatOKFLearningType`, `buildOKFDescription`, `buildOKFLearningSlug`, and `extractEnrichmentStatus`.
- Review/fix: Preserved stable sort behavior, default and `all` status handling, score filtering, raw ID path construction, byte-based description truncation, malformed-enrichment handling, Markdown title extraction, and error wrapping. Ran `gofmt` on changed files and `git diff --check`; tests deferred by request.
- Commit: Pending.
- Bar result: Pending final status; all resulting production service files are at or below 250 lines.
- Behavior note: No export format or service contract was intentionally changed.

### `internal/store/agent_store.go`

- Candidate review: The bounded review accepted the two-file split but identified responsibility density in `List` and duplicated JSON encoding in `Create` and `Update`.
- Change or disposition: Kept mutations in `agent_store.go`, moved list/count queries to `agent_store_queries.go`, extracted `marshalAgentData` and `decodeAgentData` to `agent_store_encoding.go`, and kept `List` query construction and row scanning delegated to named helpers.
- Review/fix: Preserved upsert behavior, JSON error messages, strict `Get` decoding, ignored `List` decoding errors, pagination normalization, SQL parameterization, ordering, and not-found handling. Ran `gofmt` on changed files and `git diff --check`; tests deferred by request.
- Commit: Pending.
- Bar result: Pending final status; resulting production store files are at or below 250 lines.
- Behavior note: No schema or service-interface changes were introduced.

### `internal/store/protocol_store.go`

- Candidate review: The bounded review found duplicate query executors, responsibility density in `PullSince`, and row mapping embedded in collection.
- Change or disposition: Consolidated `queryProtocols` and `queryProtocolsArgs` into variadic `loadProtocols`, extracted `scanProtocolRow`, and removed the obsolete placeholder comment.
- Review/fix: Preserved direct `PullSince` delegation for an empty cursor, active/global filters, descending ordering, list inclusion of inactive protocols, parameterization, row-close/error handling, and `%w` not-found wrapping. Ran `gofmt` and `git diff --check`; tests deferred by request.
- Commit: Pending.
- Bar result: Pending final status; file remains cohesive and is 166 lines after cleanup.
- Behavior note: No protocol lifecycle, schema, or service-interface behavior was changed.

### `internal/service/learning.go`

- Candidate review: The bounded review found validation/defaulting/persistence/supersession, deduplication, enrichment score synchronization, and protocol operations mixed in one 214-line service file.
- Change or disposition: Split deduplication into `learning_dedup.go`, enrichment parsing into `learning_enrichment.go`, and protocol operations into `protocol_service.go`. Extracted `prepareLearningRecord`, `handleLearningSupersession`, `validateLearningID`, and `synchronizeEnrichmentScore` so top-level service methods read as workflows.
- Review/fix: Corrected the extracted preparation path to preserve `CreatedAt = time.Now()`, defaults, normalization, partial-success supersession behavior, score precedence, and silent non-score enrichment handling. Ran `gofmt` and `git diff --check`; tests deferred by request.
- Commit: Pending.
- Bar result: Pending final status; resulting production service files are at or below 250 lines.
- Behavior note: No public service method name or store interface changed.
