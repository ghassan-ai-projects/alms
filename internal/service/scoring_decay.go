package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ghassan/alms/internal/models"
)

// ApplyDecay applies TTL-based score decay to a single learning record.
// Pinned learnings skip decay. Returns the new score.
func (s *ScoringEngine) ApplyDecay(ctx context.Context, learningID string) (float64, error) {
	record, err := s.store.Get(ctx, learningID)
	if err != nil {
		return 0, fmt.Errorf("apply decay: %w", err)
	}

	if record.IsPinned {
		return record.Score, nil
	}
	if record.IsDeleted {
		return 0, fmt.Errorf("%w: learning %s is deleted", models.ErrValidation, learningID)
	}
	if record.TTLDays <= 0 {
		return record.Score, nil
	}

	decayUnits := calculateDecayUnits(record.CreatedAt, record.TTLDays)
	if decayUnits <= 0 {
		return record.Score, nil
	}

	newScore := calculateDecayedScore(record.Score, decayUnits)
	if err := s.store.UpdateScore(ctx, learningID, newScore); err != nil {
		return 0, fmt.Errorf("apply decay update: %w", err)
	}

	return newScore, nil
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
	for _, record := range records {
		if record.IsPinned {
			immune++
			continue
		}
		if record.IsDeleted {
			continue
		}

		_, err := s.ApplyDecay(ctx, record.LearningID)
		if err != nil {
			return 0, 0, fmt.Errorf("batch decay for %s: %w", record.LearningID, err)
		}
		changed++
	}

	return changed, immune, nil
}

func calculateDecayUnits(createdAt time.Time, ttlDays int) int {
	daysSinceCreation := calculateElapsedDays(createdAt)
	return daysSinceCreation / ttlDays
}

func calculateElapsedDays(createdAt time.Time) int {
	daysSinceCreation := int(time.Since(createdAt).Hours() / 24)
	if daysSinceCreation < 0 {
		return 0
	}
	return daysSinceCreation
}

func calculateDecayedScore(score float64, decayUnits int) float64 {
	newScore := score - ScoreDecrementTTL*float64(decayUnits)
	if newScore < ScoreMin {
		return ScoreMin
	}
	return newScore
}
