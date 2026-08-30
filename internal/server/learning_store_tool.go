package server

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ghassan/alms/internal/models"
	"github.com/ghassan/alms/internal/service"
)

func registerLearningStoreTool(mcpSrv *server.MCPServer, learning *service.Learning) {
	mcpSrv.AddTool(mcp.NewTool("learning.store",
		mcp.WithDescription("Create a new learning record with dedup check"),
		mcp.WithString("agent_id", mcp.Required(), mcp.Description("Source agent identifier")),
		mcp.WithString("title", mcp.Required(), mcp.Description("Learning title")),
		mcp.WithString("body", mcp.Description("Learning body/content")),
		mcp.WithString("type", mcp.Required(), mcp.Description("Learning type: pattern, failure, config, protocol, edge_case")),
		mcp.WithArray("tags",
			mcp.Description("Tags for categorization"),
			mcp.Items(map[string]any{"type": "string"}),
		),
		mcp.WithString("supersedes", mcp.Description("Optional learning ID this supersedes")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		agentID := req.GetString("agent_id", "")
		title := req.GetString("title", "")
		body := req.GetString("body", "")
		ltype := req.GetString("type", "")
		tags := req.GetStringSlice("tags", nil)
		supersedes := req.GetString("supersedes", "")

		if agentID == "" {
			return mcp.NewToolResultError("agent_id is required"), nil
		}

		rec := models.LearningRecord{
			Title:      title,
			Body:       body,
			Type:       models.LearningType(ltype),
			Tags:       tags,
			Author:     agentID,
			SrcAgentID: agentID,
		}

		// Use dedup-aware store to check for exact and near duplicates
		id, dedupResult, err := learning.StoreLearningWithDedup(ctx, rec, supersedes)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Fetch stored record to get enrichment metadata
		stored, _ := learning.Get(ctx, id)
		enrichment := models.DefaultEnrichmentMetadata()
		if len(stored.EnrichmentMetadata) > 0 {
			enrichment = stored.EnrichmentMetadata
		}

		return marshalResult(map[string]any{
			"learning_id":  id,
			"status":       "created",
			"is_duplicate": dedupResult.IsExactDup || dedupResult.IsNearDup,
			"enrichment":   enrichment,
		})
	})
}
