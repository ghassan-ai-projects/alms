package service

import (
	"context"
	"testing"

	"github.com/ghassan/alms/internal/models"
	"github.com/ghassan/alms/internal/service/storemock"
)

func TestLearningStoreDedup(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	lStore := storemock.NewLearningStore()
	learning := NewLearning(lStore)

	t.Run("store success", func(t *testing.T) {
		rec := models.LearningRecord{
			Title:     "Optimize database queries",
			Type:      models.LearningTypeConfig,
			Body:      "Use batch inserts for bulk operations",
			Severity:  models.SeverityMedium,
			Tags:      []string{"database", "performance"},
			Author:    "agent-1",
			SrcAgentID: "agent-1",
		}

		id, err := learning.Store(ctx, rec)
		if err != nil {
			t.Fatalf("Store() unexpected error: %v", err)
		}
		if id == "" {
			t.Fatal("Store() returned empty ID")
		}

		// Verify defaults were set
		got, err := lStore.Get(ctx, id)
		if err != nil {
			t.Fatalf("Get() failed: %v", err)
		}
		if got.Score != 0.5 {
			t.Errorf("Score = %f, want 0.5", got.Score)
		}
		if got.TTLDays != 90 {
			t.Errorf("TTLDays = %d, want 90", got.TTLDays)
		}
		if got.Resolution != models.ResolutionOpen {
			t.Errorf("Resolution = %q, want %q", got.Resolution, models.ResolutionOpen)
		}
		if got.Severity != models.SeverityMedium {
			t.Errorf("Severity = %q, want %q", got.Severity, models.SeverityMedium)
		}
	})

	t.Run("validation error", func(t *testing.T) {
		rec := models.LearningRecord{
			Title: "",
			Type:  models.LearningTypePattern,
		}
		_, err := learning.Store(ctx, rec)
		if err == nil {
			t.Fatal("Store() expected validation error, got nil")
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		rec := models.LearningRecord{
			Title: "Test",
			Type:  "invalid",
		}
		_, err := learning.Store(ctx, rec)
		if err == nil {
			t.Fatal("Store() expected validation error for invalid type, got nil")
		}
	})

	t.Run("preserves provided values", func(t *testing.T) {
		rec := models.LearningRecord{
			Title:     "Critical bug fix",
			Type:      models.LearningTypeFailure,
			Severity:  models.SeverityCritical,
			Score:     0.99,
			TTLDays:   180,
			Resolution: models.ResolutionOpen,
		}

		id, err := learning.Store(ctx, rec)
		if err != nil {
			t.Fatalf("Store() unexpected error: %v", err)
		}

		got, _ := lStore.Get(ctx, id)
		if got.Score != 0.99 {
			t.Errorf("Score = %f, want 0.99", got.Score)
		}
		if got.TTLDays != 180 {
			t.Errorf("TTLDays = %d, want 180", got.TTLDays)
		}
		if got.Severity != models.SeverityCritical {
			t.Errorf("Severity = %q, want %q", got.Severity, models.SeverityCritical)
		}
	})
}
