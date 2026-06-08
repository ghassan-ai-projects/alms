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
	if learningID == "" {
		return fmt.Errorf("%w: learning_id is required", models.ErrValidation)
	}
	if err := l.lStore.UpdateEnrichment(ctx, learningID, enrichmentJSON); err != nil {
		return fmt.Errorf("update enrichment: %w", err)
	}

	// Extract score from enrichment metadata and sync to top-level score column
	if score, err := extractScoreFromEnrichment(enrichmentJSON); err == nil {
		// err != nil means the field is missing or not a float — that's fine, skip score update
		if err := l.lStore.UpdateScore(ctx, learningID, score); err != nil {
			return fmt.Errorf("sync score from enrichment: %w", err)
		}
	}

	return nil
}

// extractScoreFromEnrichment extracts "quality_score" or "score" (taking
// quality_score first) from a JSON enrichment patch. Returns an error if
// neither field is present or if the value is not a float64.
func extractScoreFromEnrichment(data json.RawMessage) (float64, error) {
	if len(data) == 0 {
		return 0, fmt.Errorf("empty enrichment data")
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return 0, fmt.Errorf("parse enrichment: %w", err)
	}

	// Try quality_score first (it's more specific)
	if v, ok := m["quality_score"]; ok {
		score, ok := v.(float64)
		if !ok {
			return 0, fmt.Errorf("quality_score is not a number")
		}
		return score, nil
	}

	// Fall back to score
	if v, ok := m["score"]; ok {
		score, ok := v.(float64)
		if !ok {
			return 0, fmt.Errorf("score is not a number")
		}
		return score, nil
	}

	return 0, fmt.Errorf("no score field in enrichment")
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
