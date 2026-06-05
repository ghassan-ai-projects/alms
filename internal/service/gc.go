// Package service provides business logic for ALMS operations.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ghassan/alms/internal/models"
)

// GCConfig holds configuration for the garbage collector.
type GCConfig struct {
	Enabled  bool          // whether GC runs periodically
	Interval time.Duration // interval between GC sweeps (default 24h)
}

// DefaultGCConfig returns a default GC configuration.
func DefaultGCConfig() GCConfig {
	return GCConfig{
		Enabled:  true,
		Interval: 24 * time.Hour,
	}
}

// GC manages background garbage collection of stale learning records.
type GC struct {
	store   LearningStore
	config  GCConfig
	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// NewGC creates a new GC service backed by the given store.
func NewGC(store LearningStore, config GCConfig) *GC {
	return &GC{
		store:  store,
		config: config,
		stopCh: make(chan struct{}),
	}
}

// Start begins the background GC goroutine. Does nothing if GC is disabled.
func (g *GC) Start(ctx context.Context) {
	if !g.config.Enabled {
		slog.Info("GC is disabled")
		return
	}

	g.mu.Lock()
	if g.running {
		g.mu.Unlock()
		return
	}
	g.running = true
	g.mu.Unlock()

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		slog.Info("GC started", "interval", g.config.Interval)

		ticker := time.NewTicker(g.config.Interval)
		defer ticker.Stop()

		// Run an initial sweep
		g.runSweep(ctx)

		for {
			select {
			case <-ticker.C:
				g.runSweep(ctx)
			case <-g.stopCh:
				slog.Info("GC stopped")
				return
			case <-ctx.Done():
				slog.Info("GC stopped by context")
				return
			}
		}
	}()
}

// Stop signals the GC goroutine to stop and waits for it to finish.
func (g *GC) Stop() {
	g.mu.Lock()
	if !g.running {
		g.mu.Unlock()
		return
	}
	g.mu.Unlock()

	close(g.stopCh)
	g.wg.Wait()

	g.mu.Lock()
	g.running = false
	g.mu.Unlock()
}

// SweepResult describes the outcome of a single GC sweep.
type SweepResult struct {
	Swept        int     // number of records examined
	Deleted      int     // number of records deleted
	Immune       int     // number of pinned records skipped
	ScoreChanged int     // records whose score was decremented but not deleted
}

// runSweep performs a single GC pass: identifies stale learnings and soft-deletes them.
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

	for _, rec := range records {
		result.Swept++

		if rec.IsPinned {
			result.Immune++
			continue
		}
		if rec.IsDeleted {
			continue
		}

		// Check if TTL has expired
		age := time.Since(rec.CreatedAt)
		daysSinceCreation := int(age.Hours() / 24)

		if rec.TTLDays <= 0 || daysSinceCreation < rec.TTLDays {
			// Not yet expired; no action
			continue
		}

		// TTL expired: delete if score < 0.3
		if rec.Score < 0.3 {
			if err := g.store.SoftDelete(ctx, rec.LearningID); err != nil {
				return result, fmt.Errorf("gc delete %s: %w", rec.LearningID, err)
			}
			result.Deleted++
		} else {
			// Apply decay for expired TTL
			g.applyDecayForSweep(ctx, rec)
			result.ScoreChanged++
		}
	}

	return result, nil
}

// applyDecayForSweep applies TTL-based score decay without returning error (best-effort during GC).
func (g *GC) applyDecayForSweep(ctx context.Context, rec models.LearningRecord) {
	if rec.IsPinned {
		return
	}
	if rec.TTLDays <= 0 {
		return
	}

	age := time.Since(rec.CreatedAt)
	daysSinceCreation := int(age.Hours() / 24)
	if daysSinceCreation < 0 {
		daysSinceCreation = 0
	}

	decayUnits := daysSinceCreation / rec.TTLDays
	if decayUnits <= 0 {
		return
	}

	decayAmount := ScoreDecrementTTL * float64(decayUnits)
	newScore := rec.Score - decayAmount
	if newScore < ScoreMin {
		newScore = ScoreMin
	}

	_ = g.store.UpdateScore(ctx, rec.LearningID, newScore)
}
