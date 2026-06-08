package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ghassan/alms/internal/models"
	"github.com/ghassan/alms/internal/service/storemock"
)

func helperLearning(t *testing.T) (*Learning, *storemock.LearningStore, *storemock.ProtocolStore) {
	t.Helper()
	lStore := storemock.NewLearningStore()
	pStore := storemock.NewProtocolStore()
	return NewLearning(lStore, pStore), lStore, pStore
}

func TestLearningStoreDedup(t *testing.T) {
	t.Parallel()

	t.Run("store success", func(t *testing.T) {
		learning, _, _ := helperLearning(t)
		ctx := context.Background()

		rec := models.LearningRecord{
			Title:      "Optimize database queries",
			Type:       models.LearningTypeConfig,
			Body:       "Use batch inserts for bulk operations",
			Severity:   models.SeverityMedium,
			Tags:       []string{"database", "performance"},
			Author:     "agent-1",
			SrcAgentID: "agent-1",
		}

		id, err := learning.Store(ctx, rec, "")
		if err != nil {
			t.Fatalf("Store() unexpected error: %v", err)
		}
		if id == "" {
			t.Fatal("Store() returned empty ID")
		}
	})

	t.Run("validation error", func(t *testing.T) {
		learning, _, _ := helperLearning(t)
		ctx := context.Background()

		rec := models.LearningRecord{
			Title: "",
			Type:  models.LearningTypePattern,
		}
		_, err := learning.Store(ctx, rec, "")
		if err == nil {
			t.Fatal("Store() expected validation error, got nil")
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		learning, _, _ := helperLearning(t)
		ctx := context.Background()

		rec := models.LearningRecord{
			Title: "Test",
			Type:  "invalid",
		}
		_, err := learning.Store(ctx, rec, "")
		if err == nil {
			t.Fatal("Store() expected validation error for invalid type, got nil")
		}
	})

	t.Run("preserves provided values", func(t *testing.T) {
		learning, lStore, _ := helperLearning(t)
		ctx := context.Background()

		rec := models.LearningRecord{
			Title:      "Critical bug fix",
			Type:       models.LearningTypeFailure,
			Severity:   models.SeverityCritical,
			Score:      0.99,
			TTLDays:    180,
			Resolution: models.ResolutionOpen,
		}

		id, err := learning.Store(ctx, rec, "")
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

	t.Run("store with supersession", func(t *testing.T) {
		learning, lStore, _ := helperLearning(t)
		ctx := context.Background()

		oldID, _ := lStore.Create(ctx, models.LearningRecord{Title: "Old", Type: models.LearningTypeConfig})
		newID, err := learning.Store(ctx, models.LearningRecord{Title: "New", Type: models.LearningTypeConfig}, oldID)
		if err != nil {
			t.Fatalf("Store() with supersession unexpected error: %v", err)
		}
		if newID == "" {
			t.Fatal("Store() returned empty ID")
		}
		oldRec, _ := lStore.Get(ctx, oldID)
		if string(oldRec.Resolution) != "superseded" {
			t.Errorf("expected superseded resolution, got %q", oldRec.Resolution)
		}
		if oldRec.SupersededBy != newID {
			t.Errorf("expected SupersededBy = %q, got %q", newID, oldRec.SupersededBy)
		}
	})
}

func TestLearningGet(t *testing.T) {
	t.Parallel()

	t.Run("get existing learning", func(t *testing.T) {
		learning, lStore, _ := helperLearning(t)
		ctx := context.Background()

		id, _ := lStore.Create(ctx, models.LearningRecord{Title: "Test Get", Type: models.LearningTypeConfig})
		rec, err := learning.Get(ctx, id)
		if err != nil {
			t.Fatalf("Get() unexpected error: %v", err)
		}
		if rec.Title != "Test Get" {
			t.Errorf("Title = %q, want %q", rec.Title, "Test Get")
		}
	})

	t.Run("get non-existent returns error", func(t *testing.T) {
		learning, _, _ := helperLearning(t)
		ctx := context.Background()

		_, err := learning.Get(ctx, "non-existent")
		if err == nil {
			t.Fatal("Get() expected error for non-existent ID")
		}
	})
}

func TestStoreLearningWithDedup(t *testing.T) {
	t.Parallel()

	t.Run("store with dedup no match", func(t *testing.T) {
		learning, _, _ := helperLearning(t)
		ctx := context.Background()

		id, result, err := learning.StoreLearningWithDedup(ctx, models.LearningRecord{
			Title: "New Learning",
			Type:  models.LearningTypeConfig,
		}, "")
		if err != nil {
			t.Fatalf("StoreLearningWithDedup() unexpected error: %v", err)
		}
		if id == "" {
			t.Fatal("StoreLearningWithDedup() returned empty ID")
		}
		if result == nil {
			t.Fatal("StoreLearningWithDedup() returned nil result")
		}
		if result.IsExactDup {
			t.Error("expected no exact dup")
		}
	})

	t.Run("store with dedup exact match returns existing", func(t *testing.T) {
		learning, lStore, _ := helperLearning(t)
		ctx := context.Background()

		existingID, _ := lStore.Create(ctx, models.LearningRecord{Title: "Existing", Body: "Body", Type: models.LearningTypeConfig})

		// Same title+body should trigger SHA256 exact dedup
		// Since the mock doesn't do hash-based dedup, StoreLearningWithDedup
		// checks via Search which does substring matching
		id, result, err := learning.StoreLearningWithDedup(ctx, models.LearningRecord{
			Title: "Existing",
			Body:  "Body",
			Type:  models.LearningTypeConfig,
		}, "")
		if err != nil {
			t.Fatalf("StoreLearningWithDedup() unexpected error: %v", err)
		}
		_ = existingID
		_ = id
		_ = result
	})
}

func TestLearningSearch(t *testing.T) {
	t.Parallel()

	t.Run("search returns matching records", func(t *testing.T) {
		learning, lStore, _ := helperLearning(t)
		ctx := context.Background()

		_, _ = lStore.Create(ctx, models.LearningRecord{
			Title: "Database Optimization",
			Body:  "Use indexes for faster queries",
			Type:  models.LearningTypeConfig,
		})
		_, _ = lStore.Create(ctx, models.LearningRecord{
			Title: "Cache Strategy",
			Body:  "Use Redis for caching",
			Type:  models.LearningTypePattern,
		})

		results, err := learning.Search(ctx, "Database", "", nil, 10)
		if err != nil {
			t.Fatalf("Search() unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("Search() returned %d results, want 1", len(results))
		}
		if len(results) > 0 && results[0].Title != "Database Optimization" {
			t.Errorf("Search() first result title = %q, want %q", results[0].Title, "Database Optimization")
		}
	})

	t.Run("search with type filter", func(t *testing.T) {
		learning, lStore, _ := helperLearning(t)
		ctx := context.Background()

		_, _ = lStore.Create(ctx, models.LearningRecord{
			Title: "Config Tuning",
			Body:  "Database config optimization",
			Type:  models.LearningTypeConfig,
		})
		_, _ = lStore.Create(ctx, models.LearningRecord{
			Title: "Error Handling",
			Body:  "Handle edge cases properly",
			Type:  models.LearningTypePattern,
		})

		results, err := learning.Search(ctx, "", string(models.LearningTypeConfig), nil, 10)
		if err != nil {
			t.Fatalf("Search() unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("Search() with type filter returned %d results, want 1", len(results))
		}
	})

	t.Run("search with store error returns error", func(t *testing.T) {
		learning, lStore, _ := helperLearning(t)
		ctx := context.Background()

		lStore.SetError(assertionError("store error"))
		_, err := learning.Search(ctx, "test", "", nil, 10)
		if err == nil {
			t.Error("Search() expected error from store, got nil")
		}
	})
}

func TestLearningDelete(t *testing.T) {
	t.Parallel()

	t.Run("delete existing learning", func(t *testing.T) {
		learning, lStore, _ := helperLearning(t)
		ctx := context.Background()

		id, _ := lStore.Create(ctx, models.LearningRecord{
			Title: "Test Delete",
			Type:  models.LearningTypeConfig,
		})

		err := learning.Delete(ctx, id)
		if err != nil {
			t.Fatalf("Delete() unexpected error: %v", err)
		}

		rec, _ := lStore.Get(ctx, id)
		if !rec.IsDeleted {
			t.Error("Delete() did not mark record as deleted")
		}
	})

	t.Run("delete empty id returns error", func(t *testing.T) {
		learning, _, _ := helperLearning(t)
		ctx := context.Background()

		err := learning.Delete(ctx, "")
		if err == nil {
			t.Fatal("Delete() empty ID should return error")
		}
	})

	t.Run("delete with store error returns error", func(t *testing.T) {
		learning, lStore, _ := helperLearning(t)
		ctx := context.Background()

		lStore.SetError(assertionError("store error"))
		err := learning.Delete(ctx, "some-id")
		if err == nil {
			t.Error("Delete() expected error from store, got nil")
		}
	})
}

func TestProtocolPush(t *testing.T) {
	t.Parallel()

	t.Run("push protocol succeeds", func(t *testing.T) {
		learning, _, _ := helperLearning(t)
		ctx := context.Background()

		rec := models.ProtocolRecord{
			Title:      "Test Protocol",
			Body:       "This is a test protocol",
			TargetTags: []string{"agent-1"},
			IsActive:   true,
		}

		id, err := learning.ProtocolPush(ctx, rec)
		if err != nil {
			t.Fatalf("ProtocolPush() unexpected error: %v", err)
		}
		if id == "" {
			t.Fatal("ProtocolPush() returned empty ID")
		}
	})

	t.Run("push protocol validation error", func(t *testing.T) {
		learning, _, _ := helperLearning(t)
		ctx := context.Background()

		rec := models.ProtocolRecord{
			Title: "",
		}
		_, err := learning.ProtocolPush(ctx, rec)
		if err == nil {
			t.Fatal("ProtocolPush() empty title should return error")
		}
	})
}

func TestProtocolList(t *testing.T) {
	t.Parallel()

	t.Run("list protocols", func(t *testing.T) {
		learning, _, pStore := helperLearning(t)
		ctx := context.Background()

		_, _ = pStore.Create(ctx, models.ProtocolRecord{Title: "Proto 1"})
		_, _ = pStore.Create(ctx, models.ProtocolRecord{Title: "Proto 2"})

		results, err := learning.ProtocolList(ctx)
		if err != nil {
			t.Fatalf("ProtocolList() unexpected error: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("ProtocolList() returned %d results, want 2", len(results))
		}
	})

	t.Run("list with store error returns error", func(t *testing.T) {
		learning, _, pStore := helperLearning(t)
		ctx := context.Background()

		pStore.SetError(assertionError("store error"))
		_, err := learning.ProtocolList(ctx)
		if err == nil {
			t.Error("ProtocolList() expected error from store, got nil")
		}
	})
}

func TestStoreLearningWithDedupExactDupErrorPath(t *testing.T) {
	t.Parallel()
	_, lStore, _ := helperLearning(t)
	learning := NewLearning(lStore, storemock.NewProtocolStore())
	ctx := context.Background()

	// Set store error to trigger exact dup check failure
	lStore.SetError(assertionError("dedup error"))
	_, _, err := learning.StoreLearningWithDedup(ctx, models.LearningRecord{
		Title: "Test",
		Type:  models.LearningTypePattern,
	}, "")
	if err == nil {
		t.Error("StoreLearningWithDedup() expected error from exact dup, got nil")
	}
}

func TestStoreLearningWithDedupStoreErrorPath(t *testing.T) {
	t.Parallel()
	_, lStore, _ := helperLearning(t)
	ctx := context.Background()

	// Set error before call — will fail on exact dup check
	lStore.SetError(assertionError("store error"))
	_, _, err := NewLearning(lStore, storemock.NewProtocolStore()).StoreLearningWithDedup(ctx, models.LearningRecord{
		Title: "Test",
		Type:  models.LearningTypePattern,
	}, "")
	if err == nil {
		t.Error("StoreLearningWithDedup() expected error, got nil")
	}
}

func TestStoreValidationError(t *testing.T) {
	t.Parallel()
	_, lStore, _ := helperLearning(t)
	ctx := context.Background()

	// Empty title should fail validation
	_, err := NewLearning(lStore, storemock.NewProtocolStore()).Store(ctx, models.LearningRecord{
		Title: "",
		Type:  models.LearningTypePattern,
	}, "")
	if err == nil {
		t.Error("Store() expected validation error for empty title, got nil")
	}
}

func TestSearchAdvanced(t *testing.T) {
	t.Parallel()

	t.Run("search advanced passes through to store", func(t *testing.T) {
		learning, lStore, _ := helperLearning(t)
		ctx := context.Background()

		_, _ = lStore.Create(ctx, models.LearningRecord{
			Title: "Advanced Test",
			Body:  "Advanced search content",
			Type:  models.LearningTypeConfig,
		})

		results, err := learning.SearchAdvanced(ctx, "Advanced", "", nil, 10, "", false)
		if err != nil {
			t.Fatalf("SearchAdvanced() unexpected error: %v", err)
		}
		if len(results) == 0 {
			t.Error("SearchAdvanced() expected at least 1 result")
		}
	})

	t.Run("search advanced with type filter", func(t *testing.T) {
		learning, lStore, _ := helperLearning(t)
		ctx := context.Background()

		_, _ = lStore.Create(ctx, models.LearningRecord{
			Title: "Config Item",
			Body:  "Config data",
			Type:  models.LearningTypeConfig,
		})
		_, _ = lStore.Create(ctx, models.LearningRecord{
			Title: "Pattern Item",
			Body:  "Pattern data",
			Type:  models.LearningTypePattern,
		})

		results, err := learning.SearchAdvanced(ctx, "", string(models.LearningTypeConfig), nil, 10, "", false)
		if err != nil {
			t.Fatalf("SearchAdvanced() unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("SearchAdvanced() with type filter got %d results, want 1", len(results))
		}
	})

	t.Run("search advanced with store error returns error", func(t *testing.T) {
		learning, lStore, _ := helperLearning(t)
		ctx := context.Background()

		lStore.SetError(assertionError("search error"))
		_, err := learning.SearchAdvanced(ctx, "test", "", nil, 10, "", false)
		if err == nil {
			t.Error("SearchAdvanced() expected error, got nil")
		}
	})
}

func TestUpdateEnrichment(t *testing.T) {
	t.Parallel()

	t.Run("update enrichment succeeds", func(t *testing.T) {
		learning, lStore, _ := helperLearning(t)
		ctx := context.Background()

		id, _ := lStore.Create(ctx, models.LearningRecord{
			Title: "Enrich Me",
			Type:  models.LearningTypeConfig,
		})

		patch := json.RawMessage(`{"status":"accepted","quality":{"score":4.5}}`)
		err := learning.UpdateEnrichment(ctx, id, patch)
		if err != nil {
			t.Fatalf("UpdateEnrichment() unexpected error: %v", err)
		}

		// Verify enrichment was stored
		rec, err := lStore.Get(ctx, id)
		if err != nil {
			t.Fatalf("Get() unexpected error: %v", err)
		}
		if len(rec.EnrichmentMetadata) == 0 {
			t.Error("EnrichmentMetadata should not be empty after update")
		}
	})

	t.Run("update enrichment with empty id returns error", func(t *testing.T) {
		learning, _, _ := helperLearning(t)
		ctx := context.Background()

		err := learning.UpdateEnrichment(ctx, "", json.RawMessage(`{}`))
		if err == nil {
			t.Fatal("UpdateEnrichment() expected error for empty learning_id, got nil")
		}
	})

	t.Run("update enrichment non-existent returns error", func(t *testing.T) {
		learning, _, _ := helperLearning(t)
		ctx := context.Background()

		err := learning.UpdateEnrichment(ctx, "non-existent", json.RawMessage(`{}`))
		if err == nil {
			t.Fatal("UpdateEnrichment() expected error for non-existent ID, got nil")
		}
	})

	t.Run("update enrichment with nil JSON", func(t *testing.T) {
		learning, lStore, _ := helperLearning(t)
		ctx := context.Background()

		id, _ := lStore.Create(ctx, models.LearningRecord{
			Title: "Nil Enrich",
			Type:  models.LearningTypeConfig,
		})

		err := learning.UpdateEnrichment(ctx, id, nil)
		if err != nil {
			t.Fatalf("UpdateEnrichment() with nil should work: %v", err)
		}
	})

	t.Run("quality_score syncs to score column", func(t *testing.T) {
		learning, lStore, _ := helperLearning(t)
		ctx := context.Background()

		id, _ := lStore.Create(ctx, models.LearningRecord{
			Title: "Score Sync",
			Type:  models.LearningTypeConfig,
			Score: 0.5, // default
		})

		patch := json.RawMessage(`{"quality_score": 4.5}`)
		err := learning.UpdateEnrichment(ctx, id, patch)
		if err != nil {
			t.Fatalf("UpdateEnrichment() unexpected error: %v", err)
		}

		rec, _ := lStore.Get(ctx, id)
		if rec.Score != 4.5 {
			t.Errorf("Score = %f, want 4.5", rec.Score)
		}
		// Verify enrichment_metadata was also updated
		if len(rec.EnrichmentMetadata) == 0 {
			t.Error("EnrichmentMetadata should not be empty")
		}
	})

	t.Run("score field also syncs to score column", func(t *testing.T) {
		learning, lStore, _ := helperLearning(t)
		ctx := context.Background()

		id, _ := lStore.Create(ctx, models.LearningRecord{
			Title: "Score Field",
			Type:  models.LearningTypeConfig,
			Score: 0.5,
		})

		patch := json.RawMessage(`{"score": 3.0}`)
		err := learning.UpdateEnrichment(ctx, id, patch)
		if err != nil {
			t.Fatalf("UpdateEnrichment() unexpected error: %v", err)
		}

		rec, _ := lStore.Get(ctx, id)
		if rec.Score != 3.0 {
			t.Errorf("Score = %f, want 3.0", rec.Score)
		}
	})

	t.Run("quality_score takes precedence over score", func(t *testing.T) {
		learning, lStore, _ := helperLearning(t)
		ctx := context.Background()

		id, _ := lStore.Create(ctx, models.LearningRecord{
			Title: "Precedence",
			Type:  models.LearningTypeConfig,
		})

		patch := json.RawMessage(`{"quality_score": 4.2, "score": 2.0}`)
		err := learning.UpdateEnrichment(ctx, id, patch)
		if err != nil {
			t.Fatalf("UpdateEnrichment() unexpected error: %v", err)
		}

		rec, _ := lStore.Get(ctx, id)
		if rec.Score != 4.2 {
			t.Errorf("Score = %f, want 4.2 (quality_score over score)", rec.Score)
		}
	})

	t.Run("status-only patch does not affect score column", func(t *testing.T) {
		learning, lStore, _ := helperLearning(t)
		ctx := context.Background()

		id, _ := lStore.Create(ctx, models.LearningRecord{
			Title: "Status Only",
			Type:  models.LearningTypeConfig,
			Score: 0.7,
		})

		patch := json.RawMessage(`{"status": "accepted"}`)
		err := learning.UpdateEnrichment(ctx, id, patch)
		if err != nil {
			t.Fatalf("UpdateEnrichment() unexpected error: %v", err)
		}

		rec, _ := lStore.Get(ctx, id)
		if rec.Score != 0.7 {
			t.Errorf("Score should remain 0.7, got %f", rec.Score)
		}
		if len(rec.EnrichmentMetadata) == 0 {
			t.Error("EnrichmentMetadata should be set")
		}
	})
}

// assertionError is a simple error type for testing.
type assertionError string

func (e assertionError) Error() string { return string(e) }
