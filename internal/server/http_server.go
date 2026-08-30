package server

import (
	"net/http"
	"time"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

func (s *Server) buildHTTPHandler() http.Handler {
	streamableHTTPServer := mcpserver.NewStreamableHTTPServer(s.mcp)
	var mcpHandler http.Handler = streamableHTTPServer

	if s.cfg.Auth.Token != "" {
		mcpHandler = AuthMiddleware(s.cfg.Auth.Token)(mcpHandler)
	}

	mux := http.NewServeMux()
	mux.Handle("/dashboard", DashboardHandler())
	mux.Handle("/", mcpHandler)
	return mux
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}
