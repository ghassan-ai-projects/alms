package server

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/models"
	"github.com/ghassan/alms/internal/service"
)

// registerProtocolManagementTools registers protocol creation and listing tools.
func registerProtocolManagementTools(mcpSrv *server.MCPServer, learning *service.Learning) {
	mcpSrv.AddTool(mcp.NewTool("protocol.push",
		mcp.WithDescription("Create a new protocol"),
		mcp.WithString("title", mcp.Required(), mcp.Description("Protocol title")),
		mcp.WithString("body", mcp.Description("Protocol body/content")),
		mcp.WithArray("target_tags",
			mcp.Description("Tags targeting which agents should receive this protocol"),
			mcp.Items(map[string]any{"type": "string"}),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		title := req.GetString("title", "")
		body := req.GetString("body", "")
		targetTags := req.GetStringSlice("target_tags", nil)

		if title == "" {
			return mcp.NewToolResultError("title is required"), nil
		}

		rec := models.ProtocolRecord{
			Title:      title,
			Body:       body,
			TargetTags: targetTags,
			IsActive:   true,
			Version:    1,
		}

		id, err := learning.ProtocolPush(ctx, rec)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return marshalResult(map[string]any{
			"protocol_id": id,
			"status":      "created",
		})
	})

	mcpSrv.AddTool(mcp.NewTool("protocol.list",
		mcp.WithDescription("List all protocols"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		protocols, err := learning.ProtocolList(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if protocols == nil {
			protocols = []models.ProtocolRecord{}
		}

		return marshalResult(protocols)
	})
}
