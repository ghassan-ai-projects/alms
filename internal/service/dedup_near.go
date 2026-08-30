package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/ghassan/alms/internal/models"
)

// calculateLevenshteinDistance computes the Levenshtein distance between two strings.
func calculateLevenshteinDistance(a, b string) int {
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
			curr[j] = minimumOfThree(
				prev[j]+1,
				curr[j-1]+1,
				prev[j-1]+cost,
			)
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func minimumOfThree(a, b, c int) int {
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
	dist := calculateLevenshteinDistance(a, b)
	return 1.0 - float64(dist)/float64(maxLen)
}

// CheckNearDup checks if any existing learning has a title with Levenshtein ratio >= 0.85.
func (d *DedupEngine) CheckNearDup(ctx context.Context, title string, excludeIDs []string) (*DedupResult, error) {
	const threshold = 0.85

	records, err := d.store.Search(ctx, "", "", nil, 100)
	if err != nil {
		return nil, fmt.Errorf("near dedup search: %w", err)
	}

	excludedLearningIDs := buildExcludedLearningIDs(excludeIDs)
	similarityByLearningID := collectNearDuplicateRatios(records, title, excludedLearningIDs, threshold)
	if len(similarityByLearningID) == 0 {
		return &DedupResult{}, nil
	}

	return &DedupResult{
		IsNearDup:    true,
		NearMatchIDs: sortLearningIDsBySimilarity(similarityByLearningID),
	}, nil
}

func buildExcludedLearningIDs(ids []string) map[string]bool {
	excluded := make(map[string]bool, len(ids))
	for _, id := range ids {
		excluded[id] = true
	}
	return excluded
}

func collectNearDuplicateRatios(records []models.LearningRecord, title string, excluded map[string]bool, threshold float64) map[string]float64 {
	byRatio := make(map[string]float64)
	for _, record := range records {
		if excluded[record.LearningID] || record.IsDeleted {
			continue
		}
		ratio := LevenshteinRatio(title, record.Title)
		if ratio >= threshold {
			byRatio[record.LearningID] = ratio
		}
	}
	return byRatio
}

func sortLearningIDsBySimilarity(similarityByLearningID map[string]float64) []string {
	ids := make([]string, 0, len(similarityByLearningID))
	for id := range similarityByLearningID {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return similarityByLearningID[ids[i]] > similarityByLearningID[ids[j]]
	})
	return ids
}
