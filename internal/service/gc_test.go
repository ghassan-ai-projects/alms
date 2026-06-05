package service

import (
	"context"
	"testing"
	"time"

	"github.com/ghassan/alms/internal/models"
)

func TestGC(t *testing.T) {
	t.Parallel()

	t.Run("sweep with no eligible records", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)
		ctx := context.Background()

		_, _ = lStore.Create(ctx, models.LearningRecord{
			Title:   "Recent Learning",
			Type:    models.LearningTypeConfig,
			Score:   0.5,
			TTLDays: 90,
		})

		gc := NewGC(lStore, GCConfig{Enabled: true, Interval: 24 * time.Hour})
		result, err := gc.Sweep(ctx)
		if err != nil {
			t.Fatalf("Sweep() unexpected error: %v", err)
		}
		if result.Deleted != 0 {
			t.Errorf("Expected 0 deletions, got %d", result.Deleted)
		}
	})

	t.Run("sweep deletes expired low-score records", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)
		ctx := context.Background()

		_, _ = lStore.Create(ctx, models.LearningRecord{
			Title:     "Stale Learning",
			Type:      models.LearningTypeConfig,
			Score:     0.2,
			TTLDays:   1,
			CreatedAt: time.Now().Add(-48 * time.Hour),
		})

		gc := NewGC(lStore, GCConfig{Enabled: true, Interval: 24 * time.Hour})
		result, err := gc.Sweep(ctx)
		if err != nil {
			t.Fatalf("Sweep() unexpected error: %v", err)
		}
		if result.Deleted != 1 {
			t.Errorf("Expected 1 deletion, got %d (swept=%d)", result.Deleted, result.Swept)
		}
	})

	t.Run("pinned records are immune to sweep", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)
		ctx := context.Background()

		_, _ = lStore.Create(ctx, models.LearningRecord{
			Title:     "Pinned Learning",
			Type:      models.LearningTypeConfig,
			Score:     0.2,
			TTLDays:   1,
			IsPinned:  true,
			CreatedAt: time.Now().Add(-48 * time.Hour),
		})

		gc := NewGC(lStore, GCConfig{Enabled: true, Interval: 24 * time.Hour})
		result, err := gc.Sweep(ctx)
		if err != nil {
			t.Fatalf("Sweep() unexpected error: %v", err)
		}
		if result.Deleted != 0 {
			t.Errorf("Expected 0 deletions for pinned record, got %d", result.Deleted)
		}
		if result.Immune != 1 {
			t.Errorf("Expected 1 immune record, got %d", result.Immune)
		}
	})

	t.Run("disabled config does not start", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)

		gc := NewGC(lStore, GCConfig{Enabled: false})
		gc.Start(context.Background())
		gc.Stop()
	})

	t.Run("empty store sweep", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)
		ctx := context.Background()

		gc := NewGC(lStore, GCConfig{Enabled: true, Interval: 24 * time.Hour})
		result, err := gc.Sweep(ctx)
		if err != nil {
			t.Fatalf("Sweep() unexpected error on empty store: %v", err)
		}
		if result.Swept != 0 {
			t.Errorf("Expected 0 swept on empty store, got %d", result.Swept)
		}
		if result.Deleted != 0 {
			t.Errorf("Expected 0 deleted on empty store, got %d", result.Deleted)
		}
	})

	t.Run("high score expired record not deleted", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)
		ctx := context.Background()

		_, _ = lStore.Create(ctx, models.LearningRecord{
			Title:     "High Score Expired",
			Type:      models.LearningTypeConfig,
			Score:     0.8,
			TTLDays:   1,
			CreatedAt: time.Now().Add(-48 * time.Hour),
		})

		gc := NewGC(lStore, GCConfig{Enabled: true, Interval: 24 * time.Hour})
		result, err := gc.Sweep(ctx)
		if err != nil {
			t.Fatalf("Sweep() unexpected error: %v", err)
		}
		if result.Deleted != 0 {
			t.Errorf("Expected 0 deletions for high-score record, got %d", result.Deleted)
		}
		if result.ScoreChanged < 1 {
			t.Errorf("Expected score change for high-score expired record, got %d", result.ScoreChanged)
		}
	})

	t.Run("default GC config", func(t *testing.T) {
		cfg := DefaultGCConfig()
		if !cfg.Enabled {
			t.Error("DefaultGCConfig() should have Enabled = true")
		}
		if cfg.Interval != 24*time.Hour {
			t.Errorf("DefaultGCConfig() Interval = %v, want 24h", cfg.Interval)
		}
	})
}
