package service

import (
	"context"
	"fmt"

	"github.com/ghassan/alms/internal/models"
)

// PullProtocols returns active protocols matching the agent's tags.
func (s *Syncer) PullProtocols(ctx context.Context, agentID string) ([]models.ProtocolRecord, error) {
	tags, err := s.loadAgentTags(ctx, agentID, "pull protocols")
	if err != nil {
		return nil, err
	}

	protocols, err := s.protoStore.Pull(ctx, tags)
	if err != nil {
		return nil, fmt.Errorf("pull protocols for %s: %w", agentID, err)
	}
	return protocols, nil
}

// PullProtocolsSince returns protocols matching the agent's tags that were
// created after the given protocol ID.
func (s *Syncer) PullProtocolsSince(ctx context.Context, agentID string, sinceID string) ([]models.ProtocolRecord, error) {
	tags, err := s.loadAgentTags(ctx, agentID, "pull protocols since")
	if err != nil {
		return nil, err
	}

	protocols, err := s.protoStore.PullSince(ctx, tags, sinceID)
	if err != nil {
		return nil, fmt.Errorf("pull protocols since for %s: %w", agentID, err)
	}
	return protocols, nil
}

func (s *Syncer) loadAgentTags(ctx context.Context, agentID, operation string) ([]string, error) {
	agent, err := s.agentStore.Get(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("%s: get agent %s: %w", operation, agentID, err)
	}
	return agent.Metadata.Tags, nil
}
