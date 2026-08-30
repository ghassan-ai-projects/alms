package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ghassan/alms/internal/models"
)

// SyncAck acknowledges a batch of learning IDs after validating there are no gaps.
func (s *Syncer) SyncAck(ctx context.Context, agentID string, learningIDs []string) error {
	agent, err := s.agentStore.Get(ctx, agentID)
	if err != nil {
		return fmt.Errorf("sync ack: get agent %s: %w", agentID, err)
	}

	since := agent.LastSyncTimestamp
	if since.IsZero() {
		since = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	expectedIDs, err := s.learnStore.ExpectedSyncIDs(ctx, agentID, since)
	if err != nil {
		return fmt.Errorf("sync ack: expected IDs for %s: %w", agentID, err)
	}

	if len(expectedIDs) == 0 {
		return nil
	}

	missing := findMissingLearningIDs(expectedIDs, learningIDs)
	if len(missing) > 0 {
		return fmt.Errorf("%w: missing learning IDs %v", models.ErrGapDetected, missing)
	}

	if err := s.learnStore.SyncAck(ctx, agentID, learningIDs); err != nil {
		return fmt.Errorf("sync ack: persist for %s: %w", agentID, err)
	}

	return nil
}

func findMissingLearningIDs(expectedIDs, acknowledgedIDs []string) []string {
	acknowledged := make(map[string]bool, len(acknowledgedIDs))
	for _, id := range acknowledgedIDs {
		acknowledged[id] = true
	}

	var missing []string
	for _, id := range expectedIDs {
		if !acknowledged[id] {
			missing = append(missing, id)
		}
	}
	return missing
}
