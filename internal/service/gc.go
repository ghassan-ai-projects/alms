// Package service provides business logic for ALMS operations.
package service

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// GCConfig holds configuration for the garbage collector.
type GCConfig struct {
	Enabled  bool          // whether GC runs periodically
	Interval time.Duration // interval between GC sweeps (default 24h)
}

// DefaultGCConfig returns a default GC configuration.
func DefaultGCConfig() GCConfig {
	return GCConfig{
		Enabled:  true,
		Interval: 24 * time.Hour,
	}
}

// GC manages background garbage collection of stale learning records.
type GC struct {
	store   LearningStore
	config  GCConfig
	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// NewGC creates a new GC service backed by the given store.
func NewGC(store LearningStore, config GCConfig) *GC {
	return &GC{
		store:  store,
		config: config,
		stopCh: make(chan struct{}),
	}
}

// Start begins the background GC goroutine. Does nothing if GC is disabled.
func (g *GC) Start(ctx context.Context) {
	if !g.config.Enabled {
		slog.Info("GC is disabled")
		return
	}

	if !g.claimRunning() {
		return
	}

	g.wg.Add(1)
	go g.runBackgroundLoop(ctx)
}

func (g *GC) claimRunning() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.running {
		return false
	}
	g.running = true
	return true
}

func (g *GC) runBackgroundLoop(ctx context.Context) {
	defer g.wg.Done()
	slog.Info("GC started", "interval", g.config.Interval)

	ticker := time.NewTicker(g.config.Interval)
	defer ticker.Stop()

	// Run an initial sweep
	g.runSweep(ctx)

	for {
		select {
		case <-ticker.C:
			g.runSweep(ctx)
		case <-g.stopCh:
			slog.Info("GC stopped")
			return
		case <-ctx.Done():
			slog.Info("GC stopped by context")
			return
		}
	}
}

// Stop signals the GC goroutine to stop and waits for it to finish.
func (g *GC) Stop() {
	g.mu.Lock()
	if !g.running {
		g.mu.Unlock()
		return
	}
	g.mu.Unlock()

	close(g.stopCh)
	g.wg.Wait()

	g.mu.Lock()
	g.running = false
	g.mu.Unlock()
}
