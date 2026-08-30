// Package service provides business logic for ALMS operations.
package service

import (
	"crypto/sha256"
	"fmt"
)

// DedupEngine provides exact and near-duplicate detection for learning records.
type DedupEngine struct {
	store LearningStore
}

// NewDedupEngine creates a new DedupEngine backed by the given store.
func NewDedupEngine(store LearningStore) *DedupEngine {
	return &DedupEngine{store: store}
}

// DedupResult describes the outcome of a dedup check.
type DedupResult struct {
	IsExactDup   bool     // true if SHA256 hash matches an existing record
	ExactMatchID string   // learning_id of the exact match (if IsExactDup)
	IsNearDup    bool     // true if Levenshtein ratio >= 0.85
	NearMatchIDs []string // learning_ids of near matches (if IsNearDup)
}

// SHA256Hash computes the SHA256 hex of title + body for dedup comparison.
// This matches the hash the store checks during insert.
func SHA256Hash(title, body string) string {
	h := sha256.Sum256([]byte(title + body))
	return fmt.Sprintf("%x", h)
}
