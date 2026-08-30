package server

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/service"
)

func registerHealthResource(mcpServer *server.MCPServer, registry *service.Registry) {
	mcpServer.AddResource(mcp.NewResource(
		"alms://health",
		"Server Health",
		mcp.WithResourceDescription("ALMS server health status"),
		mcp.WithMIMEType("application/json"),
	), func(ctx context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		agentCount, err := registry.AgentCount(ctx)
		if err != nil {
			agentCount = -1
		}

		health := map[string]any{
			"status":      "ok",
			"version":     "0.1.0",
			"agent_count": agentCount,
		}

		return newJSONResourceContents("alms://health", health, "marshal health")
	})
}
