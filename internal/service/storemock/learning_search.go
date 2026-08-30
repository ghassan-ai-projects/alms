package storemock

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ghassan/alms/internal/models"
)

// Search performs search on stored records (simple substring match).
func (m *LearningStore) Search(ctx context.Context, query, ltype string, tags []string, limit int) ([]models.LearningRecord, error) {
	if err := m.getErr(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []models.LearningRecord
	for _, rec := range m.records {
		if rec.IsDeleted {
			continue
		}
		if !containsSubstring(rec.Title, query) && !containsSubstring(rec.Body, query) {
			continue
		}
		if ltype != "" && string(rec.Type) != ltype {
			continue
		}
		if len(tags) > 0 && !tagsOverlap(rec.Tags, tags) {
			continue
		}
		result = append(result, *rec)
	}
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

// SearchWithStatus searches with additional filter params for status and includeRejected.
func (m *LearningStore) SearchWithStatus(ctx context.Context, query string, ltype string, tags []string, limit int, status string, includeRejected bool) ([]models.LearningRecord, error) {
	if err := m.getErr(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []models.LearningRecord
	for _, rec := range m.records {
		if rec.IsDeleted {
			continue
		}
		if !containsSubstring(rec.Title, query) && !containsSubstring(rec.Body, query) {
			continue
		}
		if ltype != "" && string(rec.Type) != ltype {
			continue
		}
		if len(tags) > 0 && !tagsOverlap(rec.Tags, tags) {
			continue
		}

		enrichment := models.NormalizeEnrichmentMetadata(rec.EnrichmentMetadata)
		var meta map[string]any
		if err := json.Unmarshal(enrichment, &meta); err != nil {
			return nil, fmt.Errorf("parse enrichment metadata: %w", err)
		}

		recStatus, _ := meta["status"].(string)
		if status != "" && recStatus != status {
			continue
		}

		if !includeRejected {
			if visible, ok := meta["is_visible"].(bool); ok && !visible {
				continue
			}
		}

		cp := *rec
		cp.EnrichmentMetadata = enrichment
		result = append(result, cp)
	}
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}
