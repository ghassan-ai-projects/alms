# ALMS Go Design v2 — Final Review

**Reviewer:** Gemini  
**Date:** 2026-06-05  
**Document:** `alms/alms-go-design.md` (v2, post-feedback)  

---

## Scoring Summary

| Dimension | Score | Notes |
|-----------|-------|-------|
| Architecture & Layering | 10/10 | Clean dep direction, no cyclic imports, explicit layering |
| Dependencies | 9/10 | 5 explicit deps with rationale. *Minor: testify should be in the table as well.* |
| Models & Types | 10/10 | Clean validation, typed constants, sentinel conventions |
| Pre-commit Hooks | 10/10 | 6 hooks, `--fast`, rationale for exclusion in CI. Exactly as recommended. |
| Linter Config | 10/10 | `thelper`, `wrapcheck`, `musttag`, `errorlint` all present. Prealloc/unconvert/wastedassign disabled with rationale. |
| Testing Strategy | 9/10 | Store interfaces defined concretely. `testify/assert` dep acknowledged. Gap-safe ack test specified. *Minor: coverage targets are stated but no guidance on how to measure integration vs unit separately.* |
| Store Interfaces | 10/10 | All three interfaces (AgentStore, LearningStore, ProtocolStore) fully specified with method signatures. |
| AGENTS.md | 10/10 | Naming, Testing conventions, Migrations section, First Run with verification curl, Commit style — all present. |
| Bootstrap / First Run | 10/10 | Prerequisites, config example, env var setup, `docker compose`, migrations, verification curl, quick test, tear down. Complete. |
| Graceful Shutdown | 10/10 | Full `signal.NotifyContext` pattern with timeout and deferred pool close. |
| MCP Transport | 10/10 | Explicitly stated: Streamable HTTP via `mark3labs/mcp-go`. No ambiguity. |
| CI Pipeline | 9/10 | Full GitHub Actions config. *Minor: missing `golang-migrate` in CI steps for migration tests.* |
| Versioning | 10/10 | Simple, correct ldflags pattern. |
| Documentation Quality | 10/10 | Well-structured, consistent, opinionated where appropriate. |

**Overall Score: 9.7 / 10**

---

## Verdict

**RECOMMENDATION: ✅ ACCEPT**

This is a production-quality Go design document. Every feedback point from the previous review has been addressed:

1. ~~Pre-commit: 11 hooks~~ → 6 hooks with `--fast` ✅
2. ~~Dependencies vague~~ → 5 explicit with per-package justification ✅
3. ~~Missing linters~~ → `thelper`, `wrapcheck`, `musttag`, `errorlint` all added ✅
4. ~~Store interfaces missing~~ → All three interfaces fully specified ✅
5. ~~Missing AGENTS.md sections~~ → Naming, Testing, Migrations, First Run, Commit style all present ✅
6. ~~Bootstrap incomplete~~ → Full First Run with verification, quick test, teardown ✅
7. ~~No graceful shutdown~~ → Full `signal.NotifyContext` pattern with code block ✅
8. ~~MCP transport unspecified~~ → "Streamable HTTP" explicitly stated ✅

---

## Minor Points (Non-blocking)

These are suggestions for improvement, **not blockers**. They can be addressed post-acceptance.

### 1. `testify/assert` belongs in the dependency table

The testing section mentions `testify/assert` as a "small dep worth keeping" but it's absent from Section 3's dependency table. It's understandably a test-only dependency (go.mod `// indirect` would handle it), but for completeness it should appear alongside the core deps with the note `(test only)`.

### 2. CI pipeline missing migration runner

The CI workflow runs `make ci-check` and `make deadcode` but never runs `migrate up` to apply the SQL migrations before integration tests. Add:

```yaml
- name: Install golang-migrate
  run: |
    curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz
    sudo mv migrate /usr/local/bin/
- name: Apply migrations
  run: migrate -path internal/store/migrations -database "$ALMS_PG_DSN" up
```

### 3. Coverage targets: integration vs unit

Coverage targets are stated (service 80%+, store 60%+, server 40%+) but `make test` runs everything. Suggest a `go test -short` / `go test -tags=integration` split so unit coverage and integration coverage can be measured independently.

### 4. `errorlint` in linter config

The linter list includes `errorlint` (great), but the `wrapcheck` config's `ignoreSigs` already covers `fmt.Errorf`. Consider also excluding `errorlint` from checking `fmt.Errorf` wraps if it causes noise on standard patterns — but this is likely fine as-is.

---

## Verdict Summary

| Criteria | Status |
|----------|--------|
| All 8 review points addressed | ✅ |
| Architecture & layering sound | ✅ |
| Dependencies justified | ✅ |
| Pre-commit minimal + fast | ✅ |
| Linter rigourous | ✅ |
| Testing concrete (interface + test cases) | ✅ |
| First Run end-to-end | ✅ |
| CI pipeline | ✅ |
| AGENTS.md complete | ✅ |

## ✅ ACCEPT v2 as-is

This document is ready to serve as the single source of truth for implementing ALMS in Go. The four minor suggestions above can be tracked as backlog items.
