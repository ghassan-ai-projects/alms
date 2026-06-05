package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ghassan/alms/internal/models"
)

// Learning manages storing, searching, and managing learning records and protocols.
type Learning struct {
	lStore LearningStore
	pStore ProtocolStore
}

// NewLearning creates a new Learning service backed by the given stores.
func NewLearning(lStore LearningStore, pStore ProtocolStore) *Learning {
	return &Learning{
		lStore: lStore,
		pStore: pStore,
	}
}

// Store persists a new learning record after validation and dedup checking.
// Returns the generated learning ID. If supersedes is non-empty, the referenced
// learning is marked as superseded.
func (l *Learning) Store(ctx context.Context, record models.LearningRecord, supersedes string) (string, error) {
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

	record.CreatedAt = time.Now()

	id, err := l.lStore.Create(ctx, record)
	if err != nil {
		return "", fmt.Errorf("store learning: %w", err)
	}

	// Handle supersession
	if supersedes != "" {
		dedup := NewDedupEngine(l.lStore)
		if err := dedup.HandleSupersession(ctx, id, supersedes); err != nil {
			return id, fmt.Errorf("store learning supersession: %w", err)
		}
	}

	return id, nil
}

// StoreLearningWithDedup stores a learning after checking for exact and near duplicates.
// Returns the result including any dedup findings.
func (l *Learning) StoreLearningWithDedup(ctx context.Context, record models.LearningRecord, supersedes string) (string, *DedupResult, error) {
	dedup := NewDedupEngine(l.lStore)

	// Check exact dup first
	exactResult, err := dedup.CheckExactDup(ctx, record.Title, record.Body)
	if err != nil {
		return "", nil, fmt.Errorf("dedup check: %w", err)
	}
	if exactResult.IsExactDup {
		return exactResult.ExactMatchID, exactResult, nil
	}

	// Check near dup
	nearResult, err := dedup.CheckNearDup(ctx, record.Title, nil)
	if err != nil {
		return "", nil, fmt.Errorf("near dedup check: %w", err)
	}
	if nearResult.IsNearDup {
		// Store the learning but flag it
		id, err := l.Store(ctx, record, supersedes)
		if err != nil {
			return "", nil, err
		}
		return id, nearResult, nil
	}

	id, err := l.Store(ctx, record, supersedes)
	if err != nil {
		return "", nil, err
	}

	return id, &DedupResult{}, nil
}

// Search performs full-text search on learnings.
func (l *Learning) Search(ctx context.Context, query string, ltype string, tags []string, limit int) ([]models.LearningRecord, error) {
	records, err := l.lStore.Search(ctx, query, ltype, tags, limit)
	if err != nil {
		return nil, fmt.Errorf("search learnings: %w", err)
	}
	return records, nil
}

// Get retrieves a single learning record by ID.
func (l *Learning) Get(ctx context.Context, learningID string) (models.LearningRecord, error) {
	record, err := l.lStore.Get(ctx, learningID)
	if err != nil {
		return record, fmt.Errorf("get learning: %w", err)
	}
	return record, nil
}

// Delete soft-deletes a learning record.
func (l *Learning) Delete(ctx context.Context, learningID string) error {
	if learningID == "" {
		return fmt.Errorf("%w: learning_id is required", models.ErrValidation)
	}
	if err := l.lStore.SoftDelete(ctx, learningID); err != nil {
		return fmt.Errorf("delete learning: %w", err)
	}
	return nil
}

// ProtocolPush creates a new protocol record.
func (l *Learning) ProtocolPush(ctx context.Context, record models.ProtocolRecord) (string, error) {
	if err := record.Validate(); err != nil {
		return "", fmt.Errorf("protocol push: %w", err)
	}
	id, err := l.pStore.Create(ctx, record)
	if err != nil {
		return "", fmt.Errorf("protocol push: %w", err)
	}
	return id, nil
}

// ProtocolList returns all protocol records.
func (l *Learning) ProtocolList(ctx context.Context) ([]models.ProtocolRecord, error) {
	protocols, err := l.pStore.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list protocols: %w", err)
	}
	return protocols, nil
}
