package server

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/models"
	"github.com/ghassan/alms/internal/service"
)

func registerAgentUpdateTool(mcpSrv *server.MCPServer, registry *service.Registry) {
	mcpSrv.AddTool(mcp.NewTool("agent.update",
		mcp.WithDescription("Update an existing agent"),
		mcp.WithString("agent_id", mcp.Required(), mcp.Description("Agent identifier")),
		mcp.WithString("agent_type", mcp.Description("systemd or mcp_client")),
		mcp.WithString("display_name", mcp.Description("Human-readable name")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		agentID, _ := args["agent_id"].(string)
		agentTypeStr, _ := args["agent_type"].(string)
		displayName, _ := args["display_name"].(string)

		existing, err := registry.Get(ctx, agentID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("agent %s not found: %v", agentID, err)), nil
		}

		if displayName != "" {
			existing.DisplayName = displayName
		}
		if agentTypeStr != "" {
			existing.AgentType = models.AgentType(agentTypeStr)
		}

		if err := registry.Update(ctx, agentID, existing); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return marshalResult(existing)
	})
}
