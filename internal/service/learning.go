package service

import (
	"context"
	"encoding/json"
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

	record = prepareLearningRecord(record)
	id, err := l.lStore.Create(ctx, record)
	if err != nil {
		return "", fmt.Errorf("store learning: %w", err)
	}

	if err := l.handleLearningSupersession(ctx, id, supersedes); err != nil {
		return id, fmt.Errorf("store learning supersession: %w", err)
	}
	return id, nil
}

func prepareLearningRecord(record models.LearningRecord) models.LearningRecord {
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
	record.EnrichmentMetadata = models.NormalizeEnrichmentMetadata(record.EnrichmentMetadata)
	record.CreatedAt = time.Now()
	return record
}

func (l *Learning) handleLearningSupersession(ctx context.Context, learningID, supersedesID string) error {
	if supersedesID == "" {
		return nil
	}
	dedup := NewDedupEngine(l.lStore)
	return dedup.HandleSupersession(ctx, learningID, supersedesID)
}

// Search performs full-text search on learnings.
func (l *Learning) Search(ctx context.Context, query string, ltype string, tags []string, limit int) ([]models.LearningRecord, error) {
	records, err := l.lStore.Search(ctx, query, ltype, tags, limit)
	if err != nil {
		return nil, fmt.Errorf("search learnings: %w", err)
	}
	return records, nil
}

// SearchAdvanced performs full-text search with status filter and includeRejected flag.
func (l *Learning) SearchAdvanced(ctx context.Context, query string, ltype string, tags []string, limit int, status string, includeRejected bool) ([]models.LearningRecord, error) {
	records, err := l.lStore.SearchWithStatus(ctx, query, ltype, tags, limit, status, includeRejected)
	if err != nil {
		return nil, fmt.Errorf("search advanced: %w", err)
	}
	return records, nil
}

// UpdateEnrichment merges enrichment metadata for a learning.
// If the enrichment JSON contains a "quality_score" or "score" field,
// the top-level score column is also updated in the same call.
func (l *Learning) UpdateEnrichment(ctx context.Context, learningID string, enrichmentJSON json.RawMessage) error {
	if err := validateLearningID(learningID); err != nil {
		return err
	}
	if err := l.lStore.UpdateEnrichment(ctx, learningID, enrichmentJSON); err != nil {
		return fmt.Errorf("update enrichment: %w", err)
	}
	return l.synchronizeEnrichmentScore(ctx, learningID, enrichmentJSON)
}

func validateLearningID(learningID string) error {
	if learningID == "" {
		return fmt.Errorf("%w: learning_id is required", models.ErrValidation)
	}
	return nil
}

func (l *Learning) synchronizeEnrichmentScore(ctx context.Context, learningID string, enrichmentJSON json.RawMessage) error {
	score, err := extractScoreFromEnrichment(enrichmentJSON)
	if err != nil {
		return nil
	}
	if err := l.lStore.UpdateScore(ctx, learningID, score); err != nil {
		return fmt.Errorf("sync score from enrichment: %w", err)
	}
	return nil
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
	if err := validateLearningID(learningID); err != nil {
		return err
	}
	if err := l.lStore.SoftDelete(ctx, learningID); err != nil {
		return fmt.Errorf("delete learning: %w", err)
	}
	return nil
}
