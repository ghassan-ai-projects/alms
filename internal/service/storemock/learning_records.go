package storemock

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ghassan/alms/internal/models"
)

// LearningStore is an in-memory mock of service.LearningStore.
type LearningStore struct {
	mu           sync.Mutex
	records      map[string]*models.LearningRecord
	acks         map[string][]string // agentID -> []learningID
	agentCursors map[string]time.Time
	err          error
}

// NewLearningStore creates an empty LearningStore mock.
func NewLearningStore() *LearningStore {
	return &LearningStore{
		records:      make(map[string]*models.LearningRecord),
		acks:         make(map[string][]string),
		agentCursors: make(map[string]time.Time),
	}
}

// SetError configures an error to be returned on all subsequent calls.
func (m *LearningStore) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

func (m *LearningStore) getErr() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.err
}

// Create stores a learning record and returns the ID.
func (m *LearningStore) Create(ctx context.Context, record models.LearningRecord) (string, error) {
	if err := m.getErr(); err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	id := fmt.Sprintf("lrn-%d", len(m.records)+1)
	cp := record
	cp.LearningID = id
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now()
	}
	cp.EnrichmentMetadata = models.NormalizeEnrichmentMetadata(cp.EnrichmentMetadata)
	m.records[id] = &cp
	return id, nil
}

// Get retrieves a learning record by ID.
func (m *LearningStore) Get(ctx context.Context, learningID string) (models.LearningRecord, error) {
	if err := m.getErr(); err != nil {
		return models.LearningRecord{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.records[learningID]
	if !ok {
		return models.LearningRecord{}, fmt.Errorf("learning %s: %w", learningID, models.ErrNotFound)
	}
	return *rec, nil
}

// SoftDelete marks a learning record as deleted.
func (m *LearningStore) SoftDelete(ctx context.Context, learningID string) error {
	if err := m.getErr(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.records[learningID]
	if !ok {
		return fmt.Errorf("learning %s: %w", learningID, models.ErrNotFound)
	}
	rec.IsDeleted = true
	now := time.Now()
	rec.DeletedAt = &now
	return nil
}

// Supersede updates the resolution of a learning record and sets superseded_by.
func (m *LearningStore) Supersede(ctx context.Context, oldID, newID string) error {
	if err := m.getErr(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.records[oldID]
	if !ok {
		return fmt.Errorf("learning %s: %w", oldID, models.ErrNotFound)
	}
	rec.Resolution = models.ResolutionSuperseded
	rec.SupersededBy = newID
	return nil
}

// UpdateEnrichment updates the enrichment_metadata for a learning record.
func (m *LearningStore) UpdateEnrichment(ctx context.Context, learningID string, enrichmentJSON json.RawMessage) error {
	if err := m.getErr(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.records[learningID]
	if !ok {
		return fmt.Errorf("learning %s: %w", learningID, models.ErrNotFound)
	}
	rec.EnrichmentMetadata = enrichmentJSON
	return nil
}

// UpdateScore updates the score of a learning record.
func (m *LearningStore) UpdateScore(ctx context.Context, learningID string, score float64) error {
	if err := m.getErr(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.records[learningID]
	if !ok {
		return fmt.Errorf("learning %s: %w", learningID, models.ErrNotFound)
	}
	rec.Score = score
	return nil
}

// UpdateLearningRecord overwrites the stored learning record with the given one.
// Used in tests to modify fields like CreatedAt that aren't exposed via the interface.
func (m *LearningStore) UpdateLearningRecord(id string, record models.LearningRecord) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := record
	cp.LearningID = id
	m.records[id] = &cp
}
