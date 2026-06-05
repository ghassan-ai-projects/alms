// Package service provides business logic for ALMS operations.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ghassan/alms/internal/models"
)

// ScoringEngine manages learning record score operations.
type ScoringEngine struct {
	store LearningStore
}

// NewScoringEngine creates a new ScoringEngine backed by the given store.
func NewScoringEngine(store LearningStore) *ScoringEngine {
	return &ScoringEngine{store: store}
}

const (
	// ScoreIncrementSync is the score added each time a learning is successfully synced.
	ScoreIncrementSync = 0.1
	// ScoreDecrementTTL is the score decremented per TTL day without update.
	ScoreDecrementTTL = 0.1
	// ScoreMin is the minimum score allowed.
	ScoreMin = 0.0
	// ScoreMax is the maximum score allowed.
	ScoreMax = 1.0
	// ScoreDefault is the default score for new learnings.
	ScoreDefault = 0.5
)

// IncrementScore increases the score for a learning record by the sync increment.
// Pinned learnings are immune to scoring changes.
func (s *ScoringEngine) IncrementScore(ctx context.Context, learningID string) error {
	rec, err := s.store.Get(ctx, learningID)
	if err != nil {
		return fmt.Errorf("increment score: %w", err)
	}
	if rec.IsPinned {
		return nil // pinned learnings are immune
	}
	if rec.IsDeleted {
		return fmt.Errorf("%w: learning %s is deleted", models.ErrValidation, learningID)
	}

	newScore := rec.Score + ScoreIncrementSync
	if newScore > ScoreMax {
		newScore = ScoreMax
	}

	if err := s.store.UpdateScore(ctx, learningID, newScore); err != nil {
		return fmt.Errorf("increment score: %w", err)
	}
	return nil
}

// DecrementScore decreases the score for a learning record by the given amount.
// Pinned learnings are immune to scoring changes.
func (s *ScoringEngine) DecrementScore(ctx context.Context, learningID string, amount float64) error {
	rec, err := s.store.Get(ctx, learningID)
	if err != nil {
		return fmt.Errorf("decrement score: %w", err)
	}
	if rec.IsPinned {
		return nil // pinned learnings are immune
	}
	if rec.IsDeleted {
		return fmt.Errorf("%w: learning %s is deleted", models.ErrValidation, learningID)
	}

	newScore := rec.Score - amount
	if newScore < ScoreMin {
		newScore = ScoreMin
	}

	if err := s.store.UpdateScore(ctx, learningID, newScore); err != nil {
		return fmt.Errorf("decrement score: %w", err)
	}
	return nil
}

// ApplyDecay applies TTL-based score decay to a single learning record.
// Pinned learnings skip decay. Returns the new score.
func (s *ScoringEngine) ApplyDecay(ctx context.Context, learningID string) (float64, error) {
	rec, err := s.store.Get(ctx, learningID)
	if err != nil {
		return 0, fmt.Errorf("apply decay: %w", err)
	}

	if rec.IsPinned {
		return rec.Score, nil // pinned learnings skip decay
	}
	if rec.IsDeleted {
		return 0, fmt.Errorf("%w: learning %s is deleted", models.ErrValidation, learningID)
	}

	// Calculate how many TTL intervals have passed
	age := time.Since(rec.CreatedAt)
	daysSinceCreation := int(age.Hours() / 24)
	if daysSinceCreation < 0 {
		daysSinceCreation = 0
	}

	if rec.TTLDays <= 0 {
		return rec.Score, nil // no decay for zero/invalid TTL
	}

	// Apply decay: -0.1 per TTL day since creation
	decayUnits := daysSinceCreation / rec.TTLDays
	if decayUnits <= 0 {
		return rec.Score, nil
	}

	decayAmount := ScoreDecrementTTL * float64(decayUnits)
	newScore := rec.Score - decayAmount
	if newScore < ScoreMin {
		newScore = ScoreMin
	}

	if err := s.store.UpdateScore(ctx, learningID, newScore); err != nil {
		return 0, fmt.Errorf("apply decay update: %w", err)
	}

	return newScore, nil
}

// GetScore returns the current score of a learning record.
func (s *ScoringEngine) GetScore(ctx context.Context, learningID string) (float64, error) {
	rec, err := s.store.Get(ctx, learningID)
	if err != nil {
		return 0, fmt.Errorf("get score: %w", err)
	}
	return rec.Score, nil
}

// BatchApplyDecay applies decay to all active learnings and returns stats.
// Used by GC for periodic score maintenance.
func (s *ScoringEngine) BatchApplyDecay(ctx context.Context) (int, int, error) {
	// Get all non-deleted learnings
	records, err := s.store.Search(ctx, "", "", nil, 10000)
	if err != nil {
		return 0, 0, fmt.Errorf("batch decay list: %w", err)
	}

	changed := 0
	immune := 0
	for _, rec := range records {
		if rec.IsPinned {
			immune++
			continue
		}
		if rec.IsDeleted {
			continue
		}

		_, err := s.ApplyDecay(ctx, rec.LearningID)
		if err != nil {
			return 0, 0, fmt.Errorf("batch decay for %s: %w", rec.LearningID, err)
		}
		changed++
	}

	return changed, immune, nil
}
