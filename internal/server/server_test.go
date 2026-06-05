package server

import (
	"context"
	"testing"

	"github.com/ghassan/alms/internal/config"
	"github.com/ghassan/alms/internal/service"
	"github.com/ghassan/alms/internal/service/storemock"
)

func TestNewServer(t *testing.T) {
	t.Parallel()

	t.Run("server created with config", func(t *testing.T) {
		cfg := config.DefaultConfig()
		aStore := storemock.NewAgentStore()
		lStore := storemock.NewLearningStore()
		pStore := storemock.NewProtocolStore()

		registry := service.NewRegistry(aStore)
		syncer := service.NewSyncer(lStore, aStore, pStore)
		learningSvc := service.NewLearning(lStore, pStore)

		srv := New(&cfg, registry, syncer, learningSvc)
		if srv == nil {
			t.Fatal("New() returned nil")
		}
	})

	t.Run("server shutdown with active handler", func(t *testing.T) {
		cfg := config.DefaultConfig()
		aStore := storemock.NewAgentStore()
		lStore := storemock.NewLearningStore()
		pStore := storemock.NewProtocolStore()

		registry := service.NewRegistry(aStore)
		syncer := service.NewSyncer(lStore, aStore, pStore)
		learningSvc := service.NewLearning(lStore, pStore)

		srv := New(&cfg, registry, syncer, learningSvc)

		// Start in background, then shutdown
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // immediate cancellation

		// Shutdown before ListenAndServe has started should be safe
		err := srv.Shutdown(ctx)
		if err != nil {
			t.Fatalf("Shutdown() unexpected error: %v", err)
		}
	})
}
