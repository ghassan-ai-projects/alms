package server

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/service"
)

func registerAgentUnregisterTool(mcpSrv *server.MCPServer, registry *service.Registry) {
	mcpSrv.AddTool(mcp.NewTool("agent.unregister",
		mcp.WithDescription("Unregister an existing agent"),
		mcp.WithString("agent_id", mcp.Required(), mcp.Description("Agent identifier to remove")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		agentID, _ := args["agent_id"].(string)

		spec, err := registry.Get(ctx, agentID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if err := registry.Delete(ctx, agentID); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return marshalResult(map[string]any{
			"deleted": agentID,
			"agent":   spec,
		})
	})
}
