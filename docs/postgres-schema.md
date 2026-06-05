# ALMS — PostgreSQL Schema

## Tables

### `agents`
| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| `agent_id` | TEXT | PRIMARY KEY | Unique, max 64 chars |
| `display_name` | TEXT | NOT NULL DEFAULT '' | Human-readable |
| `agent_type` | TEXT | NOT NULL, CHECK (systemd, mcp_client) | |
| `capabilities` | JSONB | NOT NULL DEFAULT '{}' | `{"tools":[], "skills":[]}` |
| `metadata` | JSONB | NOT NULL DEFAULT '{}' | `{"owner":"","tags":[]}` |
| `last_sync_ts` | TIMESTAMPTZ | | Agent's sync cursor |
| `last_sync_at` | TIMESTAMPTZ | | When they last acked |
| `last_heartbeat` | TIMESTAMPTZ | | Last agent.heartbeat call |
| `health_score` | REAL | NOT NULL DEFAULT 1.0 | 0.0-1.0 |
| `created_at` | TIMESTAMPTZ | NOT NULL DEFAULT now() | |
| `updated_at` | TIMESTAMPTZ | NOT NULL DEFAULT now() | |

### `learnings`
| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| `learning_id` | UUID | PRIMARY KEY DEFAULT gen_random_uuid() | |
| `type` | TEXT | NOT NULL, CHECK (pattern,failure,config,protocol,edge_case) | |
| `title` | TEXT | NOT NULL | |
| `body` | TEXT | NOT NULL DEFAULT '' | |
| `tags` | TEXT[] | NOT NULL DEFAULT '{}' | |
| `severity` | TEXT | NOT NULL DEFAULT 'medium', CHECK (low,medium,high,critical) | |
| `author` | TEXT | NOT NULL DEFAULT '' | Human name or "agent:xxx" |
| `src_agent_id` | TEXT | FK → agents(agent_id) ON DELETE SET NULL | Which agent created it |
| `ai_generated` | BOOLEAN | NOT NULL DEFAULT false | Auto-captured vs human |
| `score` | REAL | NOT NULL DEFAULT 0.5 | 0.0-1.0 |
| `is_pinned` | BOOLEAN | NOT NULL DEFAULT false | Exempt from TTL+GC |
| `is_deleted` | BOOLEAN | NOT NULL DEFAULT false | Soft-delete |
| `resolution` | TEXT | NOT NULL DEFAULT 'open', CHECK (open,resolved,superseded) | |
| `superseded_by` | UUID | FK → learnings(learning_id) ON DELETE SET NULL | |
| `ttl_days` | INT | NOT NULL DEFAULT 90 | |
| `created_at` | TIMESTAMPTZ | NOT NULL DEFAULT now() | |
| `deleted_at` | TIMESTAMPTZ | | |
| `search_vector` | TSVECTOR | GENERATED ALWAYS AS (to_tsvector('english', title \|\| ' ' \|\| body)) STORED | |

### `learning_acknowledgements`
| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| `agent_id` | TEXT | NOT NULL, FK → agents(agent_id) ON DELETE CASCADE | |
| `learning_id` | UUID | NOT NULL, FK → learnings(learning_id) ON DELETE CASCADE | |
| `acknowledged_at` | TIMESTAMPTZ | NOT NULL DEFAULT now() | |
| PRIMARY KEY | (agent_id, learning_id) | | |

### `protocols`
| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| `protocol_id` | UUID | PRIMARY KEY DEFAULT gen_random_uuid() | |
| `title` | TEXT | NOT NULL | |
| `body` | TEXT | NOT NULL DEFAULT '' | |
| `target_tags` | TEXT[] | NOT NULL DEFAULT '{}' | Agent tags this applies to |
| `version` | INT | NOT NULL DEFAULT 1 | |
| `author` | TEXT | NOT NULL DEFAULT '' | |
| `is_active` | BOOLEAN | NOT NULL DEFAULT true | |
| `created_at` | TIMESTAMPTZ | NOT NULL DEFAULT now() | |
| `updated_at` | TIMESTAMPTZ | | Set on version bump |

## Indexes

```sql
-- Sync: fast lookup of learnings since cursor (the critical query)
CREATE INDEX idx_learnings_created_at ON learnings (created_at DESC);

-- Full-text search via GIN vector index
CREATE INDEX idx_learnings_search ON learnings USING GIN (search_vector);

-- Filter by type (used in learning.sync + learning.search)
CREATE INDEX idx_learnings_type ON learnings (type);

-- Filter by tags (GIN array index)
CREATE INDEX idx_learnings_tags ON learnings USING GIN (tags);

-- Active-only queries (sync ignores deleted records)
CREATE INDEX idx_learnings_active ON learnings (created_at DESC) WHERE NOT is_deleted;

-- Soft-delete scans (GC queries)
CREATE INDEX idx_learnings_gc ON learnings (created_at, score) WHERE NOT is_pinned AND NOT is_deleted;

-- Protocol tag filtering
CREATE INDEX idx_protocols_tags ON protocols USING GIN (target_tags);

-- Ack lookups
CREATE INDEX idx_acknowledgements_agent ON learning_acknowledgements (agent_id);
```

## SQL Queries Reference

### Sync (the critical query)
```sql
SELECT l.* FROM learnings l
WHERE l.created_at > $1          -- since_timestamp
  AND NOT l.is_deleted
  AND ($2 IS NULL OR l.type = $2)  -- optional type filter
  AND ($3 IS NULL OR l.tags && $3) -- optional tags filter
ORDER BY l.created_at ASC;
```

### Ack validation
```sql
SELECT l.learning_id FROM learnings l
WHERE l.created_at > (SELECT last_sync_ts FROM agents WHERE agent_id = $1)
  AND NOT l.is_deleted
ORDER BY l.created_at ASC;
```

### GC sweep
```sql
UPDATE learnings SET is_deleted = true, deleted_at = now()
WHERE created_at + (ttl_days || ' days')::INTERVAL < now()
  AND score < 0.3
  AND NOT is_pinned
  AND NOT is_deleted;
```

### Protocol pull for agent
```sql
SELECT * FROM protocols
WHERE is_active
  AND (target_tags && $1 OR target_tags = '{}')
ORDER BY created_at DESC;
```
