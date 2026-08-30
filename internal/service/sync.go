package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ghassan/alms/internal/models"
)

// Syncer orchestrates learning sync and gap-safe acknowledgment between agents.
type Syncer struct {
	learnStore LearningStore
	agentStore AgentStore
	protoStore ProtocolStore
}

// NewSyncer creates a new Syncer backed by the given stores.
func NewSyncer(learnStore LearningStore, agentStore AgentStore, protoStore ProtocolStore) *Syncer {
	return &Syncer{
		learnStore: learnStore,
		agentStore: agentStore,
		protoStore: protoStore,
	}
}

// Sync retrieves new learnings for an agent since the given timestamp, optionally
// filtered by type and tags.
func (s *Syncer) Sync(ctx context.Context, agentID string, since time.Time, ltype string, tags []string) ([]models.LearningRecord, error) {
	records, err := s.learnStore.Sync(ctx, agentID, since, ltype, tags)
	if err != nil {
		return nil, fmt.Errorf("sync for agent %s: %w", agentID, err)
	}
	return records, nil
}
