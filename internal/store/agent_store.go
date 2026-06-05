package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ghassan/alms/internal/models"
)

// AgentStore provides CRUD operations for agents in PostgreSQL.
type AgentStore struct {
	pool *pgxpool.Pool
}

// NewAgentStore creates a new AgentStore backed by the given pool.
func NewAgentStore(pool *pgxpool.Pool) *AgentStore {
	return &AgentStore{pool: pool}
}

// Create inserts a new agent record.
func (s *AgentStore) Create(ctx context.Context, spec models.AgentSpec) error {
	capBytes, err := json.Marshal(spec.Capabilities)
	if err != nil {
		return fmt.Errorf("marshal capabilities: %w", err)
	}
	metaBytes, err := json.Marshal(spec.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	query := `
		INSERT INTO agents (agent_id, display_name, agent_type, capabilities, metadata, last_sync_ts, last_sync_at, last_heartbeat, health_score)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (agent_id) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			agent_type = EXCLUDED.agent_type,
			capabilities = EXCLUDED.capabilities,
			metadata = EXCLUDED.metadata,
			last_sync_ts = EXCLUDED.last_sync_ts,
			last_sync_at = EXCLUDED.last_sync_at,
			last_heartbeat = EXCLUDED.last_heartbeat,
			health_score = EXCLUDED.health_score,
			updated_at = now()
	`

	_, err = s.pool.Exec(ctx, query,
		spec.AgentID,
		spec.DisplayName,
		string(spec.AgentType),
		capBytes,
		metaBytes,
		spec.LastSyncTimestamp,
		spec.LastSyncAt,
		spec.LastHeartbeat,
		spec.HealthScore,
	)
	if err != nil {
		return fmt.Errorf("create agent %s: %w", spec.AgentID, err)
	}
	return nil
}

// Get retrieves a single agent by ID.
func (s *AgentStore) Get(ctx context.Context, agentID string) (models.AgentSpec, error) {
	query := `
		SELECT agent_id, display_name, agent_type, capabilities, metadata,
		       last_sync_ts, last_sync_at, last_heartbeat, health_score,
		       created_at, updated_at
		FROM agents
		WHERE agent_id = $1
	`

	var spec models.AgentSpec
	var capBytes, metaBytes []byte

	err := s.pool.QueryRow(ctx, query, agentID).Scan(
		&spec.AgentID,
		&spec.DisplayName,
		&spec.AgentType,
		&capBytes,
		&metaBytes,
		&spec.LastSyncTimestamp,
		&spec.LastSyncAt,
		&spec.LastHeartbeat,
		&spec.HealthScore,
		&spec.CreatedAt,
		&spec.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return spec, fmt.Errorf("agent %s: %w", agentID, models.ErrNotFound)
		}
		return spec, fmt.Errorf("get agent %s: %w", agentID, err)
	}

	if len(capBytes) > 0 {
		if err := json.Unmarshal(capBytes, &spec.Capabilities); err != nil {
			return spec, fmt.Errorf("unmarshal capabilities: %w", err)
		}
	}
	if len(metaBytes) > 0 {
		if err := json.Unmarshal(metaBytes, &spec.Metadata); err != nil {
			return spec, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}

	return spec, nil
}

// Update modifies an existing agent record. Returns ErrNotFound if the agent
// does not exist.
func (s *AgentStore) Update(ctx context.Context, spec models.AgentSpec) error {
	capBytes, err := json.Marshal(spec.Capabilities)
	if err != nil {
		return fmt.Errorf("marshal capabilities: %w", err)
	}
	metaBytes, err := json.Marshal(spec.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	query := `
		UPDATE agents SET
			display_name = $2,
			agent_type = $3,
			capabilities = $4,
			metadata = $5,
			last_sync_ts = $6,
			last_sync_at = $7,
			last_heartbeat = $8,
			health_score = $9,
			updated_at = now()
		WHERE agent_id = $1
	`

	tag, err := s.pool.Exec(ctx, query,
		spec.AgentID,
		spec.DisplayName,
		string(spec.AgentType),
		capBytes,
		metaBytes,
		spec.LastSyncTimestamp,
		spec.LastSyncAt,
		spec.LastHeartbeat,
		spec.HealthScore,
	)
	if err != nil {
		return fmt.Errorf("update agent %s: %w", spec.AgentID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("agent %s: %w", spec.AgentID, models.ErrNotFound)
	}
	return nil
}

// Delete removes an agent by ID.
func (s *AgentStore) Delete(ctx context.Context, agentID string) error {
	query := `DELETE FROM agents WHERE agent_id = $1`
	tag, err := s.pool.Exec(ctx, query, agentID)
	if err != nil {
		return fmt.Errorf("delete agent %s: %w", agentID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("agent %s: %w", agentID, models.ErrNotFound)
	}
	return nil
}

// Heartbeat updates the last_heartbeat timestamp for an agent and returns the
// current server time.
func (s *AgentStore) Heartbeat(ctx context.Context, agentID string) (time.Time, error) {
	var now time.Time
	query := `
		UPDATE agents SET last_heartbeat = now(), updated_at = now()
		WHERE agent_id = $1
		RETURNING last_heartbeat
	`
	err := s.pool.QueryRow(ctx, query, agentID).Scan(&now)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return now, fmt.Errorf("agent %s: %w", agentID, models.ErrNotFound)
		}
		return now, fmt.Errorf("heartbeat agent %s: %w", agentID, err)
	}
	return now, nil
}

// List returns agents matching the optional type filter with pagination.
func (s *AgentStore) List(ctx context.Context, filter map[string]string, limit, offset int) ([]models.AgentSpec, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	args := make([]any, 0, 3)
	wheres := make([]string, 0, 1)

	argIdx := 1
	if agentType, ok := filter["agent_type"]; ok && agentType != "" {
		wheres = append(wheres, fmt.Sprintf("agent_type = $%d", argIdx))
		args = append(args, agentType)
		argIdx++
	}

	query := `
		SELECT agent_id, display_name, agent_type, capabilities, metadata,
		       last_sync_ts, last_sync_at, last_heartbeat, health_score,
		       created_at, updated_at
		FROM agents
	`
	if len(wheres) > 0 {
		query += " WHERE " + strings.Join(wheres, " AND ")
	}
	query += fmt.Sprintf(" ORDER BY created_at ASC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()

	var specs []models.AgentSpec
	for rows.Next() {
		var spec models.AgentSpec
		var capBytes, metaBytes []byte
		if err := rows.Scan(
			&spec.AgentID,
			&spec.DisplayName,
			&spec.AgentType,
			&capBytes,
			&metaBytes,
			&spec.LastSyncTimestamp,
			&spec.LastSyncAt,
			&spec.LastHeartbeat,
			&spec.HealthScore,
			&spec.CreatedAt,
			&spec.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan agent row: %w", err)
		}
		if len(capBytes) > 0 {
			_ = json.Unmarshal(capBytes, &spec.Capabilities)
		}
		if len(metaBytes) > 0 {
			_ = json.Unmarshal(metaBytes, &spec.Metadata)
		}
		specs = append(specs, spec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	return specs, nil
}
