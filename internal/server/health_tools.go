package server

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/service"
)

// registerHealthTools registers the health check tool.
func registerHealthTools(mcpSrv *server.MCPServer, registry *service.Registry) {
	mcpSrv.AddTool(mcp.NewTool("health.check",
		mcp.WithDescription("Check server health: PG ping + agent count"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Use registry to get actual agent count; short timeout for safety
		hcCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		count, countErr := registry.AgentCount(hcCtx)

		status := "ok"
		if countErr != nil {
			status = "degraded"
		}

		result := map[string]any{
			"status":      status,
			"agent_count": count,
			"version":     "0.1.0",
		}

		return marshalResult(result)
	})
}
