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
