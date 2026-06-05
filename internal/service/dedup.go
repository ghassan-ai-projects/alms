// Package service provides business logic for ALMS operations.
package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/ghassan/alms/internal/models"
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
	IsExactDup   bool   // true if SHA256 hash matches an existing record
	ExactMatchID string // learning_id of the exact match (if IsExactDup)
	IsNearDup    bool   // true if Levenshtein ratio >= 0.85
	NearMatchIDs []string // learning_ids of near matches (if IsNearDup)
}

// SHA256Hash computes the SHA256 hex of title + body for dedup comparison.
// This matches the hash the store checks during insert.
func SHA256Hash(title, body string) string {
	h := sha256.Sum256([]byte(title + body))
	return fmt.Sprintf("%x", h)
}

// CheckExactDup checks if a learning with the same SHA256 hash already exists.
// It does this by listing learnings and comparing hashes (the store layer can
// also enforce this with an ON CONFLICT DO NOTHING clause).
func (d *DedupEngine) CheckExactDup(ctx context.Context, title, body string) (*DedupResult, error) {
	hash := SHA256Hash(title, body)

	// Search for existing records (get all learnings with matching title+body)
	// We use a simple approach: list recent learnings and check hash.
	// A production version would store the hash in the DB and query it.
	// For now, we rely on the store's Create returning an error on conflict,
	// and we do a best-effort check here.
	records, err := d.store.Search(ctx, title, "", nil, 50)
	if err != nil {
		return nil, fmt.Errorf("exact dedup search: %w", err)
	}

	result := &DedupResult{}
	for _, rec := range records {
		if SHA256Hash(rec.Title, rec.Body) == hash {
			result.IsExactDup = true
			result.ExactMatchID = rec.LearningID
			break
		}
	}

	return result, nil
}

// levenshtein computes the Levenshtein distance between two strings.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	// Use single row optimization for smaller allocations
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min3(
				prev[j]+1,
				curr[j-1]+1,
				prev[j-1]+cost,
			)
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// LevenshteinRatio returns the similarity ratio between two strings (1.0 = identical, 0.0 = completely different).
func LevenshteinRatio(a, b string) float64 {
	if a == "" && b == "" {
		return 1.0
	}
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	if maxLen == 0 {
		return 1.0
	}
	dist := levenshtein(a, b)
	return 1.0 - float64(dist)/float64(maxLen)
}

// CheckNearDup checks if any existing learning has a title with Levenshtein ratio >= 0.85.
func (d *DedupEngine) CheckNearDup(ctx context.Context, title string, excludeIDs []string) (*DedupResult, error) {
	const threshold = 0.85

	records, err := d.store.Search(ctx, "", "", nil, 100)
	if err != nil {
		return nil, fmt.Errorf("near dedup search: %w", err)
	}

	excludeSet := make(map[string]bool, len(excludeIDs))
	for _, id := range excludeIDs {
		excludeSet[id] = true
	}

	result := &DedupResult{}
	byRatio := make(map[string]float64)

	for _, rec := range records {
		if excludeSet[rec.LearningID] {
			continue
		}
		if rec.IsDeleted {
			continue
		}
		ratio := LevenshteinRatio(title, rec.Title)
		if ratio >= threshold {
			result.IsNearDup = true
			byRatio[rec.LearningID] = ratio
		}
	}

	if result.IsNearDup {
		// Sort by ratio descending for predictable output
		ids := make([]string, 0, len(byRatio))
		for id := range byRatio {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool {
			return byRatio[ids[i]] > byRatio[ids[j]]
		})
		result.NearMatchIDs = ids
	}

	return result, nil
}

// HandleSupersession processes the supersedes parameter during learning storage.
// It validates the supersedes relationship and updates the superseded learning.
func (d *DedupEngine) HandleSupersession(ctx context.Context, newID, supersedesID string) error {
	if supersedesID == "" {
		return nil
	}

	// Validate the superseded learning exists
	existing, err := d.store.Get(ctx, supersedesID)
	if err != nil {
		return fmt.Errorf("supersession check: %w", err)
	}
	if existing.IsDeleted {
		return fmt.Errorf("%w: superseded learning %s is deleted", models.ErrValidation, supersedesID)
	}

	// Update the superseded learning's resolution
	if err := d.store.Supersede(ctx, supersedesID, newID); err != nil {
		return fmt.Errorf("supersession update: %w", err)
	}

	return nil
}


