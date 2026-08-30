package server

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/models"
	"github.com/ghassan/alms/internal/service"
)

func registerProtocolSyncTools(mcpSrv *server.MCPServer, syncer *service.Syncer) {
	mcpSrv.AddTool(mcp.NewTool("protocol.pull",
		mcp.WithDescription("Pull active protocols for an agent"),
		mcp.WithString("agent_id", mcp.Required(), mcp.Description("Agent identifier")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		agentID, _ := args["agent_id"].(string)

		protocols, err := syncer.PullProtocols(ctx, agentID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if protocols == nil {
			protocols = []models.ProtocolRecord{}
		}

		return marshalResult(protocols)
	})

	mcpSrv.AddTool(mcp.NewTool("protocol.pull_since",
		mcp.WithDescription("Pull protocols created after a given protocol ID"),
		mcp.WithString("agent_id", mcp.Required(), mcp.Description("Agent identifier")),
		mcp.WithString("since_id", mcp.Required(), mcp.Description("Protocol ID to pull since")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		agentID, _ := args["agent_id"].(string)
		sinceID, _ := args["since_id"].(string)

		protocols, err := syncer.PullProtocolsSince(ctx, agentID, sinceID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if protocols == nil {
			protocols = []models.ProtocolRecord{}
		}

		return marshalResult(protocols)
	})
}
