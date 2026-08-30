package service

import (
	"context"
	"fmt"

	"github.com/ghassan/alms/internal/models"
)

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

	if learningID, found := findExactDuplicateID(records, hash); found {
		return &DedupResult{IsExactDup: true, ExactMatchID: learningID}, nil
	}
	return &DedupResult{}, nil
}

func findExactDuplicateID(records []models.LearningRecord, hash string) (string, bool) {
	for _, record := range records {
		if SHA256Hash(record.Title, record.Body) == hash {
			return record.LearningID, true
		}
	}
	return "", false
}
