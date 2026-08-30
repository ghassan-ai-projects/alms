// Package server provides the MCP server implementation using mark3labs/mcp-go.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/config"
	"github.com/ghassan/alms/internal/service"
)

// Server wraps the MCP server with lifecycle management.
type Server struct {
	mcp     *server.MCPServer
	httpSrv *http.Server
	cfg     *config.Config
}

// New creates a new ALMS server.
func New(cfg *config.Config, registry *service.Registry, syncer *service.Syncer, learning *service.Learning) *Server {
	mcpSrv := server.NewMCPServer(
		"ALMS",
		"0.1.0",
		server.WithInstructions("Agent Learning Management System"),
	)

	s := &Server{
		mcp: mcpSrv,
		cfg: cfg,
	}

	s.registerTools(registry, syncer, learning)
	s.registerResources(registry, learning)

	return s
}

// ListenAndServe starts the HTTP server and blocks until shutdown.
func (s *Server) ListenAndServe(ctx context.Context) error {
	handler := s.buildHTTPHandler()
	s.httpSrv = newHTTPServer(s.cfg.Server.Addr(), handler)

	slog.Info("starting server", "addr", s.cfg.Server.Addr())

	if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the server with the given context.
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("shutting down server")
	if s.httpSrv != nil {
		if err := s.httpSrv.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}
	}
	return nil
}
