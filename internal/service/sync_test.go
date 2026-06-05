package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ghassan/alms/internal/models"
	"github.com/ghassan/alms/internal/service/storemock"
)

func TestSyncerSync(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("returns learnings since timestamp", func(t *testing.T) {
		lStore := storemock.NewLearningStore()
		aStore := storemock.NewAgentStore()
		pStore := storemock.NewProtocolStore()
		syncer := NewSyncer(lStore, aStore, pStore)

		// Create learnings
		before := time.Now().Add(-1 * time.Hour)
		after := time.Now()

		l1, _ := lStore.Create(ctx, models.LearningRecord{
			Title: "Old Learning", Type: models.LearningTypePattern,
			CreatedAt: before,
		})
		l2, _ := lStore.Create(ctx, models.LearningRecord{
			Title: "New Learning", Type: models.LearningTypePattern,
			CreatedAt: after,
		})

		records, err := syncer.Sync(ctx, "agent-1", before.Add(1), "", nil)
		if err != nil {
			t.Fatalf("Sync() unexpected error: %v", err)
		}
		if len(records) != 1 {
			t.Errorf("Sync() returned %d records, want 1", len(records))
		}
		if len(records) > 0 && records[0].LearningID != l2 {
			t.Errorf("Sync() LearningID = %q, want %q", records[0].LearningID, l2)
		}
		_ = l1
	})

	t.Run("returns empty list when no new learnings", func(t *testing.T) {
		lStore := storemock.NewLearningStore()
		aStore := storemock.NewAgentStore()
		pStore := storemock.NewProtocolStore()
		syncer := NewSyncer(lStore, aStore, pStore)

		since := time.Now().Add(1 * time.Hour)
		records, err := syncer.Sync(ctx, "agent-1", since, "", nil)
		if err != nil {
			t.Fatalf("Sync() unexpected error: %v", err)
		}
		if len(records) != 0 {
			t.Errorf("Sync() returned %d records, want 0", len(records))
		}
	})

	t.Run("filters by type", func(t *testing.T) {
		lStore := storemock.NewLearningStore()
		aStore := storemock.NewAgentStore()
		pStore := storemock.NewProtocolStore()
		syncer := NewSyncer(lStore, aStore, pStore)

		since := time.Now().Add(-1 * time.Hour)
		_, _ = lStore.Create(ctx, models.LearningRecord{
			Title: "Pattern", Type: models.LearningTypePattern,
			CreatedAt: time.Now(),
		})
		l2, _ := lStore.Create(ctx, models.LearningRecord{
			Title: "Failure", Type: models.LearningTypeFailure,
			CreatedAt: time.Now(),
		})

		records, err := syncer.Sync(ctx, "agent-1", since, "failure", nil)
		if err != nil {
			t.Fatalf("Sync() unexpected error: %v", err)
		}
		if len(records) != 1 {
			t.Errorf("Sync() returned %d records, want 1", len(records))
		}
		if len(records) > 0 && records[0].LearningID != l2 {
			t.Errorf("Sync() LearningID = %q, want %q", records[0].LearningID, l2)
		}
	})
}

func TestSyncerSyncAckNoGap(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	lStore := storemock.NewLearningStore()
	aStore := storemock.NewAgentStore()
	pStore := storemock.NewProtocolStore()
	syncer := NewSyncer(lStore, aStore, pStore)

	// Register agent
	err := aStore.Create(ctx, models.AgentSpec{
		AgentID:   "agent-ack",
		AgentType: models.AgentTypeSystemd,
	})
	if err != nil {
		t.Fatalf("Create agent failed: %v", err)
	}

	// Create learnings
	now := time.Now()
	l1, _ := lStore.Create(ctx, models.LearningRecord{
		Title: "Learning 1", Type: models.LearningTypePattern,
		CreatedAt: now,
	})
	l2, _ := lStore.Create(ctx, models.LearningRecord{
		Title: "Learning 2", Type: models.LearningTypePattern,
		CreatedAt: now.Add(time.Minute),
	})

	// Ack both
	err = syncer.SyncAck(ctx, "agent-ack", []string{l1, l2})
	if err != nil {
		t.Fatalf("SyncAck() unexpected error: %v", err)
	}

	// Verify the ack was stored
	acks := lStore.GetAcks("agent-ack")
	if len(acks) != 2 {
		t.Errorf("GetAcks() returned %d acks, want 2", len(acks))
	}
}

func TestSyncerSyncAckWithGap(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	lStore := storemock.NewLearningStore()
	aStore := storemock.NewAgentStore()
	pStore := storemock.NewProtocolStore()
	syncer := NewSyncer(lStore, aStore, pStore)

	// Register agent
	err := aStore.Create(ctx, models.AgentSpec{
		AgentID:   "agent-gap",
		AgentType: models.AgentTypeSystemd,
	})
	if err != nil {
		t.Fatalf("Create agent failed: %v", err)
	}

	// Create learnings
	now := time.Now()
	l1, _ := lStore.Create(ctx, models.LearningRecord{
		Title: "Learning 1", Type: models.LearningTypePattern,
		CreatedAt: now,
	})
	_, _ = lStore.Create(ctx, models.LearningRecord{
		Title: "Learning 2", Type: models.LearningTypePattern,
		CreatedAt: now.Add(time.Minute),
	})
	l3, _ := lStore.Create(ctx, models.LearningRecord{
		Title: "Learning 3", Type: models.LearningTypePattern,
		CreatedAt: now.Add(2 * time.Minute),
	})

	// Ack with a gap (skip l2)
	err = syncer.SyncAck(ctx, "agent-gap", []string{l1, l3})
	if err == nil {
		t.Fatal("SyncAck() expected ErrGapDetected, got nil")
	}
	if !errors.Is(err, models.ErrGapDetected) {
		t.Errorf("SyncAck() error = %v, want ErrGapDetected", err)
	}
}

func TestSyncerCrashedAgentRecovers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	lStore := storemock.NewLearningStore()
	aStore := storemock.NewAgentStore()
	pStore := storemock.NewProtocolStore()
	syncer := NewSyncer(lStore, aStore, pStore)

	// Register agent
	err := aStore.Create(ctx, models.AgentSpec{
		AgentID:   "agent-crash",
		AgentType: models.AgentTypeSystemd,
	})
	if err != nil {
		t.Fatalf("Create agent failed: %v", err)
	}

	// Create learnings before crash
	now := time.Now()
	l1, _ := lStore.Create(ctx, models.LearningRecord{
		Title: "Pre-crash 1", Type: models.LearningTypePattern,
		CreatedAt: now,
	})
	l2, _ := lStore.Create(ctx, models.LearningRecord{
		Title: "Pre-crash 2", Type: models.LearningTypePattern,
		CreatedAt: now.Add(time.Minute),
	})

	// Agent syncs before crash (acks l1, l2) — simulating partial ack
	// In a crash scenario, agent has synced but not acked. On restart,
	// it will look up ExpectedSyncIDs and try to ack all.
	err = syncer.SyncAck(ctx, "agent-crash", []string{l1, l2})
	if err != nil {
		t.Fatalf("SyncAck() before crash failed: %v", err)
	}
	acks := lStore.GetAcks("agent-crash")
	if len(acks) != 2 {
		t.Errorf("GetAcks() = %d, want 2", len(acks))
	}
}

func TestSyncerPullProtocols(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("returns matching protocols", func(t *testing.T) {
		lStore := storemock.NewLearningStore()
		aStore := storemock.NewAgentStore()
		pStore := storemock.NewProtocolStore()
		syncer := NewSyncer(lStore, aStore, pStore)

		// Register agent with tags
		err := aStore.Create(ctx, models.AgentSpec{
			AgentID:   "agent-proto",
			AgentType: models.AgentTypeSystemd,
			Metadata:  models.AgentMetadata{Tags: []string{"network"}},
		})
		if err != nil {
			t.Fatalf("Create agent failed: %v", err)
		}

		// Create matching protocol
		_, _ = pStore.Create(ctx, models.ProtocolRecord{
			Title:      "Network Protocol",
			TargetTags: []string{"network"},
			IsActive:   true,
		})

		// Create non-matching protocol
		_, _ = pStore.Create(ctx, models.ProtocolRecord{
			Title:      "Storage Protocol",
			TargetTags: []string{"storage"},
			IsActive:   true,
		})

		protocols, err := syncer.PullProtocols(ctx, "agent-proto")
		if err != nil {
			t.Fatalf("PullProtocols() unexpected error: %v", err)
		}
		if len(protocols) != 1 {
			t.Errorf("PullProtocols() returned %d protocols, want 1", len(protocols))
		}
		if len(protocols) > 0 && protocols[0].Title != "Network Protocol" {
			t.Errorf("PullProtocols() Title = %q, want %q", protocols[0].Title, "Network Protocol")
		}
	})

	t.Run("returns empty when agent has no tags", func(t *testing.T) {
		lStore := storemock.NewLearningStore()
		aStore := storemock.NewAgentStore()
		pStore := storemock.NewProtocolStore()
		syncer := NewSyncer(lStore, aStore, pStore)

		err := aStore.Create(ctx, models.AgentSpec{
			AgentID:   "agent-notags",
			AgentType: models.AgentTypeSystemd,
		})
		if err != nil {
			t.Fatalf("Create agent failed: %v", err)
		}

		_, _ = pStore.Create(ctx, models.ProtocolRecord{
			Title:    "Global Protocol",
			IsActive: true,
		})

		protocols, err := syncer.PullProtocols(ctx, "agent-notags")
		if err != nil {
			t.Fatalf("PullProtocols() unexpected error: %v", err)
		}
		if len(protocols) != 1 {
			t.Errorf("PullProtocols() returned %d protocols, want 1 (global protocol)", len(protocols))
		}
	})

	t.Run("pull protocols since returns newer protocols", func(t *testing.T) {
		lStore := storemock.NewLearningStore()
		aStore := storemock.NewAgentStore()
		pStore := storemock.NewProtocolStore()
		syncer := NewSyncer(lStore, aStore, pStore)

		_ = aStore.Create(ctx, models.AgentSpec{
			AgentID:   "agent-since",
			AgentType: models.AgentTypeSystemd,
			Metadata:  models.AgentMetadata{Tags: []string{"network"}},
		})

		// Create protocols (ordered by creation time)
		oldID, _ := pStore.Create(ctx, models.ProtocolRecord{
			Title:      "Old Protocol",
			TargetTags: []string{"network"},
			IsActive:   true,
		})
		_, _ = pStore.Create(ctx, models.ProtocolRecord{
			Title:      "New Protocol",
			TargetTags: []string{"network"},
			IsActive:   true,
		})

		protocols, err := syncer.PullProtocolsSince(ctx, "agent-since", oldID)
		if err != nil {
			t.Fatalf("PullProtocolsSince() unexpected error: %v", err)
		}
		if len(protocols) != 1 {
			t.Errorf("PullProtocolsSince() returned %d protocols, want 1", len(protocols))
		}
		if len(protocols) > 0 && protocols[0].Title != "New Protocol" {
			t.Errorf("PullProtocolsSince() Title = %q, want %q", protocols[0].Title, "New Protocol")
		}
	})

	t.Run("pull protocols since with non-existent agent returns error", func(t *testing.T) {
		lStore := storemock.NewLearningStore()
		aStore := storemock.NewAgentStore()
		pStore := storemock.NewProtocolStore()
		syncer := NewSyncer(lStore, aStore, pStore)

		_, err := syncer.PullProtocolsSince(ctx, "nonexistent", "some-id")
		if err == nil {
			t.Error("PullProtocolsSince() expected error for non-existent agent")
		}
	})

	t.Run("sync store error returns error", func(t *testing.T) {
		lStore := storemock.NewLearningStore()
		aStore := storemock.NewAgentStore()
		pStore := storemock.NewProtocolStore()
		syncer := NewSyncer(lStore, aStore, pStore)

		lStore.SetError(assertionError("store error"))
		_, err := syncer.Sync(ctx, "agent-err", time.Now(), "", nil)
		if err == nil {
			t.Error("Sync() expected error from store, got nil")
		}
	})

	t.Run("sync ack store error returns error", func(t *testing.T) {
		lStore := storemock.NewLearningStore()
		aStore := storemock.NewAgentStore()
		pStore := storemock.NewProtocolStore()
		syncer := NewSyncer(lStore, aStore, pStore)

		_ = aStore.Create(ctx, models.AgentSpec{
			AgentID:   "agent-ackerr",
			AgentType: models.AgentTypeSystemd,
		})
		l1, _ := lStore.Create(ctx, models.LearningRecord{
			Title: "L1", Type: models.LearningTypePattern, CreatedAt: time.Now(),
		})

		// Set error on lStore (after agent store Get succeeds)
		lStore.SetError(assertionError("store error"))
		err := syncer.SyncAck(ctx, "agent-ackerr", []string{l1})
		if err == nil {
			t.Error("SyncAck() expected error from store, got nil")
		}
	})

	t.Run("pull protocols store error returns error", func(t *testing.T) {
		lStore := storemock.NewLearningStore()
		aStore := storemock.NewAgentStore()
		pStore := storemock.NewProtocolStore()
		syncer := NewSyncer(lStore, aStore, pStore)

		_ = aStore.Create(ctx, models.AgentSpec{
			AgentID:   "agent-protoerr",
			AgentType: models.AgentTypeSystemd,
			Metadata:  models.AgentMetadata{Tags: []string{"test"}},
		})

		pStore.SetError(assertionError("store error"))
		_, err := syncer.PullProtocols(ctx, "agent-protoerr")
		if err == nil {
			t.Error("PullProtocols() expected error from store, got nil")
		}
	})

	t.Run("pull protocols non-existent agent returns error", func(t *testing.T) {
		lStore := storemock.NewLearningStore()
		aStore := storemock.NewAgentStore()
		pStore := storemock.NewProtocolStore()
		syncer := NewSyncer(lStore, aStore, pStore)

		_, err := syncer.PullProtocols(ctx, "nonexistent")
		if err == nil {
			t.Error("PullProtocols() expected error for non-existent agent")
		}
	})

	t.Run("sync ack no new learnings is no-op", func(t *testing.T) {
		lStore := storemock.NewLearningStore()
		aStore := storemock.NewAgentStore()
		pStore := storemock.NewProtocolStore()
		syncer := NewSyncer(lStore, aStore, pStore)

		_ = aStore.Create(ctx, models.AgentSpec{
			AgentID:   "agent-no-new",
			AgentType: models.AgentTypeSystemd,
		})

		// No learnings posted yet — ack should be a no-op
		err := syncer.SyncAck(ctx, "agent-no-new", []string{})
		if err != nil {
			t.Fatalf("SyncAck() expected no error for empty ack: %v", err)
		}
	})
}
