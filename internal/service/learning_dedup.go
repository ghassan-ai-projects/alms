package service

import (
	"context"
	"fmt"

	"github.com/ghassan/alms/internal/models"
)

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
