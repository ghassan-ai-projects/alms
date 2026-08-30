package storemock

import (
	"context"
	"testing"

	"github.com/ghassan/alms/internal/models"
)

func TestMockStoresPersistBasicRecords(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	agentStore := NewAgentStore()
	agent := models.AgentSpec{AgentID: "agent-1", AgentType: models.AgentTypeSystemd}
	if err := agentStore.Create(ctx, agent); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if got, err := agentStore.Get(ctx, agent.AgentID); err != nil || got.AgentID != agent.AgentID {
		t.Fatalf("get agent: got=%+v err=%v", got, err)
	}

	learningStore := NewLearningStore()
	learningID, err := learningStore.Create(ctx, models.LearningRecord{Title: "title", Type: models.LearningTypePattern})
	if err != nil {
		t.Fatalf("create learning: %v", err)
	}
	if got, err := learningStore.Get(ctx, learningID); err != nil || got.LearningID != learningID {
		t.Fatalf("get learning: got=%+v err=%v", got, err)
	}

	protocolStore := NewProtocolStore()
	protocolID, err := protocolStore.Create(ctx, models.ProtocolRecord{Title: "protocol", IsActive: true})
	if err != nil {
		t.Fatalf("create protocol: %v", err)
	}
	if got, err := protocolStore.Get(ctx, protocolID); err != nil || got.ProtocolID != protocolID {
		t.Fatalf("get protocol: got=%+v err=%v", got, err)
	}
}

func TestMatchingHelpers(t *testing.T) {
	if !containsSubstring("Agent Learning", "learning") {
		t.Fatal("containsSubstring should ignore ASCII case")
	}
	if containsSubstring("agent", "protocol") {
		t.Fatal("containsSubstring should reject missing text")
	}
	if !tagsOverlap([]string{"agent", "mcp"}, []string{"protocol", "mcp"}) {
		t.Fatal("tagsOverlap should find a shared tag")
	}
	if tagsOverlap([]string{"agent"}, []string{"protocol"}) {
		t.Fatal("tagsOverlap should reject disjoint tags")
	}
}
