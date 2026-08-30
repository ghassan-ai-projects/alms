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
- Change or disposition: Replaced the monolithic mock file with `agent_store.go`, `learning_records.go`, `protocol_store.go`, and `helpers.go`. Reused `tagsOverlap` in learning filtering to remove duplicated nested loops without changing matching semantics.
- Review/fix: Reviewed the split for interface method preservation, ordering, limits, IDs, metadata normalization, and lock/error behavior. Ran `gofmt` and `git diff --check`; tests deferred by request.
- Commit: `49ad345` (`refactor: split storemock implementations by responsibility`).
- Bar result: Met for this file loop. Resulting files: `agent_store.go` 163, `helpers.go` 44, `learning_records.go` 147, `learning_search.go` 89, `learning_sync.go` 97, `protocol_store.go` 148 lines.
- Behavior note: This is test support used by service tests, but it is a non-test Go file and is included in the 250-line inventory.

### `internal/server/tools.go`

- Candidate review: The bounded review found mixed registration, request parsing, service calls, validation, and response shaping across eight workflows in one 554-line file.
- Change or disposition: Kept `tools.go` as orchestration only. Split agent registrations into focused tool files; split learning record registration from sync; separated protocol sync/management, health, enrichment, OKF export, and result marshaling. Renamed private registration functions to state intent.
- Review/fix: Preserved registration order, MCP names/descriptions/schemas, service error-result behavior, sync defaults, search status handling, health timeout, and existing ignored type assertions. Restored unrelated formatting-only drift in `server.go`. Ran `gofmt` on changed server files and `git diff --check`; tests deferred by request.
- Commit: `8c83674` (`refactor: split MCP tool registrations by domain`).
- Bar result: Met for this file loop; all changed production server files are at or below 250 lines.
- Behavior note: No transport validation policy or service behavior was changed.

### `internal/store/learning_store.go`

- Candidate review: The bounded review found CRUD, sync transactions, search query construction, mutations, and row scanning mixed in one 439-line file.
- Change or disposition: Split into `learning_store.go`, `learning_store_sync.go`, `learning_store_search.go`, `learning_store_mutations.go`, and `learning_store_rows.go`. Extracted nullable-field population, row scanning, acknowledgement insertion, and cursor advancement into intent-named helpers.
- Review/fix: Preserved SQL text, parameter numbering, `%w` wrapping, projections, empty-limit defaults, transaction rollback/commit behavior, and caller-supplied last-ID cursor semantics. Ran `gofmt` on changed files and `git diff --check`; tests deferred by request.
- Commit: `e353800` (`refactor: split learning persistence by responsibility`).
- Bar result: Met for this file loop; all resulting production store files are at or below 250 lines.
- Behavior note: No migration, schema, service interface, or persistence policy was changed.

### `internal/service/okf.go`

- Candidate review: The bounded review found export orchestration, option normalization, frontmatter/body rendering, index rendering, and format helpers mixed in one 325-line file.
- Change or disposition: Split into `okf.go`, `okf_options.go`, `okf_learning_file.go`, `okf_index.go`, and `okf_helpers.go`. Renamed private helpers to express their intent, including `formatOKFLearningType`, `buildOKFDescription`, `buildOKFLearningSlug`, and `extractEnrichmentStatus`.
- Review/fix: Preserved stable sort behavior, default and `all` status handling, score filtering, raw ID path construction, byte-based description truncation, malformed-enrichment handling, Markdown title extraction, and error wrapping. Ran `gofmt` on changed files and `git diff --check`; tests deferred by request.
- Commit: `eaec891` (`refactor: split OKF export responsibilities`).
- Bar result: Met for this file loop; all resulting production service files are at or below 250 lines.
- Behavior note: No export format or service contract was intentionally changed.

### `internal/store/agent_store.go`

- Candidate review: The bounded review accepted the two-file split but identified responsibility density in `List` and duplicated JSON encoding in `Create` and `Update`.
- Change or disposition: Kept mutations in `agent_store.go`, moved list/count queries to `agent_store_queries.go`, extracted `marshalAgentData` and `decodeAgentData` to `agent_store_encoding.go`, and kept `List` query construction and row scanning delegated to named helpers.
- Review/fix: Preserved upsert behavior, JSON error messages, strict `Get` decoding, ignored `List` decoding errors, pagination normalization, SQL parameterization, ordering, and not-found handling. Ran `gofmt` on changed files and `git diff --check`; tests deferred by request.
- Commit: `759d8a0` (`refactor: separate agent store query concerns`).
- Bar result: Met for this file loop; resulting production store files are at or below 250 lines.
- Behavior note: No schema or service-interface changes were introduced.

### `internal/store/protocol_store.go`

- Candidate review: The bounded review found duplicate query executors, responsibility density in `PullSince`, and row mapping embedded in collection.
- Change or disposition: Consolidated `queryProtocols` and `queryProtocolsArgs` into variadic `loadProtocols`, extracted `scanProtocolRow`, and removed the obsolete placeholder comment.
- Review/fix: Preserved direct `PullSince` delegation for an empty cursor, active/global filters, descending ordering, list inclusion of inactive protocols, parameterization, row-close/error handling, and `%w` not-found wrapping. Ran `gofmt` and `git diff --check`; tests deferred by request.
- Commit: `4d79cf8` (`refactor: simplify protocol store query loading`).
- Bar result: Met for this file loop; file remains cohesive and is 169 lines after cleanup.
- Behavior note: No protocol lifecycle, schema, or service-interface behavior was changed.

### `internal/service/learning.go`

- Candidate review: The bounded review found validation/defaulting/persistence/supersession, deduplication, enrichment score synchronization, and protocol operations mixed in one 214-line service file.
- Change or disposition: Split deduplication into `learning_dedup.go`, enrichment parsing into `learning_enrichment.go`, and protocol operations into `protocol_service.go`. Extracted `prepareLearningRecord`, `handleLearningSupersession`, `validateLearningID`, and `synchronizeEnrichmentScore` so top-level service methods read as workflows.
- Review/fix: Corrected the extracted preparation path to preserve `CreatedAt = time.Now()`, defaults, normalization, partial-success supersession behavior, score precedence, and silent non-score enrichment handling. Ran `gofmt` and `git diff --check`; tests deferred by request.
- Commit: `e8b415a` (`refactor: separate learning service workflows`).
- Bar result: Met for this file loop; resulting production service files are at or below 250 lines.
- Behavior note: No public service method name or store interface changed.

### `internal/service/gc.go`

- Candidate review: The bounded review found lifecycle management, sweep policy, deletion, and best-effort decay arithmetic mixed in one 201-line file. It also identified a separate production risk where an empty full-text query may make PostgreSQL GC ineffective.
- Change or disposition: Split lifecycle into `gc.go`, sweep orchestration/policy into `gc_sweep.go`, and best-effort decay into `gc_decay.go`. Extracted `claimRunning`, `runBackgroundLoop`, `processSweepRecord`, and `isTTLExpired`.
- Review/fix: Preserved disabled behavior, start/stop lifecycle behavior, ticker flow, pinned precedence, TTL/deletion threshold, partial results on deletion failure, score-changed counting, and ignored decay-update errors. Ran `gofmt` and `git diff --check`; tests deferred by request.
- Commit: `9f55d5d` (`refactor: separate GC lifecycle and sweep policy`).
- Bar result: Met for this file loop; all resulting production service files are at or below 250 lines.
- Behavior note: The empty-query PostgreSQL GC concern, restart-safe shutdown, interval validation, pagination, and observability remain ideas only.

### `internal/service/dedup.go`

- Candidate review: The bounded review found exact matching, near matching/ranking, Levenshtein mechanics, and supersession mixed in one 201-line file.
- Change or disposition: Split shared engine/hash types into `dedup.go`, exact matching into `dedup_exact.go`, similarity/ranking into `dedup_near.go`, and supersession into `dedup_supersession.go`. Extracted intent-named private helpers while retaining public method names used by callers/tests.
- Review/fix: Preserved bounded search limits, exact short-circuit behavior including empty match IDs, near-match exclusion/deleted handling, equal-ratio ordering behavior, byte-based similarity, and wrapped errors. Ran `gofmt` and `git diff --check`; tests deferred by request.
- Commit: `f0ce74d` (`refactor: separate deduplication workflows`).
- Bar result: Met for this file loop; all resulting production service files are at or below 250 lines.
- Behavior note: Hash delimiters, Unicode similarity, deterministic tie-breaking, and atomic dedup remain ideas only.
### `internal/service/scoring.go`

- `Candidate review`: The bounded review found score adjustments, score reads, TTL decay, batch traversal, and arithmetic mixed in one 167-line service file.
- `Change or disposition`: Kept engine construction and constants in `scoring.go`, moved adjustment workflows and shared adjustment helpers to `scoring_adjustments.go`, and moved single/batch decay plus arithmetic to `scoring_decay.go`.
- `Review/fix`: Added the required models import after review caught the extracted deleted-record validation dependency. Preserved pinned/deleted precedence, clamping, future-time handling, non-positive TTL behavior, batch limit, and error wrapping. Ran `gofmt` and `git diff --check`; tests deferred by request.
- `Commit`: `cd04c6d` (`refactor: separate scoring workflows`).
- `Bar result`: Met for this file loop; resulting production service files are 25, 87, and 91 lines.
- `Behavior note`: Score atomicity, input validation, pagination, and observability remain ideas only.

### `internal/server/resources.go`

- `Candidate review`: The bounded review found five independent resource registrations mixed with JSON encoding and static catalog data in one 164-line server file.
- `Change or disposition`: Kept `registerResources` and shared JSON response construction in `resources.go`; split agent, health, learning, tool-catalog, and protocol registrations into focused files.
- `Review/fix`: Preserved registration order, resource metadata, retrieval limits, health fallback, tool catalog content, JSON MIME type, and error prefixes. Used explicit ignored request parameters in callbacks. Ran `gofmt` and `git diff --check`; tests deferred by request.
- `Commit`: `5f051c7` (`refactor: split MCP resource registrations`).
- `Bar result`: Met for this file loop; resulting production server files are 34, 27, 32, 27, 42, and 27 lines.
- `Behavior note`: Resource pagination, health diagnostics, catalog drift detection, authorization policy, and payload exposure remain ideas only.
### `internal/service/sync.go`

- `Candidate review`: The bounded review found learning sync, acknowledgment, and protocol pulling combined in one 122-line file, with repeated agent-tag retrieval.
- `Change or disposition`: Kept the `Syncer` type and learning delegation in `sync.go`; moved acknowledgment workflow to `sync_ack.go` and protocol pulls to `protocol_sync.go`; extracted `findMissingLearningIDs` and `loadAgentTags`.
- `Review/fix`: Preserved the default cursor, early empty-expected-ID return, missing-ID order, original acknowledgment slice, nil tags, store call sequence, and wrapped errors. Ran `gofmt` and `git diff --check`; tests deferred by request.
- `Commit`: `4d3df2f` (`refactor: separate sync workflows`).
- `Bar result`: Met for this file loop; resulting production service files are 35, 57, and 45 lines.
- `Behavior note`: Cursor atomicity and equal-timestamp protocol semantics remain ideas only.
### `internal/config/config.go`

- `Candidate review`: The bounded review found `Load` mixing defaults, file selection, YAML parsing, warnings, and environment overrides.
- `Change or disposition`: Kept public configuration types and `Load` orchestration in `config.go`; moved file path selection/parsing to `config_files.go` and environment application to `config_env.go`.
- `Review/fix`: Preserved default-seeded YAML unmarshalling, explicit-path behavior, first-readable-file precedence, warning output, silent read failures, and non-empty environment overrides. The independent review caught a stale duplicate `configFilePaths` body; removed it in a follow-up fix. Ran `gofmt` and `git diff --check`; tests deferred by request.
- `Commit`: `4fc3105` (`refactor: separate configuration loading`), followed by `1c4a004` (`fix: remove duplicate configuration helper`).
- `Bar result`: Met for this file loop; resulting production config files are 66, 41, and 12 lines.
- `Behavior note`: No configuration values, precedence, diagnostics, or startup behavior changed.
### `cmd/alms/main.go`

- `Candidate review`: The bounded review found flags, configuration, database setup, dependency wiring, lifecycle, HTTP startup, and shutdown combined in the composition root.
- `Change or disposition`: Extracted `runtimeServices` and `buildRuntime` into `cmd/alms/runtime.go`; retained process lifecycle, GC start/defer, explicit `server.New`, signal wait, and shutdown flow in `main`.
- `Review/fix`: Preserved version and migration ordering, store/service constructor order, GC lifecycle timing, exit paths, server goroutine startup, and shutdown timeout. Ran `gofmt` and `git diff --check`; tests deferred by request.
- `Commit`: `4149328` (`refactor: isolate runtime dependency wiring`).
- `Bar result`: Met for this file loop; `main.go` is 85 lines and `runtime.go` is 33 lines.
- `Behavior note`: The `--migrate` flag’s existing external-tool behavior remains unchanged.
### `internal/server/server.go`

- `Candidate review`: The bounded review found transport, auth wrapping, routing, HTTP-server policy, logging, and serving combined in `ListenAndServe`.
- `Change or disposition`: Extracted `buildHTTPHandler` and `newHTTPServer` into `http_server.go`; retained the public lifecycle methods and `New` composition path.
- `Review/fix`: Preserved dashboard/public routing, MCP-only authentication, route order, timeout values, logging, unused context signature, `http.ErrServerClosed` handling, and nil-safe shutdown. Ran `gofmt` and `git diff --check`; tests deferred by request.
- `Commit`: `55f4755` (`refactor: separate HTTP server assembly`).
- `Bar result`: Met for this file loop; `server.go` is 64 lines and `http_server.go` is 32 lines.
- `Behavior note`: No endpoint, authentication, timeout, or shutdown policy changed.
### `internal/models/protocol.go`

- `Candidate review`: The bounded review found a multi-error accumulator serving one validation rule.
- `Change or disposition`: Simplified `ProtocolRecord.Validate` to an immediate return for blank/whitespace-only titles; no public symbols or model fields changed.
- `Review/fix`: Preserved `strings.TrimSpace`, exact `title is required` text, `%w` wrapping, and `ErrValidation` identity. Ran `gofmt` and `git diff --check`; tests deferred by request.
- `Commit`: `429d7e0` (`refactor: simplify protocol validation`).
- `Bar result`: Met for this file loop; file is 28 lines.
- `Behavior note`: No new protocol validation rules were introduced.
### `internal/server/middleware.go`

- `Candidate review`: The bounded review found a focused middleware with response construction embedded in the unauthorized branch.
- `Change or disposition`: Extracted `writeUnauthorizedResponse` while leaving `AuthMiddleware` control flow and its public signature unchanged.
- `Review/fix`: Preserved empty-token bypass, header, status, JSON-RPC error code/message/shape, and MCP-only scope. Ran `gofmt` and `git diff --check`; tests deferred by request.
- `Commit`: `8b8d835` (`refactor: isolate unauthorized response writing`).
- `Bar result`: Met for this file loop; file is 43 lines.
- `Behavior note`: Authentication policy and response status were not changed.
### `internal/models/learning.go`

- `Candidate review`: The bounded review found a cohesive cross-layer model contract with enums, JSON shape, metadata defaults, normalization, and multi-rule validation.
- `Change or disposition`: Inspected without code changes; its public contract and layered store/service defaults do not justify a behavior-preserving split.
- `Review/fix`: Checked exported symbols, JSON tags, validation order, raw metadata preservation, and direct-store normalization concerns. No formatting or code changes were needed; tests deferred by request.
- `Commit`: None; inspection only.
- `Bar result`: Inspected; file is 134 lines and cohesive.
- `Behavior note`: Exported valid-value maps and raw metadata semantics remain unchanged.

### `internal/service/registry.go`

- `Candidate review`: The bounded review found a small, intent-named agent-lifecycle façade.
- `Change or disposition`: Inspected without code changes; no helper extraction materially improves the already focused methods.
- `Review/fix`: Verified the ignored lookup-error path, validation-before-ID assignment, two timestamp calls, and caller-visible registration response as behavior boundaries. Tests deferred by request.
- `Commit`: None; inspection only.
- `Bar result`: Inspected; file is 99 lines and cohesive.
- `Behavior note`: Lookup/create race and registration response semantics were recorded as out-of-scope concerns.

### `internal/models/agent.go`

- `Candidate review`: The bounded review found a cohesive agent contract and focused validator.
- `Change or disposition`: Inspected without code changes; a constant extraction would not materially improve the file.
- `Review/fix`: Verified validation order, exact text, byte-based ID length, whitespace behavior, embedded fields, and JSON tags. Tests deferred by request.
- `Commit`: None; inspection only.
- `Bar result`: Inspected; file is 81 lines and cohesive.
- `Behavior note`: No validation or package-contract behavior changed.

### `internal/service/interfaces.go`

- `Candidate review`: The bounded review found three cohesive service store boundaries in a 46-line contract file; it also identified possible future interface segregation.
- `Change or disposition`: Inspected without code changes; interface narrowing is a broader compatibility migration and remains a documented idea.
- `Review/fix`: Checked indirect Learning/Dedup dependencies and constructor/mocking risk. Tests deferred by request.
- `Commit`: None; inspection only.
- `Bar result`: Inspected; file is 46 lines and cohesive.
- `Behavior note`: No interface method, constructor, or concrete-store behavior changed.

### `internal/store/postgres.go`

- `Candidate review`: The bounded review found one cohesive pool setup and readiness function.
- `Change or disposition`: Inspected without code changes; the 37-line function is linear and readable.
- `Review/fix`: Verified explicit pool policies, ping failure cleanup, supplied context, and caller ownership of successful-pool cleanup. Tests deferred by request.
- `Commit`: None; inspection only.
- `Bar result`: Inspected; file is 37 lines and cohesive.
- `Behavior note`: Operational pool values remain unchanged.

### `internal/server/dashboard.go`

- `Candidate review`: The bounded review found one focused embedded-page handler.
- `Change or disposition`: Inspected without code changes; no material clean-code payoff from extracting the handler closure.
- `Review/fix`: Verified exact path guard, all-method behavior, status/content type, public route, and ignored write errors. Tests deferred by request.
- `Commit`: None; inspection only.
- `Bar result`: Inspected; file is 22 lines and cohesive.
- `Behavior note`: Dashboard access policy remains unchanged.

### `internal/models/errors.go`

- `Candidate review`: The bounded review found four cohesive exported sentinel errors.
- `Change or disposition`: Inspected without code changes; grouping and identity are already idiomatic.
- `Review/fix`: Verified exported names, messages, sentinel identity, and `%w` usage expectations. Tests deferred by request.
- `Commit`: None; inspection only.
- `Bar result`: Inspected; file is 18 lines and cohesive.
- `Behavior note`: Error classification and serialization inputs remain unchanged.

### `tools.go`

- `Candidate review`: The bounded review found an idiomatic build-tagged blank-import tool anchor.
- `Change or disposition`: Inspected without code changes; no runtime clean-code refactor is justified.
- `Review/fix`: Verified the build tag and tool dependency imports; tests deferred by request.
- `Commit`: None; inspection only.
- `Bar result`: Inspected; file is 8 lines and cohesive.
- `Behavior note`: Tool dependency retention behavior remains unchanged.
