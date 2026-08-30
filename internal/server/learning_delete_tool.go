package server

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/service"
)

func registerLearningDeleteTool(mcpSrv *server.MCPServer, learning *service.Learning) {
	mcpSrv.AddTool(mcp.NewTool("learning.delete",
		mcp.WithDescription("Soft-delete a learning record"),
		mcp.WithString("learning_id", mcp.Required(), mcp.Description("Learning ID to delete")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		learningID := req.GetString("learning_id", "")
		if learningID == "" {
			return mcp.NewToolResultError("learning_id is required"), nil
		}

		if err := learning.Delete(ctx, learningID); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return marshalResult(map[string]any{
			"deleted": learningID,
			"status":  "soft_deleted",
		})
	})
}
