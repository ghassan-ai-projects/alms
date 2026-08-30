// Package storemock provides mock store implementations for service-level tests.
package storemock

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/ghassan/alms/internal/models"
)

// AgentStore is an in-memory mock of service.AgentStore.
type AgentStore struct {
	mu     sync.Mutex
	agents map[string]*models.AgentSpec
	err    error // injected error for testing failure paths
}

// NewAgentStore creates an empty AgentStore mock.
func NewAgentStore() *AgentStore {
	return &AgentStore{
		agents: make(map[string]*models.AgentSpec),
	}
}

// SetError configures an error to be returned on all subsequent calls.
func (m *AgentStore) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

func (m *AgentStore) getErr() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.err
}

func (m *AgentStore) Create(ctx context.Context, spec models.AgentSpec) error {
	if err := m.getErr(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.agents[spec.AgentID]; exists {
		return fmt.Errorf("%w: agent %s", models.ErrConflict, spec.AgentID)
	}
	cp := spec
	m.agents[spec.AgentID] = &cp
	return nil
}

func (m *AgentStore) Get(ctx context.Context, agentID string) (models.AgentSpec, error) {
	if err := m.getErr(); err != nil {
		return models.AgentSpec{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.agents[agentID]
	if !ok {
		return models.AgentSpec{}, fmt.Errorf("agent %s: %w", agentID, models.ErrNotFound)
	}
	return *a, nil
}

func (m *AgentStore) Update(ctx context.Context, spec models.AgentSpec) error {
	if err := m.getErr(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.agents[spec.AgentID]; !ok {
		return fmt.Errorf("agent %s: %w", spec.AgentID, models.ErrNotFound)
	}
	cp := spec
	m.agents[spec.AgentID] = &cp
	return nil
}

func (m *AgentStore) Delete(ctx context.Context, agentID string) error {
	if err := m.getErr(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.agents[agentID]; !ok {
		return fmt.Errorf("agent %s: %w", agentID, models.ErrNotFound)
	}
	delete(m.agents, agentID)
	return nil
}

func (m *AgentStore) Heartbeat(ctx context.Context, agentID string) (time.Time, error) {
	if err := m.getErr(); err != nil {
		return time.Time{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.agents[agentID]
	if !ok {
		return time.Time{}, fmt.Errorf("agent %s: %w", agentID, models.ErrNotFound)
	}
	now := time.Now()
	a.LastHeartbeat = now
	return now, nil
}

func (m *AgentStore) List(ctx context.Context, filter map[string]string, limit, offset int) ([]models.AgentSpec, error) {
	if err := m.getErr(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var filtered []models.AgentSpec
	for _, a := range m.agents {
		if agentType, ok := filter["agent_type"]; ok && agentType != "" {
			if string(a.AgentType) != agentType {
				continue
			}
		}
		filtered = append(filtered, *a)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
	})

	if offset >= len(filtered) {
		return []models.AgentSpec{}, nil
	}
	filtered = filtered[offset:]
	if limit > 0 && limit < len(filtered) {
		filtered = filtered[:limit]
	}
	if filtered == nil {
		return []models.AgentSpec{}, nil
	}
	return filtered, nil
}

// Count returns the total number of registered agents.
func (m *AgentStore) Count(ctx context.Context) (int, error) {
	if err := m.getErr(); err != nil {
		return 0, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.agents), nil
}

// GetAll returns a copy of all stored agents (for test assertions).
func (m *AgentStore) GetAll() map[string]models.AgentSpec {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[string]models.AgentSpec, len(m.agents))
	for k, v := range m.agents {
		result[k] = *v
	}
	return result
}
