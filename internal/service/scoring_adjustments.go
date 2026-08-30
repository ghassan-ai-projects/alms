package service

import (
	"context"
	"fmt"

	"github.com/ghassan/alms/internal/models"
)

// IncrementScore increases the score for a learning record by the sync increment.
// Pinned learnings are immune to scoring changes.
func (s *ScoringEngine) IncrementScore(ctx context.Context, learningID string) error {
	record, err := s.loadLearningForScoring(ctx, learningID, "increment score")
	if err != nil {
		return err
	}
	if record.IsPinned {
		return nil
	}
	if err := rejectDeletedLearning(record, learningID); err != nil {
		return err
	}
	return s.persistScoreChange(ctx, learningID, increaseScore(record.Score), "increment score")
}

// DecrementScore decreases the score for a learning record by the given amount.
// Pinned learnings are immune to scoring changes.
func (s *ScoringEngine) DecrementScore(ctx context.Context, learningID string, amount float64) error {
	record, err := s.loadLearningForScoring(ctx, learningID, "decrement score")
	if err != nil {
		return err
	}
	if record.IsPinned {
		return nil
	}
	if err := rejectDeletedLearning(record, learningID); err != nil {
		return err
	}
	return s.persistScoreChange(ctx, learningID, decreaseScore(record.Score, amount), "decrement score")
}

// GetScore returns the current score of a learning record.
func (s *ScoringEngine) GetScore(ctx context.Context, learningID string) (float64, error) {
	record, err := s.store.Get(ctx, learningID)
	if err != nil {
		return 0, fmt.Errorf("get score: %w", err)
	}
	return record.Score, nil
}

func (s *ScoringEngine) loadLearningForScoring(ctx context.Context, learningID, operation string) (models.LearningRecord, error) {
	record, err := s.store.Get(ctx, learningID)
	if err != nil {
		return record, fmt.Errorf("%s: %w", operation, err)
	}
	return record, nil
}

func rejectDeletedLearning(record models.LearningRecord, learningID string) error {
	if record.IsDeleted {
		return fmt.Errorf("%w: learning %s is deleted", models.ErrValidation, learningID)
	}
	return nil
}

func increaseScore(score float64) float64 {
	newScore := score + ScoreIncrementSync
	if newScore > ScoreMax {
		return ScoreMax
	}
	return newScore
}

func decreaseScore(score, amount float64) float64 {
	newScore := score - amount
	if newScore < ScoreMin {
		return ScoreMin
	}
	return newScore
}

func (s *ScoringEngine) persistScoreChange(ctx context.Context, learningID string, score float64, operation string) error {
	if err := s.store.UpdateScore(ctx, learningID, score); err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	return nil
}
