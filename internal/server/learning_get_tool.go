package server

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/service"
)

func registerLearningGetTool(mcpSrv *server.MCPServer, learning *service.Learning) {
	mcpSrv.AddTool(mcp.NewTool("learning.get",
		mcp.WithDescription("Get a single learning record by ID"),
		mcp.WithString("learning_id", mcp.Required(), mcp.Description("Learning ID to retrieve")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		learningID := req.GetString("learning_id", "")
		if learningID == "" {
			return mcp.NewToolResultError("learning_id is required"), nil
		}

		record, err := learning.Get(ctx, learningID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return marshalResult(record)
	})
}
