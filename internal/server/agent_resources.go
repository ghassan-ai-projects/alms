package server

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/service"
)

func registerAgentsResource(mcpServer *server.MCPServer, registry *service.Registry) {
	mcpServer.AddResource(mcp.NewResource(
		"alms://agents",
		"All Agents",
		mcp.WithResourceDescription("List of all registered agents"),
		mcp.WithMIMEType("application/json"),
	), func(ctx context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		agents, err := registry.List(ctx, nil, 100, 0)
		if err != nil {
			return nil, fmt.Errorf("list agents: %w", err)
		}

		return newJSONResourceContents("alms://agents", agents, "marshal agents")
	})
}
