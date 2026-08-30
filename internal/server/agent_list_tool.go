package server

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/models"
	"github.com/ghassan/alms/internal/service"
)

func registerAgentListTool(mcpSrv *server.MCPServer, registry *service.Registry) {
	mcpSrv.AddTool(mcp.NewTool("agent.list",
		mcp.WithDescription("List registered agents"),
		mcp.WithString("agent_type", mcp.Description("Optional type filter")),
		mcp.WithNumber("limit", mcp.Description("Max results (default 100)")),
		mcp.WithNumber("offset", mcp.Description("Result offset (default 0)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		agentType := req.GetString("agent_type", "")
		limit := req.GetInt("limit", 100)
		offset := req.GetInt("offset", 0)

		filter := make(map[string]string)
		if agentType != "" {
			filter["agent_type"] = agentType
		}

		agents, err := registry.List(ctx, filter, limit, offset)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if agents == nil {
			agents = []models.AgentSpec{}
		}

		return marshalResult(agents)
	})
}
