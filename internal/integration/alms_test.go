//go:build integration

// Package integration contains end-to-end integration tests that require
// a real PostgreSQL database. These tests are excluded from normal `go test`
// and only run with: ALMS_PG_DSN=... go test -tags=integration -race -count=1 ./...
//
// Each test creates its own set of records and cleans up after itself.
package integration

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ghassan/alms/internal/models"
	"github.com/ghassan/alms/internal/service"
	"github.com/ghassan/alms/internal/store"
)

const testAgentID = "alms-integration-test-agent"

// connectPool returns a pgx pool using ALMS_PG_DSN, or skips the test if unset.
func connectPool(tb testing.TB) *pgxpool.Pool {
	tb.Helper()

	dsn := os.Getenv("ALMS_PG_DSN")
	if dsn == "" {
		tb.Skip("ALMS_PG_DSN not set — skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := store.NewPool(ctx, dsn)
	if err != nil {
		tb.Fatalf("NewPool: %v", err)
	}
	tb.Cleanup(func() { pool.Close() })
	return pool
}

// makeStores creates concrete store instances from a pool.
func makeStores(pool *pgxpool.Pool) (
	*store.AgentStore,
	*store.LearningStore,
	*store.ProtocolStore,
) {
	return store.NewAgentStore(pool),
		store.NewLearningStore(pool),
		store.NewProtocolStore(pool)
}

// cleanupTestAgent removes the test agent if it exists.
func cleanupTestAgent(ctx context.Context, aStore *store.AgentStore, agentID string) {
	_ = aStore.Delete(ctx, agentID)
}

// ============================================================================
// Test 1: Full E2E — register agent → push 3 learnings → sync → ack → verify empty
// ============================================================================

func TestIntegrationFullSyncE2E(t *testing.T) {
	pool := connectPool(t)
	aStore, lStore, pStore := makeStores(pool)
	ctx := context.Background()

	agentID := fmt.Sprintf("%s-e2e-%d", testAgentID, time.Now().UnixNano())
	t.Cleanup(func() { cleanupTestAgent(ctx, aStore, agentID) })

	// Register agent
	reg := service.NewRegistry(aStore)
	spec := models.AgentSpec{
		AgentID:   agentID,
		AgentType: models.AgentTypeSystemd,
	}
	if err := reg.Register(ctx, spec); err != nil {
		t.Fatalf("Register agent: %v", err)
	}

	// Push 3 learnings via Learning service
	learnSvc := service.NewLearning(lStore, pStore)
	learningTitles := []string{
		"Integration test learning A",
		"Integration test learning B",
		"Integration test learning C",
	}
	var learningIDs []string
	for _, title := range learningTitles {
		id, err := learnSvc.Store(ctx, models.LearningRecord{
			Title: title,
			Body:  fmt.Sprintf("Body for %s", title),
			Type:  models.LearningTypePattern,
		}, "")
		if err != nil {
			t.Fatalf("Store learning %q: %v", title, err)
		}
		learningIDs = append(learningIDs, id)
	}
	t.Logf("Stored %d learnings: %v", len(learningIDs), learningIDs)

	// Sync from epoch — should return all 3
	syncer := service.NewSyncer(lStore, aStore, pStore)
	epoch := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	synced, err := syncer.Sync(ctx, agentID, epoch, "", nil)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if len(synced) != 3 {
		t.Fatalf("expected 3 learnings from sync, got %d", len(synced))
	}

	// Ack all 3
	if err := syncer.SyncAck(ctx, agentID, learningIDs); err != nil {
		t.Fatalf("SyncAck: %v", err)
	}

	// Sync again — should be empty
	synced2, err := syncer.Sync(ctx, agentID, epoch, "", nil)
	if err != nil {
		t.Fatalf("Second Sync: %v", err)
	}
	if len(synced2) != 0 {
		t.Fatalf("expected 0 learnings after ack, got %d", len(synced2))
	}
}

// ============================================================================
// Test 2: Crash recovery — ack partial → resync → remaining returned
// ============================================================================

func TestIntegrationCrashRecovery(t *testing.T) {
	pool := connectPool(t)
	aStore, lStore, pStore := makeStores(pool)
	ctx := context.Background()

	agentID := fmt.Sprintf("%s-crash-%d", testAgentID, time.Now().UnixNano())
	t.Cleanup(func() { cleanupTestAgent(ctx, aStore, agentID) })

	// Register agent
	reg := service.NewRegistry(aStore)
	if err := reg.Register(ctx, models.AgentSpec{
		AgentID:   agentID,
		AgentType: models.AgentTypeSystemd,
	}); err != nil {
		t.Fatalf("Register agent: %v", err)
	}

	// Push 4 learnings
	learnSvc := service.NewLearning(lStore, pStore)
	allIDs := make([]string, 4)
	for i := range allIDs {
		id, err := learnSvc.Store(ctx, models.LearningRecord{
			Title: fmt.Sprintf("Crash recovery learning %d", i+1),
			Type:  models.LearningTypeFailure,
		}, "")
		if err != nil {
			t.Fatalf("Store learning %d: %v", i, err)
		}
		allIDs[i] = id
	}

	syncer := service.NewSyncer(lStore, aStore, pStore)
	epoch := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	// Agent acks only IDs 0 and 1 (partial — simulates crash after half the ack)
	if err := syncer.SyncAck(ctx, agentID, allIDs[:2]); err != nil {
		t.Fatalf("SyncAck (partial): %v", err)
	}

	// Sync again — should return learnings 2 and 3 (the un-acked ones)
	synced, err := syncer.Sync(ctx, agentID, epoch, "", nil)
	if err != nil {
		t.Fatalf("Sync after crash: %v", err)
	}
	if len(synced) != 2 {
		t.Fatalf("expected 2 un-acked learnings after crash, got %d", len(synced))
	}
	if synced[0].LearningID != allIDs[2] || synced[1].LearningID != allIDs[3] {
		t.Fatalf("expected remaining IDs %v, got %v", allIDs[2:], idsOf(synced))
	}

	// Ack remaining
	if err := syncer.SyncAck(ctx, agentID, []string{allIDs[2], allIDs[3]}); err != nil {
		t.Fatalf("SyncAck (remaining): %v", err)
	}

	// Verify empty
	empty, err := syncer.Sync(ctx, agentID, epoch, "", nil)
	if err != nil {
		t.Fatalf("Final Sync: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected 0 after full ack, got %d", len(empty))
	}
}

// ============================================================================
// Test 3: Concurrent sync — 10 agents syncing simultaneously
// ============================================================================

func TestIntegrationConcurrentSync(t *testing.T) {
	pool := connectPool(t)
	aStore, lStore, pStore := makeStores(pool)
	ctx := context.Background()
	epoch := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	const numAgents = 10

	// Create the agents
	reg := service.NewRegistry(aStore)
	agentIDs := make([]string, numAgents)
	for i := range agentIDs {
		agentIDs[i] = fmt.Sprintf("%s-concurrent-%d-%d", testAgentID, i, time.Now().UnixNano())
		if err := reg.Register(ctx, models.AgentSpec{
			AgentID:   agentIDs[i],
			AgentType: models.AgentTypeSystemd,
		}); err != nil {
			t.Fatalf("Register agent %d: %v", i, err)
		}
		t.Cleanup(func() { cleanupTestAgent(ctx, aStore, agentIDs[i]) })
	}

	// Store 5 learnings
	learnSvc := service.NewLearning(lStore, pStore)
	learningIDs := make([]string, 5)
	for i := range learningIDs {
		id, err := learnSvc.Store(ctx, models.LearningRecord{
			Title: fmt.Sprintf("Concurrent test learning %d", i+1),
			Type:  models.LearningTypeConfig,
		}, "")
		if err != nil {
			t.Fatalf("Store learning %d: %v", i, err)
		}
		learningIDs[i] = id
	}

	// All 10 agents sync + ack concurrently
	syncer := service.NewSyncer(lStore, aStore, pStore)
	var wg sync.WaitGroup
	errCh := make(chan error, numAgents)

	for _, aid := range agentIDs {
		wg.Add(1)
		go func(agentID string) {
			defer wg.Done()

			// Sync
			records, err := syncer.Sync(ctx, agentID, epoch, "", nil)
			if err != nil {
				errCh <- fmt.Errorf("agent %s sync: %w", agentID, err)
				return
			}
			if len(records) != 5 {
				errCh <- fmt.Errorf("agent %s expected 5 learnings, got %d", agentID, len(records))
				return
			}

			// Ack
			ids := make([]string, len(records))
			for i, r := range records {
				ids[i] = r.LearningID
			}
			if err := syncer.SyncAck(ctx, agentID, ids); err != nil {
				errCh <- fmt.Errorf("agent %s sync ack: %w", agentID, err)
				return
			}
		}(aid)
	}
	wg.Wait()
	close(errCh)

	var failures []error
	for err := range errCh {
		failures = append(failures, err)
	}
	if len(failures) > 0 {
		t.Fatalf("%d concurrent sync failures: %v", len(failures), failures)
	}
}

// ============================================================================
// Test 4: Protocol matching — create SOP with tags, verify agent gets it
// ============================================================================

func TestIntegrationProtocolMatching(t *testing.T) {
	pool := connectPool(t)
	aStore, lStore, pStore := makeStores(pool)
	ctx := context.Background()

	agentID := fmt.Sprintf("%s-proto-%d", testAgentID, time.Now().UnixNano())
	t.Cleanup(func() { cleanupTestAgent(ctx, aStore, agentID) })

	// Register agent with specific tags
	reg := service.NewRegistry(aStore)
	if err := reg.Register(ctx, models.AgentSpec{
		AgentID:   agentID,
		AgentType: models.AgentTypeSystemd,
		Metadata: models.AgentMetadata{
			Tags: []string{"newsletter", "python"},
		},
	}); err != nil {
		t.Fatalf("Register agent: %v", err)
	}

	// Push 2 protocols: one matching agent tags, one without matching tags
	learnSvc := service.NewLearning(lStore, pStore)

	matchingProtoID, err := learnSvc.ProtocolPush(ctx, models.ProtocolRecord{
		Title:      "Newsletter SOP",
		Body:       "Standard operating procedure for newsletters",
		TargetTags: []string{"newsletter", "python"},
		Author:     "integration-test",
	})
	if err != nil {
		t.Fatalf("ProtocolPush (matching): %v", err)
	}
	t.Logf("Matching protocol ID: %s", matchingProtoID)

	nonMatchingProtoID, err := learnSvc.ProtocolPush(ctx, models.ProtocolRecord{
		Title:      "System Admin SOP",
		Body:       "Standard operating procedure for sysadmins",
		TargetTags: []string{"sysadmin", "linux"},
		Author:     "integration-test",
	})
	if err != nil {
		t.Fatalf("ProtocolPush (non-matching): %v", err)
	}
	t.Logf("Non-matching protocol ID: %s", nonMatchingProtoID)

	// Agent pulls protocols through the syncer
	syncer := service.NewSyncer(lStore, aStore, pStore)
	protocols, err := syncer.PullProtocols(ctx, agentID)
	if err != nil {
		t.Fatalf("PullProtocols: %v", err)
	}

	// Should only get the matching one
	found := false
	for _, p := range protocols {
		if p.ProtocolID == matchingProtoID {
			found = true
		}
		if p.ProtocolID == nonMatchingProtoID {
			t.Errorf("agent received non-matching protocol (sysadmin SOP)")
		}
	}
	if !found {
		t.Errorf("agent did not receive matching protocol (Newsletter SOP)")
	}
}

// ============================================================================
// Helpers
// ============================================================================

// idsOf extracts learning IDs from a slice of records.
func idsOf(records []models.LearningRecord) []string {
	ids := make([]string, len(records))
	for i, r := range records {
		ids[i] = r.LearningID
	}
	return ids
}

// TestIntegrationMain ensures the package compiles and the build tag is correct.
func TestIntegrationMain(t *testing.T) {
	t.Log("integration test package loaded (build tag: integration)")
}
