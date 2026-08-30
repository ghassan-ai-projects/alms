package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ghassan/alms/internal/models"
)

// SoftDelete marks a learning as deleted (soft delete).
func (s *LearningStore) SoftDelete(ctx context.Context, learningID string) error {
	query := `UPDATE learnings SET is_deleted = true, deleted_at = now() WHERE learning_id = $1`
	tag, err := s.pool.Exec(ctx, query, learningID)
	if err != nil {
		return fmt.Errorf("soft delete learning %s: %w", learningID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("learning %s: %w", learningID, models.ErrNotFound)
	}
	return nil
}

// Supersede marks a learning as superseded by another.
func (s *LearningStore) Supersede(ctx context.Context, oldID, newID string) error {
	query := `UPDATE learnings SET resolution = 'superseded', superseded_by = $2 WHERE learning_id = $1`
	tag, err := s.pool.Exec(ctx, query, oldID, newID)
	if err != nil {
		return fmt.Errorf("supersede learning %s -> %s: %w", oldID, newID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("learning %s: %w", oldID, models.ErrNotFound)
	}
	return nil
}

// UpdateScore updates the score of a learning record.
func (s *LearningStore) UpdateScore(ctx context.Context, learningID string, score float64) error {
	query := `UPDATE learnings SET score = $2 WHERE learning_id = $1`
	tag, err := s.pool.Exec(ctx, query, learningID, score)
	if err != nil {
		return fmt.Errorf("update score %s: %w", learningID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("learning %s: %w", learningID, models.ErrNotFound)
	}
	return nil
}

// UpdateEnrichment merges a JSON patch into enrichment_metadata for a learning.
// Uses JSONB || operator for shallow merge. Must NOT re-trigger "pending" status.
func (s *LearningStore) UpdateEnrichment(ctx context.Context, learningID string, enrichmentJSON json.RawMessage) error {
	query := `UPDATE learnings SET enrichment_metadata = enrichment_metadata || $2::jsonb WHERE learning_id = $1`
	tag, err := s.pool.Exec(ctx, query, learningID, enrichmentJSON)
	if err != nil {
		return fmt.Errorf("update enrichment %s: %w", learningID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("learning %s: %w", learningID, models.ErrNotFound)
	}
	return nil
}

// nullIfEmpty returns nil for empty strings (used for nullable PG columns).
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
