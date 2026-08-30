package server

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/models"
	"github.com/ghassan/alms/internal/service"
)

func registerLearningSyncTools(mcpSrv *server.MCPServer, syncer *service.Syncer) {
	mcpSrv.AddTool(mcp.NewTool("learning.sync",
		mcp.WithDescription("Sync new learnings for an agent"),
		mcp.WithString("agent_id", mcp.Required(), mcp.Description("Agent identifier")),
		mcp.WithString("since", mcp.Description("RFC3339 timestamp to sync from")),
		mcp.WithString("type", mcp.Description("Optional learning type filter")),
		mcp.WithArray("tags",
			mcp.Description("Optional tag filters"),
			mcp.Items(map[string]any{"type": "string"}),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		agentID := req.GetString("agent_id", "")
		sinceStr := req.GetString("since", "")
		ltype := req.GetString("type", "")
		tags := req.GetStringSlice("tags", nil)

		var since time.Time
		if sinceStr != "" {
			var err error
			since, err = time.Parse(time.RFC3339, sinceStr)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid since timestamp: %v", err)), nil
			}
		} else {
			since = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		}

		records, err := syncer.Sync(ctx, agentID, since, ltype, tags)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if records == nil {
			records = []models.LearningRecord{}
		}

		return marshalResult(records)
	})

	mcpSrv.AddTool(mcp.NewTool("learning.sync_ack",
		mcp.WithDescription("Acknowledge received learnings (gap-safe)"),
		mcp.WithString("agent_id", mcp.Required(), mcp.Description("Agent identifier")),
		mcp.WithArray("learning_ids",
			mcp.Required(),
			mcp.Description("Ordered list of acknowledged learning IDs"),
			mcp.Items(map[string]any{"type": "string"}),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		agentID := req.GetString("agent_id", "")
		ids := req.GetStringSlice("learning_ids", nil)

		if len(ids) == 0 {
			return mcp.NewToolResultError("learning_ids is required"), nil
		}

		if err := syncer.SyncAck(ctx, agentID, ids); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return marshalResult(map[string]any{
			"status": "acknowledged",
			"count":  len(ids),
		})
	})
}
