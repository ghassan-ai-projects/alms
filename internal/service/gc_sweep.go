package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ghassan/alms/internal/models"
)

// SweepResult describes the outcome of a single GC sweep.
type SweepResult struct {
	Swept        int // number of records examined
	Deleted      int // number of records deleted
	Immune       int // number of pinned records skipped
	ScoreChanged int // records whose score was decremented but not deleted
}

// runSweep performs a sweep and logs its outcome.
func (g *GC) runSweep(ctx context.Context) {
	result, err := g.Sweep(ctx)
	if err != nil {
		slog.Error("GC sweep failed", "error", err)
		return
	}
	slog.Info("GC completed",
		"swept", result.Swept,
		"deleted", result.Deleted,
		"immune", result.Immune,
		"score_changed", result.ScoreChanged,
	)
}

// Sweep performs a single GC pass and returns statistics.
func (g *GC) Sweep(ctx context.Context) (SweepResult, error) {
	// Get all non-deleted learnings
	records, err := g.store.Search(ctx, "", "", nil, 10000)
	if err != nil {
		return SweepResult{}, fmt.Errorf("gc sweep list: %w", err)
	}

	result := SweepResult{}
	for _, record := range records {
		result.Swept++
		if err := g.processSweepRecord(ctx, record, &result); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (g *GC) processSweepRecord(ctx context.Context, record models.LearningRecord, result *SweepResult) error {
	if record.IsPinned {
		result.Immune++
		return nil
	}
	if record.IsDeleted || !isTTLExpired(record) {
		return nil
	}

	// TTL expired: delete if score < 0.3
	if record.Score < 0.3 {
		if err := g.store.SoftDelete(ctx, record.LearningID); err != nil {
			return fmt.Errorf("gc delete %s: %w", record.LearningID, err)
		}
		result.Deleted++
		return nil
	}

	// Apply decay for expired TTL
	g.applyDecayForSweep(ctx, record)
	result.ScoreChanged++
	return nil
}

func isTTLExpired(record models.LearningRecord) bool {
	if record.TTLDays <= 0 {
		return false
	}
	age := time.Since(record.CreatedAt)
	daysSinceCreation := int(age.Hours() / 24)
	return daysSinceCreation >= record.TTLDays
}
