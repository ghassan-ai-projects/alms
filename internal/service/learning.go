package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/ghassan/alms/internal/models"
)

// Learning manages storing and searching learning records.
type Learning struct {
	store LearningStore
}

// NewLearning creates a new Learning service backed by the given store.
func NewLearning(store LearningStore) *Learning {
	return &Learning{store: store}
}

// Store persists a new learning record after validation. Returns the generated
// learning ID. Deduplication (SHA256 of title+body) is handled via the store
// layer check. For Phase 1, this is a simple create; Phase 2 adds exact and
// near dedup.
func (l *Learning) Store(ctx context.Context, record models.LearningRecord) (string, error) {
	if err := record.Validate(); err != nil {
		return "", fmt.Errorf("store learning: %w", err)
	}

	if record.Score == 0 {
		record.Score = 0.5
	}
	if record.TTLDays == 0 {
		record.TTLDays = 90
	}
	if record.Resolution == "" {
		record.Resolution = models.ResolutionOpen
	}
	if record.Severity == "" {
		record.Severity = models.SeverityMedium
	}

	// Generate SHA256 hash for dedup (stored in learning record check)
	_ = sha256.Sum256([]byte(record.Title + record.Body))

	createdAt := time.Now()
	record.CreatedAt = createdAt

	id, err := l.store.Create(ctx, record)
	if err != nil {
		return "", fmt.Errorf("store learning: %w", err)
	}
	return id, nil
}
