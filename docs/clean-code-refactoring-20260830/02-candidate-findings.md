# Candidate Findings

This is an append-only working record of bounded candidate reviews. Findings are classified as:

- `Fact`: directly observed in the current checkout.
- `Inference`: a likely design consequence of observed code.
- `Proposal`: a possible improvement that is outside this refactoring scope.
- `Evidence gap`: something that cannot be confirmed until the final validation pass.

No candidate is implemented solely because it was suggested; the owning file loop decides whether it satisfies the quality bar without changing behavior.

## `internal/service/storemock/mock.go`

- `Fact`: one file contains three independent mock stores plus shared matching helpers.
- `Fact`: learning search and sync duplicate type/tag predicates; protocol pull paths duplicate active/tag filtering and ordering.
- `Proposal`: split by store responsibility and isolate shared matching helpers.
- `Behavior risk`: preserve oldest-first agent ordering, newest-first protocol ordering, unlimited `limit <= 0`, generated IDs, metadata normalization, invalid-enrichment errors, and the separate lock phases in `PullSince`.
- `Out of scope`: deep-copy snapshots, cancellation-aware mocks, deterministic equal-timestamp tie-breaking, compile-time mock assertions, and a Unicode-aware matcher.
- `Review source`: bounded sub-agent review completed without edits, tests, or commits.

## `internal/server/tools.go`

- `Fact`: registration, request decoding, service calls, validation, and response shaping are mixed in one 554-line file.
- `Proposal`: split registrations by domain and extract named request handlers while preserving registration order and MCP schemas.
- `Behavior risk`: preserve exact tool names/descriptions, required fields, defaults, error-result contract, sync timestamp default, search status semantics, health timeout, and existing ignored assertion behavior.
- `Out of scope`: malformed-input validation, enrichment retrieval error policy, injected health version, health failure detail, update-clearing semantics, and shared nil-slice response helpers.
- `Review source`: bounded sub-agent review completed without edits, tests, or commits.

## `internal/store/learning_store.go`

- `Fact`: the file combines CRUD, sync transactions, query building, row scanning, and updates in 439 lines.
- `Proposal`: split into core CRUD, search, sync, mutations, and row-scanning files; keep receiver methods and interface contracts unchanged.
- `Behavior risk`: preserve `%w` errors, transaction/rollback/commit semantics, parameterized SQL, distinct empty-query behavior, projections, and caller-supplied last-ID cursor advancement.
- `Out of scope`: deterministic sync tie-breakers, cursor derivation, acknowledgement batching, migration layout correction, and nullable JSON compatibility.
- `Review source`: bounded sub-agent review completed without edits, tests, or commits.

## `internal/service/okf.go`

- `Fact`: export orchestration, option normalization, document rendering, index rendering, and format helpers are mixed in 325 lines.
- `Proposal`: split export orchestration, options, learning-file rendering, and index rendering; preserve stable sorting and option sentinel/default semantics.
- `Behavior risk`: preserve raw ID path construction, byte-based description truncation, silent malformed-enrichment handling, Markdown title reparsing, and error wrapping unless separately approved.
- `Out of scope`: path sanitization, UTF-8-safe truncation, structured index titles, Markdown escaping, richer summary output, clock injection, and malformed metadata diagnostics.
- `Review source`: bounded sub-agent review completed without edits, tests, or commits.

## `internal/store/agent_store.go`

- `Fact`: the initial split separated list/count queries from agent mutations; the bounded follow-up review found JSON encoding/decoding still duplicated or mixed into persistence methods.
- `Change`: extracted `marshalAgentData` and `decodeAgentData` while preserving the existing distinction between strict `Get` decoding and ignored `List` decoding errors.
- `Behavior risk`: preserve upsert fields, `RowsAffected()` not-found handling, parameterized filter values, pagination defaults, and created-at ordering.
- `Out of scope`: deterministic pagination tie-breakers, strict list decode errors, typed filters, payload-size validation, and compile-time interface assertions.
- `Review source`: bounded sub-agent review completed without edits, tests, or commits.

## `internal/store/protocol_store.go`

- `Fact`: the 180-line file was cohesive but duplicated query execution helpers and embedded row mapping in collection.
- `Change`: consolidated the query helpers into `loadProtocols` and extracted `scanProtocolRow`; removed an obsolete placeholder.
- `Behavior risk`: preserve empty-cursor delegation, active/global filters, descending ordering, inactive protocol listing, parameterization, and wrapped not-found errors.
- `Out of scope`: deterministic protocol cursors, snapshot semantics, lifecycle operations, tag normalization, nil-pool validation, and compile-time assertions.
- `Review source`: bounded sub-agent review completed without edits, tests, or commits.

## `internal/service/learning.go`

- `Fact`: the 214-line service combined learning lifecycle, deduplication, enrichment score synchronization, and protocol workflows.
- `Change`: split deduplication, enrichment parsing, and protocol operations; extracted record preparation, supersession, ID validation, and score synchronization helpers.
- `Behavior risk`: preserve validation-before-defaults, score/TTL/resolution/severity defaults, timestamp assignment, partial-success supersession, exact/near dedup short-circuits, score precedence, and error wrapping.
- `Out of scope`: transactional creation/supersession, injected clocks, score validation, atomic enrichment/score updates, object-only patch validation, database-backed dedup, and separate service types.
- `Review source`: bounded sub-agent review completed without edits, tests, or commits.

## `internal/service/gc.go`

- `Fact`: the 201-line file mixed background lifecycle, sweep policy, soft deletion, and best-effort decay arithmetic.
- `Change`: split lifecycle, sweep, and decay responsibilities; extracted named helpers for loop ownership, record processing, and TTL eligibility.
- `Behavior risk`: preserve pinned precedence, deletion threshold, partial-result errors, best-effort decay updates, and current shutdown/restart behavior.
- `Evidence gap`: PostgreSQL `Search` may not treat an empty query as “all records”; this is a separate behavior concern and was not changed.
- `Out of scope`: interval validation, idempotent/restart-safe shutdown, explicit all-active store operation, pagination beyond 10,000 records, decay error observability, and audit metrics.
- `Review source`: bounded sub-agent review completed without edits, tests, or commits.

## `internal/service/dedup.go`

- `Fact`: the 201-line file mixed exact matching, near matching/ranking, Levenshtein mechanics, and supersession.
- `Change`: split the workflows and extracted named helpers for duplicate lookup, exclusion, ratio collection, and ranking; retained public names for compatibility.
- `Behavior risk`: preserve bounded search limits, exact/near result shapes, deleted/excluded filtering, byte-based similarity, equal-ratio ordering, and wrapped errors.
- `Out of scope`: persisted canonical digests, delimiter rules, Unicode-aware similarity, input limits, deterministic tie-breaking, search-semantic alignment, transactional supersession, and dedup metrics.
- `Review source`: bounded sub-agent review completed without edits, tests, or commits.

## `internal/service/scoring.go`

- `Fact`: Score adjustment, score reads, TTL decay, batch maintenance, and decay arithmetic were mixed in one 167-line file.
- `Proposal`: Keep the scoring engine and constants together, split adjustment workflows from decay workflows, and name the score-loading, validation, clamping, and persistence steps explicitly.
- `Behavior risk`: Preserve pinned precedence over deleted validation, exact error prefixes, asymmetric increment/decrement clamping, future-time decay clamping, non-positive TTL no-op behavior, the 10,000-record batch bound, and batch count semantics.
- `Out of scope`: Injected clocks, atomic or optimistic score updates, NaN/Inf/negative-input validation, database-side score bounds, pagination beyond 10,000 records, revised changed/skipped counts, and decay observability.
- `Review source`: bounded sub-agent review completed without edits, tests, or commits.
## `internal/server/resources.go`

- `Fact`: Five resource registrations mixed MCP metadata, service retrieval, health fallback, static catalog data, JSON encoding, and response shaping in one 164-line file.
- `Proposal`: Keep registration orchestration and shared JSON response construction in `resources.go`, then isolate agent, health, learning, tool-catalog, and protocol registrations by domain.
- `Behavior risk`: Preserve registration order, URI/name/description/MIME metadata, list limits, error prefixes, health degraded output, and the manually maintained tool catalog.
- `Out of scope`: Protocol-resource pagination, health failure diagnostics/version injection, catalog drift detection, authorization policy changes, and exposure reduction for full resource payloads.
- `Review source`: bounded sub-agent review completed without edits, tests, or commits.
## `internal/service/sync.go`

- `Fact`: Learning sync, gap-safe acknowledgment, and protocol pulling were combined in one 122-line service file; the two protocol methods also repeated agent-tag loading.
- `Proposal`: Keep the `Syncer` type and learning delegation in `sync.go`, isolate acknowledgments and protocol pulls, and extract ordered missing-ID detection plus shared agent-tag loading.
- `Behavior risk`: Preserve the 2020-01-01 UTC default cursor, early empty-expected-ID return, original acknowledgment slice, missing-ID order, nil tag behavior, store call order, and all error prefixes/wrapping.
- `Out of scope`: Gap semantics, cursor atomicity, interface decomposition, protocol cursor semantics, and MCP schema changes.
- `Review source`: bounded sub-agent review completed without edits, tests, or commits.
## `internal/config/config.go`

- `Fact`: `Load` combined defaults, explicit/implicit file selection, YAML parsing, warnings, and environment overrides in one 101-line file.
- `Proposal`: Keep the public configuration types and `Load` orchestration together; isolate file discovery/parsing and environment overlay helpers.
- `Behavior risk`: Preserve default-seeded unmarshalling, explicit-path behavior, first-readable candidate precedence even after parse failure, silent read failures, warning destination/text, and only-non-empty environment overrides. Preserve `Addr` formatting.
- `Out of scope`: Config validation, default changes, environment names, precedence policy, diagnostics policy, and migration/startup behavior.
- `Review source`: bounded sub-agent review completed without edits, tests, or commits.
## `cmd/alms/main.go`

- `Fact`: `main` combined flags, configuration, database setup, store/service wiring, GC lifecycle, HTTP startup, signal waiting, and shutdown in one 97-line composition root.
- `Proposal`: Extract a small `buildRuntime` dependency-wiring helper while keeping process lifecycle and server construction visible in `main`.
- `Behavior risk`: Preserve version-before-config behavior, database connection before `--migrate` exit, constructor order, GC start/defer timing, `os.Exit` paths, background server startup, and the 10-second shutdown timeout.
- `Out of scope`: Fixing the misleading migration flag, replacing `os.Exit`, changing startup/shutdown ownership, centralizing versions, or introducing a broad application container.
- `Review source`: bounded sub-agent review completed without edits, tests, or commits.
## `internal/server/server.go`

- `Fact`: `ListenAndServe` combined MCP transport creation, auth wrapping, dashboard/MCP routing, HTTP-server policy, logging, and serving in one method; `New` is otherwise an explicit composition path.
- `Proposal`: Extract handler/mux construction and HTTP-server configuration into focused helpers while retaining the existing lifecycle method and public signature.
- `Behavior risk`: Preserve auth only around MCP, public dashboard routing, route order, timeout values, nil-safe shutdown, unused context signature, logging, and `http.ErrServerClosed` handling.
- `Out of scope`: Auth policy, endpoint routing changes, timeout changes, context-driven shutdown, constructor redesign, and MCP registration behavior.
- `Review source`: bounded sub-agent review completed without edits, tests, or commits.
## `internal/models/protocol.go`

- `Fact`: `Validate` used a multi-error accumulator for a single title rule in a 33-line model file.
- `Change`: Replace the accumulator with an immediate validation return while preserving Unicode whitespace handling, original title data, error text, and `%w` wrapping.
- `Behavior risk`: Do not replace `strings.TrimSpace`, normalize stored titles, or change the `ErrValidation` sentinel.
- `Out of scope`: Body/tag/version/activity validation, constructor changes, store validation, and API changes.
- `Review source`: bounded sub-agent review completed without edits, tests, or commits.
## `internal/server/middleware.go`

- `Fact`: The middleware has three clear control-flow paths, but the unauthorized JSON-RPC response construction was embedded in the main branch.
- `Proposal`: Extract a private unauthorized-response writer and leave the middleware’s routing/auth decisions intact.
- `Behavior risk`: Preserve empty-token bypass, `X-ALMS-TOKEN`, method independence, HTTP 200, error code `-32001`, message `unauthorized`, JSON shape, and MCP-only application scope.
- `Out of scope`: Auth scheme redesign, constant-time comparison, token rotation, TLS, dashboard protection, and status-code changes.
- `Review source`: bounded sub-agent review completed without edits, tests, or commits.
## `internal/models/learning.go`

- `Fact`: The file is a cohesive cross-layer learning model containing enums, JSON shape, metadata defaults, normalization, and validation in 134 lines.
- `Disposition`: No refactor is justified without changing a shared contract. Preserve exported symbols, JSON tags, validation order, raw metadata preservation, and store-level normalization.
- `Behavior risk`: Moving defaults only to the service, trimming titles, parsing/copying metadata, or replacing exported mutable valid-value maps would change behavior or package contract.
- `Out of scope`: Service defaulting, persistence/schema behavior, deduplication, supersession, and API handling.
- `Review source`: bounded sub-agent review completed without edits, tests, or commits.

## `internal/service/registry.go`

- `Fact`: The file is a small agent-lifecycle façade over `AgentStore`; its public methods are already intent-named and focused enough.
- `Disposition`: Inspect only; do not extract helpers or alter the lookup/create/update flow.
- `Behavior risk`: The current ignored lookup errors, validation-before-ID assignment, two timestamp calls, and caller-visible registration response are observable behavior.
- `Out of scope`: Fixing lookup/create races, registration response data, validation policy, and service contract changes.
- `Review source`: bounded sub-agent review completed without edits, tests, or commits.

## `internal/models/agent.go`

- `Fact`: The file is a cohesive agent data contract plus one focused validator in 81 lines.
- `Disposition`: Inspect only; a constant extraction would add little value and the exported valid-type map is part of the package contract.
- `Behavior risk`: Preserve validation order, exact messages, byte-based ID length, whitespace behavior, embedded fields, and JSON tags.
- `Out of scope`: Agent ID contract changes, server behavior, persistence validation, and mutable-map redesign.
- `Review source`: bounded sub-agent review completed without edits, tests, or commits.

## `internal/service/interfaces.go`

- `Fact`: The file groups the three store boundaries used by service code and is 46 lines; each interface is internally cohesive.
- `Disposition`: Inspect only. Interface segregation is a worthwhile follow-up, but changing constructor types and mock requirements is a broader contract migration than this refactor.
- `Behavior risk`: Narrowing interfaces can break indirect Learning/Dedup dependencies and mocks even when runtime behavior is unchanged.
- `Out of scope`: Interface decomposition, concrete store changes, mock behavior, and API changes.
- `Review source`: bounded sub-agent review completed without edits, tests, or commits.

## `internal/store/postgres.go`

- `Fact`: `NewPool` has one cohesive responsibility: parse the DSN, apply pool policy, connect, ping, and clean up on ping failure.
- `Disposition`: Inspect only; the 37-line function is already linear and readable.
- `Behavior risk`: Preserve ping readiness, explicit pool settings, assignment order, context use, and cleanup behavior.
- `Out of scope`: Pool sizing, runtime configurability, shutdown ownership, and migration-mode behavior.
- `Review source`: bounded sub-agent review completed without edits, tests, or commits.

## `internal/server/dashboard.go`

- `Fact`: The file embeds and serves one static page through one focused handler in 22 lines.
- `Disposition`: Inspect only; extracting the closure would not materially improve ownership.
- `Behavior risk`: Preserve the exact path guard, all-method 200 behavior, content type, public route, and ignored write errors.
- `Out of scope`: Dashboard authentication, headers, HTML, and static asset/product changes.
- `Review source`: bounded sub-agent review completed without edits, tests, or commits.

## `internal/models/errors.go`

- `Fact`: The file contains four cohesive exported sentinel errors in 18 lines.
- `Disposition`: Inspect only; the current grouping is idiomatic and changing it risks sentinel identity.
- `Behavior risk`: Preserve names, messages, identity, and `%w` wrapping used by callers and API error classification.
- `Out of scope`: Error taxonomy redesign, serialization, and persistence behavior.
- `Review source`: bounded sub-agent review completed without edits, tests, or commits.

## `tools.go`

- `Fact`: The build-tagged file is an idiomatic tool-dependency anchor with no runtime logic in 8 lines.
- `Disposition`: Inspect only; no clean-code change is justified.
- `Behavior risk`: Preserve the build tag and blank imports so tool dependencies remain retained by module tooling.
- `Out of scope`: Makefile/tool installation modernization and command-resolution consistency.
- `Review source`: bounded sub-agent review completed without edits, tests, or commits.
