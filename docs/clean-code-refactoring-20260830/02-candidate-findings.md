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
