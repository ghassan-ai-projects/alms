CREATE TABLE agents (
    agent_id       TEXT PRIMARY KEY,
    display_name   TEXT NOT NULL DEFAULT '',
    agent_type     TEXT NOT NULL CHECK (agent_type IN ('systemd', 'mcp_client')),
    capabilities   JSONB NOT NULL DEFAULT '{}',
    metadata       JSONB NOT NULL DEFAULT '{}',
    last_sync_ts   TIMESTAMPTZ,
    last_sync_at   TIMESTAMPTZ,
    last_heartbeat TIMESTAMPTZ,
    health_score   REAL NOT NULL DEFAULT 1.0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE learnings (
    learning_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type           TEXT NOT NULL CHECK (type IN ('pattern','failure','config','protocol','edge_case')),
    title          TEXT NOT NULL,
    body           TEXT NOT NULL DEFAULT '',
    tags           TEXT[] NOT NULL DEFAULT '{}',
    severity       TEXT NOT NULL DEFAULT 'medium' CHECK (severity IN ('low','medium','high','critical')),
    author         TEXT NOT NULL DEFAULT '',
    src_agent_id   TEXT REFERENCES agents(agent_id) ON DELETE SET NULL,
    ai_generated   BOOLEAN NOT NULL DEFAULT false,
    score          REAL NOT NULL DEFAULT 0.5,
    is_pinned      BOOLEAN NOT NULL DEFAULT false,
    is_deleted     BOOLEAN NOT NULL DEFAULT false,
    resolution     TEXT NOT NULL DEFAULT 'open' CHECK (resolution IN ('open','resolved','superseded')),
    superseded_by  UUID REFERENCES learnings(learning_id) ON DELETE SET NULL,
    ttl_days       INT NOT NULL DEFAULT 90,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ,
    search_vector  TSVECTOR GENERATED ALWAYS AS (to_tsvector('english', title || ' ' || body)) STORED
);

CREATE INDEX idx_learnings_created_at ON learnings (created_at DESC);
CREATE INDEX idx_learnings_search ON learnings USING GIN (search_vector);
CREATE INDEX idx_learnings_type ON learnings (type);
CREATE INDEX idx_learnings_tags ON learnings USING GIN (tags);
CREATE INDEX idx_learnings_active ON learnings (created_at DESC) WHERE NOT is_deleted;
CREATE INDEX idx_learnings_gc ON learnings (created_at, score) WHERE NOT is_pinned AND NOT is_deleted;

CREATE TABLE learning_acknowledgements (
    agent_id        TEXT NOT NULL REFERENCES agents(agent_id) ON DELETE CASCADE,
    learning_id     UUID NOT NULL REFERENCES learnings(learning_id) ON DELETE CASCADE,
    acknowledged_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (agent_id, learning_id)
);

CREATE INDEX idx_acknowledgements_agent ON learning_acknowledgements (agent_id);

CREATE TABLE protocols (
    protocol_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title         TEXT NOT NULL,
    body          TEXT NOT NULL DEFAULT '',
    target_tags   TEXT[] NOT NULL DEFAULT '{}',
    version       INT NOT NULL DEFAULT 1,
    author        TEXT NOT NULL DEFAULT '',
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ
);

CREATE INDEX idx_protocols_tags ON protocols USING GIN (target_tags);
