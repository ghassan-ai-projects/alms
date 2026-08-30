package storemock

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/ghassan/alms/internal/models"
)

// ProtocolStore is an in-memory mock of service.ProtocolStore.
type ProtocolStore struct {
	mu        sync.Mutex
	protocols map[string]*models.ProtocolRecord
	err       error
}

// NewProtocolStore creates an empty ProtocolStore mock.
func NewProtocolStore() *ProtocolStore {
	return &ProtocolStore{
		protocols: make(map[string]*models.ProtocolRecord),
	}
}

// SetError configures an error to be returned on all subsequent calls.
func (m *ProtocolStore) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

func (m *ProtocolStore) getErr() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.err
}

// Create stores a protocol record and returns the ID.
func (m *ProtocolStore) Create(ctx context.Context, record models.ProtocolRecord) (string, error) {
	if err := m.getErr(); err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	id := fmt.Sprintf("proto-%d", len(m.protocols)+1)
	cp := record
	cp.ProtocolID = id
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now()
	}
	m.protocols[id] = &cp
	return id, nil
}

// Get retrieves a protocol by ID.
func (m *ProtocolStore) Get(ctx context.Context, protocolID string) (models.ProtocolRecord, error) {
	if err := m.getErr(); err != nil {
		return models.ProtocolRecord{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.protocols[protocolID]
	if !ok {
		return models.ProtocolRecord{}, fmt.Errorf("protocol %s: %w", protocolID, models.ErrNotFound)
	}
	return *p, nil
}

// Pull returns active protocols matching the given tags.
func (m *ProtocolStore) Pull(ctx context.Context, agentTags []string) ([]models.ProtocolRecord, error) {
	if err := m.getErr(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []models.ProtocolRecord
	for _, p := range m.protocols {
		if !p.IsActive {
			continue
		}
		if len(p.TargetTags) == 0 || tagsOverlap(p.TargetTags, agentTags) {
			result = append(result, *p)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

// PullSince returns protocols matching tags created after the given protocol ID.
func (m *ProtocolStore) PullSince(ctx context.Context, agentTags []string, sinceID string) ([]models.ProtocolRecord, error) {
	if err := m.getErr(); err != nil {
		return nil, err
	}
	if sinceID == "" {
		return m.Pull(ctx, agentTags)
	}

	m.mu.Lock()
	sinceProto, ok := m.protocols[sinceID]
	m.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("since protocol %s: %w", sinceID, models.ErrNotFound)
	}
	sinceTime := sinceProto.CreatedAt

	m.mu.Lock()
	defer m.mu.Unlock()

	var result []models.ProtocolRecord
	for _, p := range m.protocols {
		if !p.IsActive {
			continue
		}
		if !p.CreatedAt.After(sinceTime) {
			continue
		}
		if len(p.TargetTags) == 0 || tagsOverlap(p.TargetTags, agentTags) {
			result = append(result, *p)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

// List returns all protocol records.
func (m *ProtocolStore) List(ctx context.Context) ([]models.ProtocolRecord, error) {
	if err := m.getErr(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []models.ProtocolRecord
	for _, p := range m.protocols {
		result = append(result, *p)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}
