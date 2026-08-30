// Package service provides business logic for ALMS operations.
package service

// ScoringEngine manages learning record score operations.
type ScoringEngine struct {
	store LearningStore
}

// NewScoringEngine creates a new ScoringEngine backed by the given store.
func NewScoringEngine(store LearningStore) *ScoringEngine {
	return &ScoringEngine{store: store}
}

const (
	// ScoreIncrementSync is the score added each time a learning is successfully synced.
	ScoreIncrementSync = 0.1
	// ScoreDecrementTTL is the score decremented per TTL day without update.
	ScoreDecrementTTL = 0.1
	// ScoreMin is the minimum score allowed.
	ScoreMin = 0.0
	// ScoreMax is the maximum score allowed.
	ScoreMax = 1.0
	// ScoreDefault is the default score for new learnings.
	ScoreDefault = 0.5
)
