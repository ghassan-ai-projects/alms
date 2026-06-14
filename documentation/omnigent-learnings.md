# OmniGent Learnings For ALMS

## Scope

This note captures implementation and product lessons from the local OmniGent checkout at `/Users/ghassan/external-projects/omnigent`, reviewed against ALMS's current shape as a Go MCP server for shared agent memory.

OmniGent is much broader than ALMS: it is an agent harness, session host, web UI, policy engine, sandbox launcher, and collaboration server. ALMS should not copy that orchestration surface. The useful patterns are the ones that strengthen ALMS's narrower control-plane role: durable memory, protocol distribution, trust, sync, and operator experience.

Reviewed areas:

- `README.md`
- `docs/AGENT_YAML_SPEC.md`
- `docs/POLICIES.md`
- `omnigent/server/app.py`
- `omnigent/server/routes/session_policies.py`
- `omnigent/stores/agent_store/sqlalchemy_store.py`
- `omnigent/stores/policy_store/sqlalchemy_store.py`
- `omnigent/runtime/policies/engine.py`
- `omnigent/policies/registry.py`
- `tests/server/test_openapi_drift.py`
- `.github/workflows/ci.yml`

## High-Value Patterns

### 1. Treat Distributed Instructions As Versioned Artifacts

OmniGent stores built-in agents as deterministic tarballs whose storage key includes a SHA-256 content hash. Startup seeding is idempotent: if the content hash is unchanged, no version bump occurs; if it changed, the existing record is updated in place and cache state is refreshed.

ALMS protocols currently have `version`, `updated_at`, `is_active`, and tag targeting, but they do not have an explicit content identity. Adding content hashing would make protocol distribution more reliable and observable.

Recommended ALMS direction:

- Add a `content_hash` to protocols, computed from canonical protocol fields such as title, body, target tags, and any future metadata.
- Make `protocol.push` idempotent when the same author/title/tag scope submits identical content.
- Increment protocol version only when canonical content changes.
- Return `content_hash` and `version` from `protocol.pull` and `protocol.pull_since`.

Why this matters: agents can distinguish "already processed this exact protocol" from "same title, new content", and operators get a cleaner audit trail.

### 2. Add Trust Metadata To Protocols And Learnings

OmniGent's policy system separates trusted admin configuration from user/session-level additions. Session users can attach only registry-approved policy handlers; custom handlers must be exposed by server admins. The key idea is not Python policy execution, but trust-aware distribution.

ALMS should keep protocols as text/data, not executable code. Still, protocols can influence agent behavior, so they need provenance and trust semantics.

Recommended ALMS direction:

- Add protocol fields such as `source`, `trust_level`, `created_by_agent_id`, `created_by_role`, and optional `reviewed_by`.
- Distinguish operator-authored protocols from agent-authored learnings promoted into protocols.
- Consider a `pending_review` or `active` lifecycle for protocols published by agents.
- Include trust metadata in pull results so clients can decide whether to auto-apply, summarize, or ask a human.

Why this matters: shared memory becomes safer when consumers can tell whether guidance came from an operator, a trusted automation, or an arbitrary peer agent.

### 3. Make Protocol And Learning State Diff-Friendly

OmniGent's OpenAPI drift test performs a byte-for-byte comparison between the generated API artifact and the checked-in file, and the failure message tells developers exactly how to regenerate it. This is a good pattern for any generated or replicated contract.

ALMS does not currently expose a generated OpenAPI artifact because its public surface is MCP tools/resources. The equivalent contract is the tool/resource catalog and JSON argument schemas.

Recommended ALMS direction:

- Add a generated MCP catalog artifact under `documentation/` or `internal/server/testdata/`.
- Include tool names, descriptions, required arguments, resources, and example payloads.
- Add a drift test that regenerates the catalog from `internal/server` registration code and compares it with the checked-in artifact.
- Make the failure message include the exact regeneration command.

Why this matters: MCP clients and agent skills depend on stable tool shapes. A drift test catches accidental contract changes before release.

### 4. Prefer Cursor Semantics Over Client-Provided Time Windows

OmniGent's stores consistently use cursor-style pagination with stable ordering by timestamp plus ID. ALMS's learning sync currently accepts a client-provided `since` timestamp while also tracking acknowledgement state. The documentation already calls out that these overlap conceptually.

Recommended ALMS direction:

- Move toward an opaque sync cursor that encodes the ordered boundary, rather than requiring clients to provide raw timestamps.
- Keep ordering stable with a `(created_at, learning_id)` boundary.
- Return `next_cursor` from `learning.sync`.
- Have `learning.sync_ack` acknowledge a specific cursor or batch token, not just a list of IDs.

Why this matters: timestamp cursors are easy to misuse and can behave poorly around equal timestamps, clock assumptions, filtered syncs, and retries.

### 5. Keep Runtime Policy Out Of ALMS, But Borrow Policy Concepts

OmniGent has a full runtime policy engine with ALLOW, DENY, and ASK decisions across enforcement points such as requests, tool calls, responses, and LLM traffic. ALMS is not an agent runtime and should not become one.

The transferable concept is policy metadata for ALMS's own control-plane operations.

Recommended ALMS direction:

- Add lightweight authorization policy concepts for ALMS operations, not agent tool calls.
- Examples: who can push protocols, who can soft-delete learnings, whether agent-authored critical learnings require review, and whether untrusted agents can update enrichment.
- Represent these as server configuration and store-level checks, not as arbitrary executable plugins.

Why this matters: ALMS can become safer without taking on OmniGent's orchestration responsibility.

### 6. Strengthen Agent Registration With Ownership And Capabilities

OmniGent models sessions, hosts, permissions, and agents with explicit ownership. ALMS has agent capabilities and metadata, but current workflows are mostly token-wide.

Recommended ALMS direction:

- Add optional owner or namespace metadata to agents.
- Add capability declarations that are normalized enough to query, not only free-form JSON.
- Use capabilities to target protocols and filter learnings more precisely than tags alone.
- Consider per-agent or per-namespace API tokens later, especially before multi-tenant use.

Why this matters: shared memory needs stronger boundaries once multiple agent families or teams use the same ALMS instance.

### 7. Package Integration Assets As First-Class Artifacts

OmniGent ships examples, skills, SDKs, deploy targets, and UI assets as part of the product surface. ALMS already has `prompts/`, `skill/`, and deployment scripts. The useful lesson is to treat these assets as tested product interfaces rather than ancillary files.

Recommended ALMS direction:

- Add validation for the shipped skill and prompt examples.
- Add smoke tests that parse or exercise example MCP payloads from docs.
- Keep deployment examples synchronized with config defaults via tests or a documented check.
- Consider a small "agent integration bundle" release artifact that includes the ALMS skill, prompt snippets, and generated MCP catalog.

Why this matters: ALMS adoption depends as much on clean agent integration as on the server internals.

### 8. Improve Operational Observability Around Sync

OmniGent's server code invests in low-cardinality metrics, request duration tracking, WebSocket connection metrics, and CI progress artifacts. ALMS has a smaller operational profile, but sync and protocol distribution still need visibility.

Recommended ALMS direction:

- Add counters for `learning.store`, `learning.sync`, `learning.sync_ack`, gap-detected errors, `protocol.push`, and `protocol.pull`.
- Add gauges for registered agents, recently healthy agents, active protocols, and pending/unacknowledged learning count.
- Keep labels low-cardinality: status, tool name, learning type, protocol active state. Avoid raw agent IDs as metric labels unless explicitly configured.

Why this matters: operators need to know whether agents are learning, syncing, and applying protocols, not just whether the process is alive.

## What Not To Copy

- Do not turn ALMS into an agent runner, session host, terminal multiplexer, or collaboration UI.
- Do not add arbitrary executable policy handlers to the ALMS protocol path.
- Do not adopt OmniGent's Python/FastAPI/SQLAlchemy architecture; ALMS's Go, pgx, PostgreSQL, and thin MCP transport shape is cleaner for its scope.
- Do not copy cloud sandbox management. ALMS should remain out of the execution path.
- Do not add a large web frontend before the MCP and operational contracts are hardened.

## Suggested ALMS Backlog

1. Protocol content identity: add `content_hash`, idempotent push behavior, and version bump rules.
2. MCP catalog drift guard: generate and test the public tool/resource contract.
3. Sync cursor v2: introduce `next_cursor` and batch/cursor acknowledgements while keeping current tools backward compatible during a transition.
4. Protocol trust metadata: add source/review/lifecycle fields for operator vs agent-authored instructions.
5. Integration asset validation: test shipped skill, prompts, and documentation examples.
6. Sync observability: add metrics around store/sync/protocol flows.

## Near-Term Implementation Notes

The smallest useful first change is protocol content identity. It fits ALMS's existing model without changing the runtime shape:

- Add a migration with `content_hash TEXT NOT NULL DEFAULT ''` on `protocols`.
- Compute the hash in `internal/service` before persistence.
- Add a unique or partial index only after deciding the idempotency key, likely `(title, author, content_hash)` or a more explicit future `namespace`.
- Extend model, store, service, MCP response, and tests.

The second best change is the MCP catalog drift test because it protects every future API change and requires no production schema change.
