package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ghassan/alms/internal/config"
	"github.com/ghassan/alms/internal/server"
	"github.com/ghassan/alms/internal/service"
	"github.com/ghassan/alms/internal/store"
)

// Version and Commit are set via ldflags at build time.
var (
	Version = "dev"
	Commit  = "none"
)

func main() {
	cfgPath := flag.String("config", "", "Path to config file")
	runMigrate := flag.Bool("migrate", false, "Run database migrations and exit")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("alms %s (%s)\n", Version, Commit)
		os.Exit(0)
	}

	cfg := config.Load(*cfgPath)

	// Signal-aware context for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Connect to PostgreSQL
	pool, err := store.NewPool(ctx, cfg.Database.DSN)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("connected to database")

	if *runMigrate {
		slog.Info("migration flag set; run migration tool externally (e.g., migrate CLI)")
		slog.Info("example: migrate -path internal/store/migrations -database \"$ALMS_PG_DSN\" up")
		os.Exit(0)
	}

	// Init stores (concrete implementations)
	agentStore := store.NewAgentStore(pool)
	learningStore := store.NewLearningStore(pool)
	protocolStore := store.NewProtocolStore(pool)

	// Init services (business logic)
	registrySvc := service.NewRegistry(agentStore)
	syncerSvc := service.NewSyncer(learningStore, agentStore, protocolStore)
	learningSvc := service.NewLearning(learningStore, protocolStore)

	// Init GC service for background garbage collection
	gcSvc := service.NewGC(learningStore, service.DefaultGCConfig())
	gcSvc.Start(ctx)
	defer gcSvc.Stop()

	// Init MCP server
	srv := server.New(&cfg, registrySvc, syncerSvc, learningSvc)

	// Start server in background goroutine
	go func() {
		slog.Info("starting ALMS server", "addr", cfg.Server.Addr())
		if err := srv.ListenAndServe(ctx); err != nil {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()
	slog.Info("shutting down...")

	// Graceful shutdown with 10s timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped gracefully")
}
