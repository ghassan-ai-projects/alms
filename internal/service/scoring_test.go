package service

import (
	"context"
	"testing"
	"time"

	"github.com/ghassan/alms/internal/models"
)

func TestScoringEngine(t *testing.T) {
	t.Parallel()

	t.Run("increment score", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)
		ctx := context.Background()

		id, err := lStore.Create(ctx, models.LearningRecord{
			Title: "Test",
			Type:  models.LearningTypeConfig,
			Score: 0.5,
		})
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		scoring := NewScoringEngine(lStore)
		if err := scoring.IncrementScore(ctx, id); err != nil {
			t.Fatalf("IncrementScore() unexpected error: %v", err)
		}

		rec, _ := lStore.Get(ctx, id)
		expected := 0.6
		if rec.Score != expected {
			t.Errorf("Score = %f, want %f", rec.Score, expected)
		}
	})

	t.Run("decrement score", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)
		ctx := context.Background()

		id, err := lStore.Create(ctx, models.LearningRecord{
			Title: "Test",
			Type:  models.LearningTypeConfig,
			Score: 0.8,
		})
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		scoring := NewScoringEngine(lStore)
		if err := scoring.DecrementScore(ctx, id, 0.3); err != nil {
			t.Fatalf("DecrementScore() unexpected error: %v", err)
		}

		rec, _ := lStore.Get(ctx, id)
		expected := 0.5
		if rec.Score != expected {
			t.Errorf("Score = %f, want %f", rec.Score, expected)
		}
	})

	t.Run("score stays within bounds", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)
		ctx := context.Background()

		id, err := lStore.Create(ctx, models.LearningRecord{
			Title: "Test",
			Type:  models.LearningTypeConfig,
			Score: 0.05,
		})
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		scoring := NewScoringEngine(lStore)
		if err := scoring.DecrementScore(ctx, id, 0.5); err != nil {
			t.Fatalf("DecrementScore() unexpected error: %v", err)
		}

		rec, _ := lStore.Get(ctx, id)
		if rec.Score < ScoreMin {
			t.Errorf("Score %f should be at least %f", rec.Score, ScoreMin)
		}
	})

	t.Run("pinned learning immune to increments", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)
		ctx := context.Background()

		id, err := lStore.Create(ctx, models.LearningRecord{
			Title:    "Test",
			Type:     models.LearningTypeConfig,
			Score:    0.5,
			IsPinned: true,
		})
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		scoring := NewScoringEngine(lStore)
		if err := scoring.IncrementScore(ctx, id); err != nil {
			t.Fatalf("IncrementScore() on pinned should not error: %v", err)
		}

		rec, _ := lStore.Get(ctx, id)
		if rec.Score != 0.5 {
			t.Errorf("Pinned learning score changed from 0.5 to %f", rec.Score)
		}
	})

	t.Run("pinned learning immune to decrements", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)
		ctx := context.Background()

		id, err := lStore.Create(ctx, models.LearningRecord{
			Title:    "Test",
			Type:     models.LearningTypeConfig,
			Score:    0.5,
			IsPinned: true,
		})
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		scoring := NewScoringEngine(lStore)
		if err := scoring.DecrementScore(ctx, id, 0.3); err != nil {
			t.Fatalf("DecrementScore() on pinned should not error: %v", err)
		}

		rec, _ := lStore.Get(ctx, id)
		if rec.Score != 0.5 {
			t.Errorf("Pinned learning score changed from 0.5 to %f", rec.Score)
		}
	})

	t.Run("score cap at max", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)
		ctx := context.Background()

		id, err := lStore.Create(ctx, models.LearningRecord{
			Title: "Test",
			Type:  models.LearningTypeConfig,
			Score: 0.95,
		})
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		scoring := NewScoringEngine(lStore)
		for i := 0; i < 5; i++ {
			_ = scoring.IncrementScore(ctx, id)
		}

		rec, _ := lStore.Get(ctx, id)
		if rec.Score > ScoreMax {
			t.Errorf("Score %f should not exceed %f", rec.Score, ScoreMax)
		}
	})

	t.Run("apply decay", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)
		ctx := context.Background()

		id, err := lStore.Create(ctx, models.LearningRecord{
			Title:   "Test",
			Type:    models.LearningTypeConfig,
			Score:   0.5,
			TTLDays: 1,
		})
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		scoring := NewScoringEngine(lStore)

		// Immediately after creation, there should be no decay
		newScore, err := scoring.ApplyDecay(ctx, id)
		if err != nil {
			t.Fatalf("ApplyDecay() unexpected error: %v", err)
		}
		if newScore != 0.5 {
			t.Errorf("Expected score 0.5 immediately after creation, got %f", newScore)
		}
	})

	t.Run("pinned learning skips decay", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)
		ctx := context.Background()

		id, err := lStore.Create(ctx, models.LearningRecord{
			Title:    "Test",
			Type:     models.LearningTypeConfig,
			Score:    0.5,
			IsPinned: true,
			TTLDays:  1,
		})
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		scoring := NewScoringEngine(lStore)
		newScore, err := scoring.ApplyDecay(ctx, id)
		if err != nil {
			t.Fatalf("ApplyDecay() on pinned should not error: %v", err)
		}
		if newScore != 0.5 {
			t.Errorf("Pinned learning should not decay, got %f", newScore)
		}
	})

	t.Run("get score returns correct value", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)
		ctx := context.Background()

		id, err := lStore.Create(ctx, models.LearningRecord{
			Title: "Test",
			Type:  models.LearningTypeConfig,
			Score: 0.75,
		})
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		scoring := NewScoringEngine(lStore)
		score, err := scoring.GetScore(ctx, id)
		if err != nil {
			t.Fatalf("GetScore() unexpected error: %v", err)
		}
		if score != 0.75 {
			t.Errorf("GetScore() = %f, want 0.75", score)
		}
	})

	t.Run("get score non-existent returns error", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)
		ctx := context.Background()

		scoring := NewScoringEngine(lStore)
		_, err := scoring.GetScore(ctx, "non-existent")
		if err == nil {
			t.Error("GetScore() expected error for non-existent ID")
		}
	})

	t.Run("batch apply decay", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)
		ctx := context.Background()

		_, _ = lStore.Create(ctx, models.LearningRecord{
			Title:     "Old Learning",
			Type:      models.LearningTypeConfig,
			Score:     0.5,
			TTLDays:   1,
			CreatedAt: time.Now().Add(-48 * time.Hour),
		})

		scoring := NewScoringEngine(lStore)
		changed, immune, err := scoring.BatchApplyDecay(ctx)
		if err != nil {
			t.Fatalf("BatchApplyDecay() unexpected error: %v", err)
		}
		if changed < 1 {
			t.Errorf("BatchApplyDecay() changed = %d, want >= 1", changed)
		}
		if immune != 0 {
			t.Errorf("BatchApplyDecay() immune = %d, want 0", immune)
		}
	})

	t.Run("batch apply decay with pinned", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)
		ctx := context.Background()

		_, _ = lStore.Create(ctx, models.LearningRecord{
			Title:     "Pinned Learning",
			Type:      models.LearningTypeConfig,
			Score:     0.5,
			TTLDays:   1,
			IsPinned:  true,
			CreatedAt: time.Now().Add(-48 * time.Hour),
		})

		scoring := NewScoringEngine(lStore)
		_, immune, err := scoring.BatchApplyDecay(ctx)
		if err != nil {
			t.Fatalf("BatchApplyDecay() unexpected error: %v", err)
		}
		if immune != 1 {
			t.Errorf("BatchApplyDecay() immune = %d, want 1", immune)
		}
	})
}
