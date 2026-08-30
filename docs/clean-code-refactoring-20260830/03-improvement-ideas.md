# Improvement Ideas

This document captures worthwhile features or engineering improvements noticed during the refactoring. It is intentionally non-implementation scope: ideas are recorded here and are not created as part of this task.

| ID | Area | Idea | Why it may matter | Evidence/status |
|---|---|---|---|---|
| IMP-001 | Mock infrastructure | Add deep-copy helpers for returned models so test assertions cannot mutate mock state through aliased slices or raw JSON. | Reduces false confidence from test-only state aliasing. | Proposal from storemock review; out of scope. |
| IMP-002 | Mock infrastructure | Add explicit cancellation behavior or document that mock context arguments are intentionally ignored. | Makes cancellation coverage and limitations visible. | Proposal from storemock review; out of scope. |
| IMP-003 | Sync semantics | Define deterministic tie-breaking for equal timestamps and consider a timestamp/ID cursor tuple. | Prevents unstable limited results and cursor gaps. | Proposal from storemock review; out of scope. |
| IMP-004 | Mock infrastructure | Add compile-time assertions that mock stores satisfy service store interfaces. | Detects interface drift early. | Proposal from storemock review; out of scope. |
| IMP-005 | Matching | Replace or document the ASCII-only case-insensitive matcher. | Clarifies Unicode behavior in search fixtures. | Proposal from storemock review; out of scope. |
| IMP-006 | Transport validation | Replace ignored request type assertions with explicit malformed-input errors. | Makes invalid MCP payload behavior diagnosable. | Proposal from server review; out of scope. |
| IMP-007 | Learning store response | Decide whether enrichment retrieval failure after `learning.store` should fail the request or return a degraded response. | Avoids silently returning incomplete enrichment state. | Proposal from server review; out of scope. |
| IMP-008 | Health | Inject the reported version and expose a safe structured failure reason or metric. | Removes hardcoded release data and improves operability. | Proposal from server review; out of scope. |
| IMP-009 | Agent update | Define whether an explicit empty display name should clear the field. | Makes update semantics unambiguous. | Proposal from server review; out of scope. |
| IMP-010 | Sync semantics | Add a deterministic timestamp/ID tie-breaker or cursor tuple for learning sync. | Prevents equal-timestamp ordering and cursor gaps. | Proposal from learning-store review; out of scope. |
| IMP-011 | Sync acknowledgements | Derive the cursor from the newest acknowledged record and consider batching acknowledgement inserts. | Avoids caller-order regressions and reduces database round trips. | Proposal from learning-store review; out of scope. |
| IMP-012 | Persistence migrations | Consolidate or correct the migration layout and documented migration command, including enrichment metadata. | Keeps fresh installations aligned with the queries in the store. | Proposal from learning-store review; out of scope. |
| IMP-013 | JSON persistence | Use a nullable-safe JSONB merge for legacy rows. | Prevents `NULL || patch` from discarding enrichment updates. | Proposal from learning-store review; out of scope. |
| IMP-014 | OKF paths | Validate or sanitize learning IDs before constructing bundle paths. | Prevents unsafe path segments in generated bundles. | Proposal from OKF review; out of scope. |
| IMP-015 | OKF descriptions | Make description truncation UTF-8 safe. | Avoids splitting multibyte characters. | Proposal from OKF review; out of scope. |
| IMP-016 | OKF index | Pass structured titles to index rendering instead of reparsing Markdown, and escape generated Markdown metadata. | Reduces format coupling and malformed output risk. | Proposal from OKF review; out of scope. |
| IMP-017 | OKF determinism | Inject the generation clock for deterministic bundle output. | Improves reproducible exports and testability. | Proposal from OKF review; out of scope. |
| IMP-018 | OKF diagnostics | Decide whether malformed enrichment metadata should remain silent or become diagnosable. | Makes export data-quality failures visible. | Proposal from OKF review; out of scope. |
| IMP-019 | Agent pagination | Add deterministic pagination ordering, such as `created_at, agent_id`. | Prevents unstable page boundaries for equal timestamps. | Proposal from agent-store review; out of scope. |
| IMP-020 | Agent JSON | Decide whether list decoding failures should fail the request and share strict/lenient decode policy explicitly. | Avoids silent data loss while preserving deliberate compatibility. | Proposal from agent-store review; out of scope. |
| IMP-021 | Agent API | Consider a typed agent filter and service-boundary validation for IDs and payload sizes. | Makes supported filters and resource limits explicit. | Proposal from agent-store review; out of scope. |
