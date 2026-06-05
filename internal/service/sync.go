package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ghassan/alms/internal/models"
)

// Syncer orchestrates learning sync and gap-safe acknowledgment between agents.
type Syncer struct {
	learnStore  LearningStore
	agentStore  AgentStore
	protoStore  ProtocolStore
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

// SyncAck acknowledges a batch of learning IDs after validating there are no gaps.
// This implements the gap-safe algorithm:
//
//  1. Fetch expected learning IDs since agent's last sync timestamp
//  2. Check none are missing from the provided ack list
//  3. If gaps found, return ErrGapDetected with missing IDs
//  4. If no gaps, persist acknowledgements and advance sync cursor
func (s *Syncer) SyncAck(ctx context.Context, agentID string, learningIDs []string) error {
	// Get the agent's current sync state
	agent, err := s.agentStore.Get(ctx, agentID)
	if err != nil {
		return fmt.Errorf("sync ack: get agent %s: %w", agentID, err)
	}

	since := agent.LastSyncTimestamp
	if since.IsZero() {
		since = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	// Get all learnings the agent SHOULD have received
	expectedIDs, err := s.learnStore.ExpectedSyncIDs(ctx, agentID, since)
	if err != nil {
		return fmt.Errorf("sync ack: expected IDs for %s: %w", agentID, err)
	}

	if len(expectedIDs) == 0 {
		// Nothing to ack
		return nil
	}

	// Build set of acknowledged IDs
	ackSet := make(map[string]bool, len(learningIDs))
	for _, id := range learningIDs {
		ackSet[id] = true
	}

	// Check for gaps
	var missing []string
	for _, id := range expectedIDs {
		if !ackSet[id] {
			missing = append(missing, id)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("%w: missing learning IDs %v", models.ErrGapDetected, missing)
	}

	// Persist the acknowledgements
	if err := s.learnStore.SyncAck(ctx, agentID, learningIDs); err != nil {
		return fmt.Errorf("sync ack: persist for %s: %w", agentID, err)
	}

	return nil
}

// PullProtocols returns active protocols matching the agent's tags.
func (s *Syncer) PullProtocols(ctx context.Context, agentID string) ([]models.ProtocolRecord, error) {
	agent, err := s.agentStore.Get(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("pull protocols: get agent %s: %w", agentID, err)
	}

	tags := agent.Metadata.Tags
	protocols, err := s.protoStore.Pull(ctx, tags)
	if err != nil {
		return nil, fmt.Errorf("pull protocols for %s: %w", agentID, err)
	}
	return protocols, nil
}

// PullProtocolsSince returns protocols matching the agent's tags that were
// created after the given protocol ID.
func (s *Syncer) PullProtocolsSince(ctx context.Context, agentID string, sinceID string) ([]models.ProtocolRecord, error) {
	agent, err := s.agentStore.Get(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("pull protocols since: get agent %s: %w", agentID, err)
	}

	tags := agent.Metadata.Tags
	protocols, err := s.protoStore.PullSince(ctx, tags, sinceID)
	if err != nil {
		return nil, fmt.Errorf("pull protocols since for %s: %w", agentID, err)
	}
	return protocols, nil
}
