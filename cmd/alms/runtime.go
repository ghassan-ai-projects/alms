package main

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ghassan/alms/internal/service"
	"github.com/ghassan/alms/internal/store"
)

type runtimeServices struct {
	registry *service.Registry
	syncer   *service.Syncer
	learning *service.Learning
	gc       *service.GC
}

func buildRuntime(pool *pgxpool.Pool) runtimeServices {
	agentStore := store.NewAgentStore(pool)
	learningStore := store.NewLearningStore(pool)
	protocolStore := store.NewProtocolStore(pool)

	registry := service.NewRegistry(agentStore)
	syncer := service.NewSyncer(learningStore, agentStore, protocolStore)
	learning := service.NewLearning(learningStore, protocolStore)
	gc := service.NewGC(learningStore, service.DefaultGCConfig())

	return runtimeServices{
		registry: registry,
		syncer:   syncer,
		learning: learning,
		gc:       gc,
	}
}
