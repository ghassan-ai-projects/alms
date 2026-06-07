// Package service provides business logic for ALMS operations.
// All service methods accept store interfaces for testability.
package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ghassan/alms/internal/models"
)

// AgentStore defines the data access interface for agents.
type AgentStore interface {
	Create(ctx context.Context, spec models.AgentSpec) error
	Get(ctx context.Context, agentID string) (models.AgentSpec, error)
	Update(ctx context.Context, spec models.AgentSpec) error
	Delete(ctx context.Context, agentID string) error
	Heartbeat(ctx context.Context, agentID string) (time.Time, error)
	List(ctx context.Context, filter map[string]string, limit, offset int) ([]models.AgentSpec, error)
	Count(ctx context.Context) (int, error)
}

// LearningStore defines the data access interface for learnings.
type LearningStore interface {
	Create(ctx context.Context, record models.LearningRecord) (string, error)
	Get(ctx context.Context, learningID string) (models.LearningRecord, error)
	Sync(ctx context.Context, agentID string, since time.Time, ltype string, tags []string) ([]models.LearningRecord, error)
	SyncAck(ctx context.Context, agentID string, learningIDs []string) error
	Search(ctx context.Context, query string, ltype string, tags []string, limit int) ([]models.LearningRecord, error)
	SoftDelete(ctx context.Context, learningID string) error
	ExpectedSyncIDs(ctx context.Context, agentID string, since time.Time) ([]string, error)
	Supersede(ctx context.Context, oldID, newID string) error
	UpdateScore(ctx context.Context, learningID string, score float64) error
	UpdateEnrichment(ctx context.Context, learningID string, enrichmentJSON json.RawMessage) error
	SearchWithStatus(ctx context.Context, query string, ltype string, tags []string, limit int, status string, includeRejected bool) ([]models.LearningRecord, error)
}

// ProtocolStore defines the data access interface for protocols.
type ProtocolStore interface {
	Create(ctx context.Context, record models.ProtocolRecord) (string, error)
	Get(ctx context.Context, protocolID string) (models.ProtocolRecord, error)
	Pull(ctx context.Context, agentTags []string) ([]models.ProtocolRecord, error)
	PullSince(ctx context.Context, agentTags []string, sinceID string) ([]models.ProtocolRecord, error)
	List(ctx context.Context) ([]models.ProtocolRecord, error)
}
