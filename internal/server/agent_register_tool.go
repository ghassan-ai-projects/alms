package server

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/models"
	"github.com/ghassan/alms/internal/service"
)

func registerAgentRegisterTool(mcpSrv *server.MCPServer, registry *service.Registry) {
	mcpSrv.AddTool(mcp.NewTool("agent.register",
		mcp.WithDescription("Register a new agent"),
		mcp.WithString("agent_id", mcp.Required(), mcp.Description("Unique agent identifier")),
		mcp.WithString("agent_type", mcp.Required(), mcp.Description("systemd or mcp_client")),
		mcp.WithString("display_name", mcp.Description("Human-readable name")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		agentID, _ := args["agent_id"].(string)
		agentTypeStr, _ := args["agent_type"].(string)
		displayName, _ := args["display_name"].(string)

		spec := models.AgentSpec{
			AgentID:     agentID,
			AgentType:   models.AgentType(agentTypeStr),
			DisplayName: displayName,
		}

		if err := registry.Register(ctx, spec); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return marshalResult(spec)
	})
}
