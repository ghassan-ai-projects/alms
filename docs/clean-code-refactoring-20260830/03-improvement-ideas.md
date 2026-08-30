# Improvement Ideas

This document captures worthwhile features or engineering improvements noticed during the refactoring. It is intentionally non-implementation scope: ideas are recorded here and are not created as part of this task.

| ID | Area | Idea | Why it may matter | Evidence/status |
|---|---|---|---|---|
| IMP-001 | Mock infrastructure | Add deep-copy helpers for returned models so test assertions cannot mutate mock state through aliased slices or raw JSON. | Reduces false confidence from test-only state aliasing. | Proposal from storemock review; out of scope. |
| IMP-002 | Mock infrastructure | Add explicit cancellation behavior or document that mock context arguments are intentionally ignored. | Makes cancellation coverage and limitations visible. | Proposal from storemock review; out of scope. |
| IMP-003 | Sync semantics | Define deterministic tie-breaking for equal timestamps and consider a timestamp/ID cursor tuple. | Prevents unstable limited results and cursor gaps. | Proposal from storemock review; out of scope. |
| IMP-004 | Mock infrastructure | Add compile-time assertions that mock stores satisfy service store interfaces. | Detects interface drift early. | Proposal from storemock review; out of scope. |
| IMP-005 | Matching | Replace or document the ASCII-only case-insensitive matcher. | Clarifies Unicode behavior in search fixtures. | Proposal from storemock review; out of scope. |
