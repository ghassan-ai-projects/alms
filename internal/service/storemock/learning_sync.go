package storemock

import (
	"context"
	"sort"
	"time"

	"github.com/ghassan/alms/internal/models"
)

// Sync returns learnings after the given timestamp, optionally filtered.
func (m *LearningStore) Sync(ctx context.Context, agentID string, since time.Time, ltype string, tags []string) ([]models.LearningRecord, error) {
	if err := m.getErr(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []models.LearningRecord
	for _, rec := range m.records {
		if !rec.CreatedAt.After(since) || rec.IsDeleted {
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
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result, nil
}

// SyncAck stores acknowledgements and advances the agent cursor.
func (m *LearningStore) SyncAck(ctx context.Context, agentID string, learningIDs []string) error {
	if err := m.getErr(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	m.acks[agentID] = append(m.acks[agentID], learningIDs...)

	// Advance cursor to latest acknowledged learning's timestamp
	if len(learningIDs) > 0 {
		lastID := learningIDs[len(learningIDs)-1]
		if rec, ok := m.records[lastID]; ok {
			m.agentCursors[agentID] = rec.CreatedAt
		}
	}
	return nil
}

// ExpectedSyncIDs returns the ordered IDs of learnings after the given time.
func (m *LearningStore) ExpectedSyncIDs(ctx context.Context, agentID string, since time.Time) ([]string, error) {
	if err := m.getErr(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var ids []string
	var records []*models.LearningRecord
	for _, rec := range m.records {
		if rec.CreatedAt.After(since) && !rec.IsDeleted {
			records = append(records, rec)
		}
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt.Before(records[j].CreatedAt)
	})
	for _, rec := range records {
		ids = append(ids, rec.LearningID)
	}
	return ids, nil
}

// GetAcks returns the acknowledged IDs for an agent (for test assertions).
func (m *LearningStore) GetAcks(agentID string) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	acks := m.acks[agentID]
	result := make([]string, len(acks))
	copy(result, acks)
	return result
}

// GetAgentCursor returns the agent's sync cursor (for test assertions).
func (m *LearningStore) GetAgentCursor(agentID string) time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.agentCursors[agentID]
}
