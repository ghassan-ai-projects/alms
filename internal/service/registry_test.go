package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ghassan/alms/internal/models"
	"github.com/ghassan/alms/internal/service/storemock"
)

func TestRegistryRegister(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		store := storemock.NewAgentStore()
		reg := NewRegistry(store)

		spec := models.AgentSpec{
			AgentID:   "agent-1",
			AgentType: models.AgentTypeSystemd,
		}
		err := reg.Register(ctx, spec)
		if err != nil {
			t.Fatalf("Register() unexpected error: %v", err)
		}

		// Verify agent was created
		got, err := store.Get(ctx, "agent-1")
		if err != nil {
			t.Fatalf("Get() failed: %v", err)
		}
		if got.AgentID != "agent-1" {
			t.Errorf("Get() AgentID = %q, want %q", got.AgentID, "agent-1")
		}
		if got.HealthScore != 1.0 {
			t.Errorf("Get() HealthScore = %f, want 1.0", got.HealthScore)
		}
		if got.CreatedAt.IsZero() {
			t.Error("Get() CreatedAt should not be zero")
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		store := storemock.NewAgentStore()
		reg := NewRegistry(store)

		spec := models.AgentSpec{
			AgentID:   "agent-dup",
			AgentType: models.AgentTypeSystemd,
		}
		err := reg.Register(ctx, spec)
		if err != nil {
			t.Fatalf("first Register() failed: %v", err)
		}

		err = reg.Register(ctx, spec)
		if err == nil {
			t.Fatal("Register() expected error for duplicate, got nil")
		}
		if !errors.Is(err, models.ErrConflict) {
			t.Errorf("Register() error = %v, want ErrConflict", err)
		}
	})

	t.Run("validation error", func(t *testing.T) {
		store := storemock.NewAgentStore()
		reg := NewRegistry(store)

		spec := models.AgentSpec{
			AgentID:   "",
			AgentType: models.AgentTypeSystemd,
		}
		err := reg.Register(ctx, spec)
		if err == nil {
			t.Fatal("Register() expected validation error, got nil")
		}
	})

	t.Run("invalid agent type", func(t *testing.T) {
		store := storemock.NewAgentStore()
		reg := NewRegistry(store)

		spec := models.AgentSpec{
			AgentID:   "agent-bad-type",
			AgentType: "invalid",
		}
		err := reg.Register(ctx, spec)
		if err == nil {
			t.Fatal("Register() expected validation error, got nil")
		}
	})
}

func TestRegistryGet(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		store := storemock.NewAgentStore()
		reg := NewRegistry(store)

		spec := models.AgentSpec{
			AgentID:   "agent-get",
			AgentType: models.AgentTypeMCPClient,
		}
		err := store.Create(ctx, spec)
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		got, err := reg.Get(ctx, "agent-get")
		if err != nil {
			t.Fatalf("Get() unexpected error: %v", err)
		}
		if got.AgentID != "agent-get" {
			t.Errorf("Get() AgentID = %q, want %q", got.AgentID, "agent-get")
		}
	})

	t.Run("not found", func(t *testing.T) {
		store := storemock.NewAgentStore()
		reg := NewRegistry(store)

		_, err := reg.Get(ctx, "nonexistent")
		if err == nil {
			t.Fatal("Get() expected error, got nil")
		}
	})
}

func TestRegistryUpdate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		store := storemock.NewAgentStore()
		reg := NewRegistry(store)

		spec := models.AgentSpec{
			AgentID:     "agent-upd",
			AgentType:   models.AgentTypeSystemd,
			DisplayName: "Original",
		}
		err := store.Create(ctx, spec)
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		spec.DisplayName = "Updated"
		err = reg.Update(ctx, "agent-upd", spec)
		if err != nil {
			t.Fatalf("Update() unexpected error: %v", err)
		}

		got, _ := store.Get(ctx, "agent-upd")
		if got.DisplayName != "Updated" {
			t.Errorf("Update() DisplayName = %q, want %q", got.DisplayName, "Updated")
		}
	})

	t.Run("validation error", func(t *testing.T) {
		store := storemock.NewAgentStore()
		reg := NewRegistry(store)

		spec := models.AgentSpec{
			AgentID:   "agent-upd2",
			AgentType: "invalid",
		}
		err := reg.Update(ctx, "agent-upd2", spec)
		if err == nil {
			t.Fatal("Update() expected validation error, got nil")
		}
	})
}

func TestRegistryDelete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		store := storemock.NewAgentStore()
		reg := NewRegistry(store)

		spec := models.AgentSpec{
			AgentID:   "agent-del",
			AgentType: models.AgentTypeSystemd,
		}
		err := store.Create(ctx, spec)
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		err = reg.Delete(ctx, "agent-del")
		if err != nil {
			t.Fatalf("Delete() unexpected error: %v", err)
		}

		_, err = store.Get(ctx, "agent-del")
		if err == nil {
			t.Fatal("Get() expected error after delete, got nil")
		}
	})

	t.Run("not found", func(t *testing.T) {
		store := storemock.NewAgentStore()
		reg := NewRegistry(store)

		err := reg.Delete(ctx, "nonexistent")
		if err == nil {
			t.Fatal("Delete() expected error for nonexistent, got nil")
		}
	})
}

func TestRegistryList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("list all", func(t *testing.T) {
		store := storemock.NewAgentStore()
		reg := NewRegistry(store)

		for i := 0; i < 5; i++ {
			spec := models.AgentSpec{
				AgentID:   fmt.Sprintf("agent-%d", i),
				AgentType: models.AgentTypeSystemd,
				CreatedAt: time.Now(),
			}
			_ = store.Create(ctx, spec)
		}

		agents, err := reg.List(ctx, nil, 100, 0)
		if err != nil {
			t.Fatalf("List() unexpected error: %v", err)
		}
		if len(agents) != 5 {
			t.Errorf("List() returned %d agents, want 5", len(agents))
		}
	})

	t.Run("list with filter", func(t *testing.T) {
		store := storemock.NewAgentStore()
		reg := NewRegistry(store)

		spec1 := models.AgentSpec{AgentID: "sys-agent", AgentType: models.AgentTypeSystemd, CreatedAt: time.Now()}
		spec2 := models.AgentSpec{AgentID: "mcp-agent", AgentType: models.AgentTypeMCPClient, CreatedAt: time.Now()}
		_ = store.Create(ctx, spec1)
		_ = store.Create(ctx, spec2)

		filter := map[string]string{"agent_type": "systemd"}
		agents, err := reg.List(ctx, filter, 100, 0)
		if err != nil {
			t.Fatalf("List() unexpected error: %v", err)
		}
		if len(agents) != 1 {
			t.Errorf("List() returned %d agents, want 1", len(agents))
		}
		if agents[0].AgentID != "sys-agent" {
			t.Errorf("List() AgentID = %q, want %q", agents[0].AgentID, "sys-agent")
		}
	})

	t.Run("list with pagination", func(t *testing.T) {
		store := storemock.NewAgentStore()
		reg := NewRegistry(store)

		for i := 0; i < 10; i++ {
			spec := models.AgentSpec{
				AgentID:   fmt.Sprintf("agent-%d", i),
				AgentType: models.AgentTypeSystemd,
				CreatedAt: time.Now(),
			}
			_ = store.Create(ctx, spec)
		}

		agents, err := reg.List(ctx, nil, 3, 5)
		if err != nil {
			t.Fatalf("List() unexpected error: %v", err)
		}
		if len(agents) != 3 {
			t.Errorf("List() returned %d agents, want 3", len(agents))
		}
	})
}

func TestRegistryHeartbeat(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		store := storemock.NewAgentStore()
		reg := NewRegistry(store)

		spec := models.AgentSpec{
			AgentID:   "agent-hb",
			AgentType: models.AgentTypeSystemd,
		}
		err := store.Create(ctx, spec)
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		ts, err := reg.Heartbeat(ctx, "agent-hb")
		if err != nil {
			t.Fatalf("Heartbeat() unexpected error: %v", err)
		}
		if ts.IsZero() {
			t.Error("Heartbeat() returned zero timestamp")
		}

		// Verify the heartbeat was stored
		got, _ := store.Get(ctx, "agent-hb")
		if got.LastHeartbeat.IsZero() {
			t.Error("Heartbeat() did not update LastHeartbeat")
		}
	})

	t.Run("not found", func(t *testing.T) {
		store := storemock.NewAgentStore()
		reg := NewRegistry(store)

		_, err := reg.Heartbeat(ctx, "nonexistent")
		if err == nil {
			t.Fatal("Heartbeat() expected error for nonexistent, got nil")
		}
	})
}


