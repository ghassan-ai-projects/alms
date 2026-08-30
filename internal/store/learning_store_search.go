package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghassan/alms/internal/models"
)

// Search performs full-text search on learnings using PostgreSQL tsquery.
func (s *LearningStore) Search(ctx context.Context, query, ltype string, tags []string, limit int) ([]models.LearningRecord, error) {
	if limit <= 0 {
		limit = 20
	}

	args := make([]any, 0, 5)
	args = append(args, query)

	q := `
		SELECT l.learning_id, l.type, l.title, l.body, l.tags, l.severity, l.author,
		       l.src_agent_id, l.ai_generated, l.score, l.is_pinned, l.resolution,
		       l.superseded_by, l.ttl_days, l.created_at, l.enrichment_metadata
		FROM learnings l
		WHERE l.search_vector @@ plainto_tsquery('english', $1)
		  AND NOT l.is_deleted
	`
	argIdx := 2
	if ltype != "" {
		q += fmt.Sprintf(" AND l.type = $%d", argIdx)
		args = append(args, ltype)
		argIdx++
	}
	if len(tags) > 0 {
		q += fmt.Sprintf(" AND l.tags && $%d", argIdx)
		args = append(args, tags)
		argIdx++
	}
	q += fmt.Sprintf(" ORDER BY l.score DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("search learnings: %w", err)
	}
	defer rows.Close()

	return scanLearnings(rows)
}

// SearchWithStatus performs the same search as Search() with additional filters.
// If status is non-empty, filters by enrichment_metadata->>'status' = status.
// If includeRejected is false (default), filters out rejected learnings.
func (s *LearningStore) SearchWithStatus(ctx context.Context, query, ltype string, tags []string, limit int, status string, includeRejected bool) ([]models.LearningRecord, error) {
	if limit <= 0 {
		limit = 20
	}

	q, args := buildSearchWithStatusQuery(query, ltype, tags, limit, status, includeRejected)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("search learnings: %w", err)
	}
	defer rows.Close()

	return scanLearnings(rows)
}

func buildSearchWithStatusQuery(query, ltype string, tags []string, limit int, status string, includeRejected bool) (string, []any) {
	args := make([]any, 0, 6)

	q := `
		SELECT l.learning_id, l.type, l.title, l.body, l.tags, l.severity, l.author,
		       l.src_agent_id, l.ai_generated, l.score, l.is_pinned, l.resolution,
		       l.superseded_by, l.ttl_days, l.created_at, l.enrichment_metadata
		FROM learnings l
		WHERE NOT l.is_deleted
	`
	argIdx := 1

	if strings.TrimSpace(query) != "" {
		q += fmt.Sprintf(" AND l.search_vector @@ plainto_tsquery('english', $%d)", argIdx)
		args = append(args, query)
		argIdx++
	}

	if ltype != "" {
		q += fmt.Sprintf(" AND l.type = $%d", argIdx)
		args = append(args, ltype)
		argIdx++
	}
	if len(tags) > 0 {
		q += fmt.Sprintf(" AND l.tags && $%d", argIdx)
		args = append(args, tags)
		argIdx++
	}
	if status != "" {
		q += fmt.Sprintf(" AND l.enrichment_metadata->>'status' = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if !includeRejected {
		q += ` AND (l.enrichment_metadata->>'is_visible' IS NULL OR l.enrichment_metadata->>'is_visible' != 'false')`
	}

	q += fmt.Sprintf(" ORDER BY l.score DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	return q, args
}
