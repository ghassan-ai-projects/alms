package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/ghassan/alms/internal/models"
)

// List returns agents matching the optional type filter with pagination.
func (s *AgentStore) List(ctx context.Context, filter map[string]string, limit, offset int) ([]models.AgentSpec, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	query, args := buildAgentListQuery(filter, limit, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()

	return scanAgentRows(rows)
}

func buildAgentListQuery(filter map[string]string, limit, offset int) (string, []any) {
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
	return query, args
}

func scanAgentRows(rows pgx.Rows) ([]models.AgentSpec, error) {
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
		_ = decodeAgentData(capBytes, metaBytes, &spec)
		specs = append(specs, spec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	return specs, nil
}

// Count returns the total number of registered agents.
func (s *AgentStore) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM agents`
	var count int
	err := s.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count agents: %w", err)
	}
	return count, nil
}
