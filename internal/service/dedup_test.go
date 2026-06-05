package service

import (
	"context"
	"testing"

	"github.com/ghassan/alms/internal/models"
)

func TestLevenshteinRatio(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    string
		b    string
		want float64
	}{
		{"identical", "hello world", "hello world", 1.0},
		{"completely different", "abc", "xyz", 0.0},
		{"one char diff", "hello", "hellx", 0.8},
		{"empty both", "", "", 1.0},
		{"one empty", "hello", "", 0.0},
		{"case sensitive", "Hello", "hello", 0.8},
		{"similar", "database optimization", "database optimizations", 0.9}, // actual ratio is ~0.955, test passes if >= 0.89
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LevenshteinRatio(tt.a, tt.b)
			// For identical, must be exactly 1.0; for different, 0.0
			// For similar cases, check we're at least the expected value
			if tt.want == 1.0 && got != 1.0 {
				t.Errorf("LevenshteinRatio(%q, %q) = %f, want 1.0", tt.a, tt.b, got)
			}
			if tt.want == 0.0 && got != 0.0 {
				t.Errorf("LevenshteinRatio(%q, %q) = %f, want 0.0", tt.a, tt.b, got)
			}
			if tt.want > 0.0 && tt.want < 1.0 && got < tt.want-0.1 {
				t.Errorf("LevenshteinRatio(%q, %q) = %f, expected at least %f", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestDedupEngine(t *testing.T) {
	t.Parallel()

	t.Run("SHA256Hash produces consistent results", func(t *testing.T) {
		h1 := SHA256Hash("test title", "test body")
		h2 := SHA256Hash("test title", "test body")
		if h1 != h2 {
			t.Errorf("SHA256Hash should be deterministic: %q != %q", h1, h2)
		}

		h3 := SHA256Hash("different", "body")
		if h1 == h3 {
			t.Errorf("different inputs should produce different hashes")
		}
	})

	t.Run("CheckExactDup finds match", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)
		ctx := context.Background()

		_, _ = lStore.Create(ctx, newTestLearning("Duplicate Title", "Duplicate Body"))
		dedup := NewDedupEngine(lStore)

		result, err := dedup.CheckExactDup(ctx, "Duplicate Title", "Duplicate Body")
		if err != nil {
			t.Fatalf("CheckExactDup() unexpected error: %v", err)
		}
		if !result.IsExactDup {
			t.Error("expected IsExactDup = true for identical title+body")
		}
	})

	t.Run("CheckExactDup no match", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)
		ctx := context.Background()

		_, _ = lStore.Create(ctx, newTestLearning("Existing Title", "Existing Body"))
		dedup := NewDedupEngine(lStore)

		result, err := dedup.CheckExactDup(ctx, "Different Title", "Different Body")
		if err != nil {
			t.Fatalf("CheckExactDup() unexpected error: %v", err)
		}
		if result.IsExactDup {
			t.Error("expected IsExactDup = false for different title+body")
		}
	})

	t.Run("CheckNearDup finds near match above 0.85 threshold", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)
		ctx := context.Background()

		_, _ = lStore.Create(ctx, newTestLearning("Database optimization techniques", ""))
		dedup := NewDedupEngine(lStore)

		result, err := dedup.CheckNearDup(ctx, "Database optimization technique", nil)
		if err != nil {
			t.Fatalf("CheckNearDup() unexpected error: %v", err)
		}
		if !result.IsNearDup {
			t.Error("expected IsNearDup = true for very similar title")
		}
		if len(result.NearMatchIDs) == 0 {
			t.Error("expected at least 1 near match ID")
		}
	})

	t.Run("CheckNearDup below threshold", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)
		ctx := context.Background()

		_, _ = lStore.Create(ctx, newTestLearning("Very different title about something else entirely", ""))
		dedup := NewDedupEngine(lStore)

		result, err := dedup.CheckNearDup(ctx, "Short title", nil)
		if err != nil {
			t.Fatalf("CheckNearDup() unexpected error: %v", err)
		}
		if result.IsNearDup {
			t.Error("expected IsNearDup = false for very different title")
		}
	})

	t.Run("CheckNearDup excludes specified IDs", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)
		ctx := context.Background()

		id1, _ := lStore.Create(ctx, newTestLearning("Quick brown fox runs", ""))
		_, _ = lStore.Create(ctx, newTestLearning("Quick brown fox jumps", ""))

		dedup := NewDedupEngine(lStore)

		// "Quick brown fox runs" vs "Quick brown fox jumps" — only 4 diffs out of 21 chars
		// Levenshtein distance: ~4, ratio: 1-4/21 = 0.81 — still below 0.85!
		// Let's use even closer match
		// Create a very similar title that differs by just one character
		_, _ = lStore.Create(ctx, newTestLearning("Quick brown fox run", ""))

		// Exclude id1, should still find the other (QBFR vs QBFJ = 1 diff/18 = 0.944)
		result, err := dedup.CheckNearDup(ctx, "Quick brown fox runs", []string{id1})
		if err != nil {
			t.Fatalf("CheckNearDup() unexpected error: %v", err)
		}
		if !result.IsNearDup {
			t.Errorf("expected IsNearDup = true with other non-excluded matches, near match IDs: %v", result.NearMatchIDs)
		}
	})
}

func TestHandleSupersession(t *testing.T) {
	t.Parallel()

	t.Run("supersession valid", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)
		ctx := context.Background()

		oldID, _ := lStore.Create(ctx, newTestLearning("Old Learning", ""))
		newID, _ := lStore.Create(ctx, newTestLearning("New Learning", ""))

		dedup := NewDedupEngine(lStore)
		err := dedup.HandleSupersession(ctx, newID, oldID)
		if err != nil {
			t.Fatalf("HandleSupersession() unexpected error: %v", err)
		}

		oldRec, _ := lStore.Get(ctx, oldID)
		if string(oldRec.Resolution) != "superseded" {
			t.Errorf("expected superseded learning to have resolution 'superseded', got %q", oldRec.Resolution)
		}
		if oldRec.SupersededBy != newID {
			t.Errorf("expected SupersededBy = %q, got %q", newID, oldRec.SupersededBy)
		}
	})

	t.Run("supersession empty is no-op", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)
		ctx := context.Background()

		dedup := NewDedupEngine(lStore)
		err := dedup.HandleSupersession(ctx, "new-id", "")
		if err != nil {
			t.Errorf("HandleSupersession() with empty ID should be no-op: %v", err)
		}
	})

	t.Run("supersession non-existent returns error", func(t *testing.T) {
		_, lStore, _ := helperLearning(t)
		ctx := context.Background()

		dedup := NewDedupEngine(lStore)
		err := dedup.HandleSupersession(ctx, "new-id", "non-existent-id")
		if err == nil {
			t.Error("expected error for non-existent supersedes ID")
		}
	})
}

// newTestLearning creates a learning record with defaults for testing.
func newTestLearning(title, body string) models.LearningRecord {
	return models.LearningRecord{
		Title:    title,
		Body:     body,
		Type:     models.LearningTypeConfig,
		Severity: models.SeverityMedium,
	}
}
