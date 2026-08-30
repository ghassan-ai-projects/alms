package service

import (
	"context"
	"time"

	"github.com/ghassan/alms/internal/models"
)

// applyDecayForSweep applies TTL-based score decay without returning error (best-effort during GC).
func (g *GC) applyDecayForSweep(ctx context.Context, record models.LearningRecord) {
	if record.IsPinned {
		return
	}
	if record.TTLDays <= 0 {
		return
	}

	age := time.Since(record.CreatedAt)
	daysSinceCreation := int(age.Hours() / 24)
	if daysSinceCreation < 0 {
		daysSinceCreation = 0
	}

	decayUnits := daysSinceCreation / record.TTLDays
	if decayUnits <= 0 {
		return
	}

	decayAmount := ScoreDecrementTTL * float64(decayUnits)
	newScore := record.Score - decayAmount
	if newScore < ScoreMin {
		newScore = ScoreMin
	}

	_ = g.store.UpdateScore(ctx, record.LearningID, newScore)
}
