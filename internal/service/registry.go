package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ghassan/alms/internal/models"
)

// Registry manages agent lifecycle: register, unregister, update, list, heartbeat.
type Registry struct {
	store AgentStore
}

// NewRegistry creates a new Registry backed by the given store.
func NewRegistry(store AgentStore) *Registry {
	return &Registry{store: store}
}

// Register creates a new agent after validation. Returns ErrConflict if the
// agent already exists.
func (r *Registry) Register(ctx context.Context, spec models.AgentSpec) error {
	if err := spec.Validate(); err != nil {
		return fmt.Errorf("register agent: %w", err)
	}

	// Check if agent already exists
	existing, err := r.store.Get(ctx, spec.AgentID)
	if err == nil && existing.AgentID != "" {
		return fmt.Errorf("register agent %s: %w", spec.AgentID, models.ErrConflict)
	}

	spec.HealthScore = 1.0
	spec.CreatedAt = time.Now()
	spec.UpdatedAt = time.Now()

	if err := r.store.Create(ctx, spec); err != nil {
		return fmt.Errorf("register agent %s: %w", spec.AgentID, err)
	}
	return nil
}

// Get retrieves an agent by ID.
func (r *Registry) Get(ctx context.Context, agentID string) (models.AgentSpec, error) {
	spec, err := r.store.Get(ctx, agentID)
	if err != nil {
		return spec, fmt.Errorf("get agent %s: %w", agentID, err)
	}
	return spec, nil
}

// Update modifies an existing agent after validation.
func (r *Registry) Update(ctx context.Context, agentID string, spec models.AgentSpec) error {
	if err := spec.Validate(); err != nil {
		return fmt.Errorf("update agent: %w", err)
	}

	spec.AgentID = agentID
	if err := r.store.Update(ctx, spec); err != nil {
		return fmt.Errorf("update agent %s: %w", agentID, err)
	}
	return nil
}

// List returns agents matching the optional filter with pagination.
func (r *Registry) List(ctx context.Context, filter map[string]string, limit, offset int) ([]models.AgentSpec, error) {
	agents, err := r.store.List(ctx, filter, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	return agents, nil
}

// Delete removes an agent by ID.
func (r *Registry) Delete(ctx context.Context, agentID string) error {
	if err := r.store.Delete(ctx, agentID); err != nil {
		return fmt.Errorf("delete agent %s: %w", agentID, err)
	}
	return nil
}

// Heartbeat updates the last_heartbeat timestamp for an agent.
func (r *Registry) Heartbeat(ctx context.Context, agentID string) (time.Time, error) {
	ts, err := r.store.Heartbeat(ctx, agentID)
	if err != nil {
		return ts, fmt.Errorf("heartbeat agent %s: %w", agentID, err)
	}
	return ts, nil
}
