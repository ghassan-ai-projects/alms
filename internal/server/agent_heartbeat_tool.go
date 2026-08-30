package server

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/service"
)

func registerAgentHeartbeatTool(mcpSrv *server.MCPServer, registry *service.Registry) {
	mcpSrv.AddTool(mcp.NewTool("agent.heartbeat",
		mcp.WithDescription("Send a heartbeat for an agent"),
		mcp.WithString("agent_id", mcp.Required(), mcp.Description("Agent identifier")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		agentID, _ := args["agent_id"].(string)

		ts, err := registry.Heartbeat(ctx, agentID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return marshalResult(map[string]any{
			"agent_id":       agentID,
			"last_heartbeat": ts.Format(time.RFC3339),
		})
	})
}
